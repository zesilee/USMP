import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import ModuleFormTab from '../../src/components/config/ModuleFormTab'
import { UiProvider } from '../../src/ui'
import { deriveTabs } from '../../src/utils/moduleConsole'
import { useChangesetStore } from '../../src/stores/changeset'
import * as apiModule from '../../src/api'
import type { Field } from '../../src/utils/crdSchemaParser'

// ModuleFormTab F2（FE-10/14/24）：回填/暂存改动集/只读 Tab <get> 通道/
// 节点不支持占位与重试逃生/读失败降级。
vi.mock('../../src/api')

const fields: Field[] = [
  {
    path: '/ifm/global', type: 'group', label: 'global',
    fields: [
      { path: '/ifm/global/mtu', type: 'number', label: 'mtu' },
      { path: '/ifm/global/mode', type: 'string', label: 'mode' },
    ],
  },
]

const stateFields: Field[] = [
  {
    path: '/devm/summary', type: 'group', label: 'summary', readonly: true,
    fields: [{ path: '/devm/summary/uptime', type: 'string', label: 'uptime', readonly: true }],
  },
]

const S = () => useChangesetStore.getState()

function mount(fs: Field[], extra: Partial<Parameters<typeof ModuleFormTab>[0]> = {}) {
  const tab = deriveTabs(fs)[0]
  return render(
    <UiProvider>
      <ModuleFormTab tab={tab} rootName="ifm" device="10.0.0.1" {...extra} />
    </UiProvider>,
  )
}

describe('ModuleFormTab · 表单 Tab（FE-10/14/24）', () => {
  beforeEach(() => {
    useChangesetStore.setState({ byDevice: {} })
    vi.clearAllMocks()
  })

  it('取数回填 schema 命中键；改字段暂存 op=update、path 带前导斜杠、只发改动', async () => {
    vi.mocked(apiModule.getConfig).mockResolvedValue({
      data: { success: true, data: { data: { mtu: 1500, mode: 'strict', junk: 'x' } } },
    } as any)
    mount(fields)
    await waitFor(() => expect(screen.getByRole('spinbutton')).toHaveValue('1500'))
    expect(screen.getByRole('textbox')).toHaveValue('strict')

    fireEvent.change(screen.getByRole('spinbutton'), { target: { value: '9000' } })
    fireEvent.blur(screen.getByRole('spinbutton'))
    await waitFor(() => expect(document.querySelector('[data-test="form-stage"]')).not.toBeDisabled())
    fireEvent.click(document.querySelector('[data-test="form-stage"]')!)

    const entry = S().entriesFor('10.0.0.1')[0]
    expect(entry.op).toBe('update')
    expect(entry.path).toBe('/ifm:ifm/ifm:global')
    expect(entry.payload).toEqual({ mtu: 9000 }) // 未动 mode 不入（改动集语义）
    expect(entry.keyValue).toBeUndefined()
  })

  it('只读 Tab：include_state=true 走 <get> 通道、无暂存入口（FE-14）', async () => {
    vi.mocked(apiModule.getConfig).mockResolvedValue({
      data: { success: true, data: { data: { uptime: '3d' } } },
    } as any)
    const tab = deriveTabs(stateFields)[0]
    render(
      <UiProvider>
        <ModuleFormTab tab={tab} rootName="devm" device="10.0.0.1" />
      </UiProvider>,
    )
    await waitFor(() =>
      expect(vi.mocked(apiModule.getConfig)).toHaveBeenCalledWith('10.0.0.1', expect.any(String), false, true),
    )
    expect(document.querySelector('[data-test="form-stage"]')).toBeNull()
  })

  it('节点不支持占位（FE-24）：预标记零请求；重试 force 放行且成功即恢复', async () => {
    vi.mocked(apiModule.getConfig).mockResolvedValue({
      data: { success: true, data: { data: { mtu: 1500 } } },
    } as any)
    mount(fields, { unsupported: true })
    expect(document.querySelector('[data-test="node-unsupported"]')).toBeTruthy()
    expect(vi.mocked(apiModule.getConfig)).not.toHaveBeenCalled() // 占位零请求

    fireEvent.click(screen.getByRole('button'))
    await waitFor(() => expect(document.querySelector('[data-test="module-form-tab"]')).toBeTruthy())
    expect(vi.mocked(apiModule.getConfig)).toHaveBeenCalledWith('10.0.0.1', expect.any(String), true, false)
  })

  it('运行中学习：信封 node-unsupported 语义转占位（不弹裸错误）', async () => {
    // 后端契约（BR-12）：HTTP 200 统一信封，结构化 reason 字段（禁文案匹配）。
    vi.mocked(apiModule.getConfig).mockResolvedValue({
      data: { success: false, code: 1, message: 'x', data: { reason: 'node-unsupported' } },
    } as any)
    mount(fields)
    await waitFor(() => expect(document.querySelector('[data-test="node-unsupported"]')).toBeTruthy())
  })

  it('读失败降级：空表单 + 告警条，不崩（§9 负路径）', async () => {
    vi.mocked(apiModule.getConfig).mockRejectedValue(new Error('path not supported'))
    mount(fields)
    expect(await screen.findByText('path not supported')).toBeInTheDocument()
    expect(screen.getByRole('spinbutton')).toHaveValue('')
  })
})
