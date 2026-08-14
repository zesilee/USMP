import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import DryRunDialog from '../../src/components/config/DryRunDialog'
import BatchCommitDialog from '../../src/components/config/BatchCommitDialog'
import XmlViewer from '../../src/components/config/XmlViewer'
import { UiProvider } from '../../src/ui'
import { useChangesetStore } from '../../src/stores/changeset'
import * as apiModule from '../../src/api'

// 试运行/提交进度弹窗 + XmlViewer F2（FE-23/FE-03/CS-01/CS-03）。
vi.mock('../../src/api')

const DEV = '10.0.0.1'
const S = () => useChangesetStore.getState()

function seed() {
  S().upsert(DEV, {
    op: 'update', path: '/vlan:vlan/vlan:vlans', listKey: 'vlan', keyValue: '10',
    payload: { id: 10, name: 'x' }, baseline: null, label: 'vlan 10',
  })
}

describe('DryRunDialog（CS-01 纯计算不下发）', () => {
  beforeEach(() => {
    useChangesetStore.setState({ byDevice: {} })
    vi.clearAllMocks()
  })

  it('成功：双栏正向/回滚报文 + 无 XML 条目降级 + diff Tab 基线标注', async () => {
    seed()
    vi.mocked(apiModule.previewChangeset).mockResolvedValue({
      data: {
        success: true,
        data: {
          device: DEV,
          entries: [
            {
              path: '/vlan:vlan/vlan:vlans',
              forward_xml: '<vlan><id>10</id></vlan>',
              rollback_xml: '<vlan/>',
              baseline_source: 'cache',
              diff: [{ path: 'vlan[10]/name', type: 'MODIFY', old: 'a', new: 'x' }],
            },
            { path: '/x', unsupported: true, unsupported_reason: 'no xml channel', baseline_source: 'none' },
          ],
        },
      },
    } as any)
    render(
      <UiProvider>
        <DryRunDialog open device={DEV} onClose={vi.fn()} />
      </UiProvider>,
    )
    await waitFor(() => expect(document.querySelectorAll('[data-test="xml-viewer"]').length).toBe(2))
    expect(screen.getByText(/no xml channel/)).toBeInTheDocument() // CS-03 如实降级
    expect(vi.mocked(apiModule.previewChangeset)).toHaveBeenCalledTimes(1)

    // diff Tab：基线来源标注 + 三列差异行
    const { fireEvent } = await import('@testing-library/react')
    fireEvent.click(screen.getByRole('tab', { name: /差异|Diff|对比/i }))
    await waitFor(() => expect(document.querySelector('[data-test="dryrun-diff"]')).toBeTruthy())
    expect(screen.getByText('vlan[10]/name')).toBeInTheDocument()
  })

  it('preview 失败：如实报错且不影响变更集（R08/§9）', async () => {
    seed()
    vi.mocked(apiModule.previewChangeset).mockResolvedValue({
      data: { success: false, message: 'preview boom' },
    } as any)
    render(
      <UiProvider>
        <DryRunDialog open device={DEV} onClose={vi.fn()} />
      </UiProvider>,
    )
    await waitFor(() => expect(document.querySelector('[data-test="dryrun-error"]')).toBeTruthy())
    expect(S().countFor(DEV)).toBe(1)
  })
})

describe('BatchCommitDialog（FE-03 提交进度）', () => {
  beforeEach(() => {
    useChangesetStore.setState({ byDevice: {} })
    vi.clearAllMocks()
  })

  it('打开即执行：成功链步骤条推进至终局、committed 回调、完成后可关', async () => {
    seed()
    vi.mocked(apiModule.commitChangeset).mockResolvedValue({ data: { code: 0, success: true } } as any)
    vi.mocked(apiModule.getConfig).mockResolvedValue({ data: { data: {} } } as any)
    vi.mocked(apiModule.getDeviceReconcile)
      .mockResolvedValueOnce({ data: { data: { statuses: [] } } } as any)
      .mockResolvedValue({
        data: { data: { statuses: [{ path: 'vlan:vlan/vlan:vlans', last_run: '2026-08-14T12:00:00Z', outcome: 'converged' }] } },
      } as any)
    const onCommitted = vi.fn()
    render(
      <UiProvider>
        <BatchCommitDialog open device={DEV} onClose={vi.fn()} onCommitted={onCommitted} />
      </UiProvider>,
    )
    await waitFor(() => expect(onCommitted).toHaveBeenCalled(), { timeout: 8000 })
    expect(document.querySelector('[data-test="reconcile-steps"]')).toBeTruthy()
    await waitFor(() => expect(document.querySelector('[data-test="commit-close"]')).not.toBeDisabled())
    expect(document.querySelector('[data-test="recon-result"]')).toBeTruthy()
  })

  it('提交失败：错误条展示、变更集保留、可关闭', async () => {
    seed()
    vi.mocked(apiModule.getDeviceReconcile).mockResolvedValue({ data: { data: { statuses: [] } } } as any)
    vi.mocked(apiModule.commitChangeset).mockResolvedValue({
      data: { code: 502, success: false, message: 'commit boom' },
    } as any)
    render(
      <UiProvider>
        <BatchCommitDialog open device={DEV} onClose={vi.fn()} onCommitted={vi.fn()} />
      </UiProvider>,
    )
    await waitFor(() => expect(document.querySelector('[data-test="commit-error"]')).toBeTruthy())
    expect(S().countFor(DEV)).toBe(1)
    expect(document.querySelector('[data-test="commit-close"]')).not.toBeDisabled()
  })
})

describe('ownershipGate（BR-11 结构化判定）', () => {
  it('409+intents 判定命中；其余形态 null', async () => {
    const { ownershipRejectionOf, confirmOwnershipOverride } = await import('../../src/composables/ownershipGate')
    expect(ownershipRejectionOf({ data: { code: 409, message: 'm', data: { intents: ['a/b'] } } })).toEqual({ intents: ['a/b'], message: 'm' })
    expect(ownershipRejectionOf({ data: { code: 409, data: {} } })).toBeNull()
    expect(ownershipRejectionOf({ data: { code: 200 } })).toBeNull()
    // confirm 弹出（危险样式确认框）——取消路径
    const p = confirmOwnershipOverride({ intents: ['a/b'], message: '' })
    const btns = await waitFor(() => {
      const el = document.querySelector('.ant-modal-confirm-btns')
      if (!el) throw new Error('not open')
      return el as HTMLElement
    })
    const { fireEvent } = await import('@testing-library/react')
    const { within } = await import('@testing-library/react')
    fireEvent.click(within(btns).getByRole('button', { name: /取\s*消|Cancel/ }))
    await expect(p).resolves.toBe(false)
  })
})

describe('XmlViewer（展示层零改动/XSS 免疫）', () => {
  it('着色分行渲染；空报文空态', () => {
    const { container, rerender } = render(
      <UiProvider>
        <XmlViewer xml={'<a x="1"><b>text</b></a>'} />
      </UiProvider>,
    )
    expect(container.querySelectorAll('.xml-line').length).toBeGreaterThan(0)
    expect(container.querySelector('.tk-tag')).toBeTruthy()

    rerender(
      <UiProvider>
        <XmlViewer xml="" />
      </UiProvider>,
    )
    expect(container.querySelector('.xml-empty')).toBeTruthy()
  })
})
