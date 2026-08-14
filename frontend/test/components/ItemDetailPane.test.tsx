import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import ItemDetailPane from '../../src/components/config/ItemDetailPane'
import { UiProvider } from '../../src/ui'
import { deriveTabs } from '../../src/utils/moduleConsole'
import { useChangesetStore } from '../../src/stores/changeset'
import * as apiModule from '../../src/api'
import type { Field } from '../../src/utils/crdSchemaParser'

// ItemDetailPane F2（FE-21/FE-22）：编辑/创建种子、变更集回填保持首次基线、
// 字段级清除双语义、暂存 upsert 契约、编辑态 key 禁用、include_state 状态合并。
vi.mock('../../src/api')

const fields: Field[] = [
  {
    path: '/vlan/vlans', type: 'group', label: 'vlans',
    fields: [
      {
        path: '/vlan/vlans/vlan', type: 'list', label: 'vlan',
        fields: [
          { path: '/vlan/vlans/vlan/id', type: 'number', label: 'id', isKey: true },
          { path: '/vlan/vlans/vlan/name', type: 'string', label: 'name' },
          { path: '/vlan/vlans/vlan/desc', type: 'string', label: 'desc' },
          { path: '/vlan/vlans/vlan/status', type: 'string', label: 'status', readonly: true },
        ],
      },
    ],
  },
]

const S = () => useChangesetStore.getState()

function mountPane(mode: 'edit' | 'create', row: Record<string, any> | null, onStaged = vi.fn()) {
  const tab = deriveTabs(fields)[0]
  const utils = render(
    <UiProvider>
      <ItemDetailPane
        tab={tab}
        rootName="vlan"
        device="10.0.0.1"
        mode={mode}
        row={row}
        onClose={vi.fn()}
        onStaged={onStaged}
      />
    </UiProvider>,
  )
  return { ...utils, onStaged }
}

describe('ItemDetailPane · 编辑/暂存链路（FE-21）', () => {
  beforeEach(() => {
    useChangesetStore.setState({ byDevice: {} })
    vi.mocked(apiModule.getConfig).mockResolvedValue({ data: { success: true, data: {} } } as any)
  })

  it('编辑态：行数据回填、key 叶禁用、面包屑=主键值', async () => {
    mountPane('edit', { id: 10, name: 'mgmt' })
    await waitFor(() => expect(screen.getByRole('spinbutton')).toHaveValue('10'))
    expect(screen.getByRole('spinbutton')).toBeDisabled() // 编辑态 key 只读（FE-22）
    expect(screen.getByText('10', { selector: '.ant-breadcrumb *' })).toBeInTheDocument()
  })

  it('改字段 → diff 出现 → 暂存 upsert：payload=主键+改动集、label、path 前导斜杠', async () => {
    const { onStaged } = mountPane('edit', { id: 10, name: 'mgmt', desc: 'old' })
    const nameInput = screen.getAllByRole('textbox')[0]
    fireEvent.change(nameInput, { target: { value: 'core' } })
    await waitFor(() => expect(document.querySelector('[data-test="diff-preview"]')).toBeTruthy())

    fireEvent.click(document.querySelector('[data-test="detail-submit"]')!)
    expect(onStaged).toHaveBeenCalledWith('10')
    const entry = S().entryFor('10.0.0.1', '/vlan:vlan/vlan:vlans', '10')!
    expect(entry.op).toBe('update')
    expect(entry.payload).toEqual({ id: 10, name: 'core' }) // 主键+改动，未动 desc 不入
    expect(entry.listKey).toBe('vlan')
  })

  it('创建态：暂存 op=create；key 可编辑', async () => {
    const { onStaged } = mountPane('create', null)
    const spin = screen.getByRole('spinbutton')
    expect(spin).not.toBeDisabled()
    fireEvent.change(spin, { target: { value: '30' } })
    fireEvent.blur(spin)
    fireEvent.change(screen.getAllByRole('textbox')[0], { target: { value: 'iot' } })
    await waitFor(() => expect(document.querySelector('[data-test="detail-submit"]')).not.toBeDisabled())
    fireEvent.click(document.querySelector('[data-test="detail-submit"]')!)
    expect(onStaged).toHaveBeenCalledWith('30')
    expect(S().entryFor('10.0.0.1', '/vlan:vlan/vlan:vlans', '30')!.op).toBe('create')
  })

  it('变更集回填：payload 覆盖行数据且基线锚定首次快照（回填值呈现为已改动）', async () => {
    S().upsert('10.0.0.1', {
      op: 'update', path: '/vlan:vlan/vlan:vlans', listKey: 'vlan', keyValue: '10',
      payload: { id: 10, name: 'draft' }, baseline: { id: 10, name: 'mgmt' },
    })
    mountPane('edit', { id: 10, name: 'mgmt' })
    await waitFor(() => expect(screen.getAllByRole('textbox')[0]).toHaveValue('draft'))
    // 基线=首次快照 mgmt → draft 呈现为改动（diff 面板在场）
    await waitFor(() => expect(document.querySelector('[data-test="diff-preview"]')).toBeTruthy())
  })

  it('字段级清除：有基线值删除意图 tooltip；点击后键消失、clearedKeys 入暂存（FE-22）', async () => {
    const { onStaged } = mountPane('edit', { id: 10, name: 'mgmt', desc: 'x' })
    await waitFor(() => expect(document.querySelector('[data-test="clear-desc"]')).toBeTruthy())
    fireEvent.click(document.querySelector('[data-test="clear-desc"]')!)
    await waitFor(() => expect(document.querySelector('[data-test="clear-desc"]')).toBeNull()) // 值没了钮也没了
    await waitFor(() => expect(document.querySelector('[data-test="detail-submit"]')).not.toBeDisabled())
    fireEvent.click(document.querySelector('[data-test="detail-submit"]')!)
    expect(onStaged).toHaveBeenCalled()
    expect(S().entryFor('10.0.0.1', '/vlan:vlan/vlan:vlans', '10')!.cleared).toContain('desc')
  })

  it('include_state 单行状态读：readonly 值合入展示、可编辑草稿不被覆盖', async () => {
    vi.mocked(apiModule.getConfig).mockResolvedValue({
      data: { success: true, data: { vlan: [{ id: 10, status: 'up', name: 'DEVICE-NAME' }] } },
    } as any)
    mountPane('edit', { id: 10, name: 'mgmt' })
    await waitFor(() => {
      const boxes = screen.getAllByRole('textbox')
      // status readonly 叶合入（禁用框值 up）；name 保留行值不被设备值覆盖
      expect(boxes.some((b) => (b as HTMLInputElement).value === 'up')).toBe(true)
      expect(boxes.some((b) => (b as HTMLInputElement).value === 'mgmt')).toBe(true)
      expect(boxes.every((b) => (b as HTMLInputElement).value !== 'DEVICE-NAME')).toBe(true)
    })
    expect(vi.mocked(apiModule.getConfig)).toHaveBeenCalledWith(
      '10.0.0.1',
      expect.stringContaining("vlan[id='10']"),
      false,
      true,
    )
  })
})
