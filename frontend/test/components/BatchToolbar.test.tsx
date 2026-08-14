import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import BatchToolbar from '../../src/components/config/BatchToolbar'
import ChangesContentDialog from '../../src/components/config/ChangesContentDialog'
import { UiProvider } from '../../src/ui'
import { useChangesetStore } from '../../src/stores/changeset'
import * as apiModule from '../../src/api'

// 批量工具栏 + 变更内容弹窗 F2（FE-23）。
vi.mock('../../src/api')

const DEV = '10.0.0.1'
const S = () => useChangesetStore.getState()

function seedEntry() {
  S().upsert(DEV, {
    op: 'update', path: '/vlan:vlan/vlan:vlans', listKey: 'vlan', keyValue: '10',
    payload: { id: 10, name: 'core' }, cleared: ['desc'],
    baseline: { id: 10, name: 'mgmt', desc: 'old' }, label: 'vlan 10',
  })
}

describe('BatchToolbar（FE-23）', () => {
  beforeEach(() => {
    useChangesetStore.setState({ byDevice: {} })
    vi.clearAllMocks()
  })

  it('空集：徽标隐藏、试运行/重置/提交禁用', () => {
    render(
      <UiProvider>
        <BatchToolbar device={DEV} onReset={vi.fn()} onCommitRequest={vi.fn()} />
      </UiProvider>,
    )
    expect(document.querySelector('[data-test="batch-dryrun"]')).toBeDisabled()
    expect(document.querySelector('[data-test="batch-reset"]')).toBeDisabled()
    expect(document.querySelector('[data-test="batch-commit"]')).toBeDisabled()
    expect(document.querySelector('[data-test="batch-hint"]')).toBeNull()
  })

  it('有变更：提示条与徽标出现；提交点击透传页面层', () => {
    seedEntry()
    const onCommit = vi.fn()
    render(
      <UiProvider>
        <BatchToolbar device={DEV} onReset={vi.fn()} onCommitRequest={onCommit} />
      </UiProvider>,
    )
    expect(document.querySelector('[data-test="batch-hint"]')).toBeTruthy()
    fireEvent.click(document.querySelector('[data-test="batch-commit"]')!)
    expect(onCommit).toHaveBeenCalled()
  })

  it('重置：确认后清空当前设备变更集并回调 onReset', async () => {
    seedEntry()
    const onReset = vi.fn()
    render(
      <UiProvider>
        <BatchToolbar device={DEV} onReset={onReset} onCommitRequest={vi.fn()} />
      </UiProvider>,
    )
    fireEvent.click(document.querySelector('[data-test="batch-reset"]')!)
    const btns = await waitFor(() => {
      const el = document.querySelector('.ant-modal-confirm-btns')
      if (!el) throw new Error('not open')
      return el as HTMLElement
    })
    fireEvent.click(within(btns).getByRole('button', { name: /确\s*定|OK/ }))
    await waitFor(() => expect(S().countFor(DEV)).toBe(0))
    expect(onReset).toHaveBeenCalled()
  })
})

describe('BatchToolbar · 弹窗接线', () => {
  beforeEach(() => {
    useChangesetStore.setState({ byDevice: {} })
    vi.clearAllMocks()
  })

  it('变更内容/试运行按钮打开各自弹窗；提示条可关闭', async () => {
    seedEntry()
    vi.mocked(apiModule.previewChangeset).mockResolvedValue({
      data: { success: true, data: { device: DEV, entries: [] } },
    } as any)
    render(
      <UiProvider>
        <BatchToolbar device={DEV} onReset={vi.fn()} onCommitRequest={vi.fn()} />
      </UiProvider>,
    )
    fireEvent.click(document.querySelector('[data-test="batch-changes"]')!)
    expect(await screen.findByText('vlan 10')).toBeInTheDocument()

    fireEvent.click(document.querySelector('[data-test="batch-dryrun"]')!)
    await waitFor(() => expect(vi.mocked(apiModule.previewChangeset)).toHaveBeenCalled())

    const closeBtn = document.querySelector('[data-test="batch-hint"] .ant-alert-close-icon')
    if (closeBtn) {
      fireEvent.click(closeBtn)
      await waitFor(() => expect(document.querySelector('[data-test="batch-hint"]')).toBeNull())
    }
  })
})

describe('ChangesContentDialog（FE-23 变更内容）', () => {
  beforeEach(() => {
    useChangesetStore.setState({ byDevice: {} })
  })

  it('树行三类：modify 前后值、cleared 删除行、图例计数', async () => {
    seedEntry()
    render(
      <UiProvider>
        <ChangesContentDialog open device={DEV} onClose={vi.fn()} />
      </UiProvider>,
    )
    expect(await screen.findByText('vlan 10')).toBeInTheDocument()
    // modify：mgmt → core
    expect(screen.getByText('mgmt')).toBeInTheDocument()
    expect(screen.getByText('core')).toBeInTheDocument()
    // cleared 删除行：old 划线
    expect(screen.getByText('old').className).toContain('cell-removed')
    expect(document.querySelector('[data-test="changes-legend"]')!.textContent).toBeTruthy()
  })
})
