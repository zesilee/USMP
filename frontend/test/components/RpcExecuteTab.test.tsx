import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import RpcExecuteTab from '../../src/components/config/RpcExecuteTab'
import { UiProvider } from '../../src/ui'
import * as apiModule from '../../src/api'
import type { RpcDef } from '../../src/utils/moduleConsole'

// RpcExecuteTab F2（FE-19/20）：必填门禁、二次确认（取消不下发/确认才执行）、
// 高危标识与危险确认、leafref 下拉注入、结果/失败回显、ExecuteRPC 单次调用。
vi.mock('../../src/api')

const plainRpc: RpcDef = {
  name: 'reset-vlan-statistics',
  label: '清除 VLAN 统计',
  input: [{ path: 'vlan-id', type: 'number', label: 'vlan-id', required: true } as any],
}

const highRiskRpc: RpcDef = {
  name: 'reboot-member',
  label: '重启成员',
  highRisk: true,
  input: [],
}

const leafrefRpc: RpcDef = {
  name: 'clear-if-counters',
  label: '清接口计数',
  input: [
    { path: 'if-name', type: 'string', label: 'if-name', required: true, leafRef: '/ifm:ifm/ifm:interfaces/ifm:interface/ifm:name' } as any,
  ],
}

function mount(rpc: RpcDef) {
  return render(
    <UiProvider>
      <RpcExecuteTab rpc={rpc} module="vlan" device="10.0.0.1" />
    </UiProvider>,
  )
}

let seenDialogs = 0
async function confirmDialog(accept: boolean) {
  // happy-dom 下 Modal 离场动画不结束、旧弹窗 DOM 残留——按出现顺序取**新**弹窗，
  // 查询限定其按钮区（页面执行钮与确认钮同文案）。
  const btns = await waitFor(() => {
    const all = document.querySelectorAll('.ant-modal-confirm-btns')
    if (all.length <= seenDialogs) throw new Error('confirm not open')
    return all[all.length - 1] as HTMLElement
  })
  seenDialogs++
  const name = accept ? /执\s*行|OK/ : /取\s*消|Cancel/
  fireEvent.click(within(btns).getByRole('button', { name }))
}

describe('RpcExecuteTab · 执行门禁与确认（FE-19/20）', () => {
  beforeEach(() => {
    seenDialogs = 0
    vi.clearAllMocks()
    vi.mocked(apiModule.getConfig).mockResolvedValue({ data: { success: true, data: {} } } as any)
    vi.mocked(apiModule.executeRpc).mockResolvedValue({ data: { success: true, data: { reply: '' } } } as any)
  })

  it('必填未填：执行钮禁用（§9 拦截不下发）', () => {
    mount(plainRpc)
    expect(document.querySelector('[data-test="rpc-execute"]')).toBeDisabled()
  })

  it('确认后执行一次并回显成功；取消则零调用（FE-20）', async () => {
    mount(plainRpc)
    fireEvent.change(screen.getByRole('spinbutton'), { target: { value: '10' } })
    fireEvent.blur(screen.getByRole('spinbutton'))
    await waitFor(() => expect(document.querySelector('[data-test="rpc-execute"]')).not.toBeDisabled())

    // 第一轮：取消 → 不下发
    fireEvent.click(document.querySelector('[data-test="rpc-execute"]')!)
    await confirmDialog(false)
    expect(vi.mocked(apiModule.executeRpc)).not.toHaveBeenCalled()

    // 第二轮：确认 → 单次执行 + 成功回显
    fireEvent.click(document.querySelector('[data-test="rpc-execute"]')!)
    await confirmDialog(true)
    await waitFor(() => expect(document.querySelector('[data-test="rpc-result"]')).toBeTruthy())
    expect(vi.mocked(apiModule.executeRpc)).toHaveBeenCalledTimes(1)
    expect(vi.mocked(apiModule.executeRpc)).toHaveBeenCalledWith('10.0.0.1', 'vlan', 'reset-vlan-statistics', { 'vlan-id': '10' })
  })

  it('高危 rpc：标识渲染、确认钮危险样式（FE-20 升级）', async () => {
    mount(highRiskRpc)
    expect(document.querySelector('[data-test="rpc-highrisk"]')).toBeTruthy()
    fireEvent.click(document.querySelector('[data-test="rpc-execute"]')!)
    await waitFor(() => expect(document.querySelector('.ant-modal-confirm-btns .ant-btn-dangerous')).toBeTruthy())
    await confirmDialog(true)
    await waitFor(() => expect(vi.mocked(apiModule.executeRpc)).toHaveBeenCalled())
  })

  it('执行失败：错误如实回显（不重试，R08）', async () => {
    vi.mocked(apiModule.executeRpc).mockResolvedValue({
      data: { success: false, message: 'device busy' },
    } as any)
    mount(highRiskRpc)
    fireEvent.click(document.querySelector('[data-test="rpc-execute"]')!)
    await confirmDialog(true)
    expect(await screen.findByText('device busy')).toBeInTheDocument()
    expect(vi.mocked(apiModule.executeRpc)).toHaveBeenCalledTimes(1)
  })

  it('leafref 输入：目标值注入下拉、选值入 inputs（FE-19）', async () => {
    const user = userEvent.setup()
    vi.mocked(apiModule.getConfig).mockResolvedValue({
      data: { success: true, data: { data: { interface: [{ name: 'GE0/0/1' }, { name: 'GE0/0/2' }] } } },
    } as any)
    mount(leafrefRpc)
    await waitFor(() => expect(document.querySelector('[data-test="leafref-select"]')).toBeTruthy())
    await user.click(screen.getByRole('combobox'))
    await user.click(await screen.findByTitle('GE0/0/1'))
    await waitFor(() => expect(document.querySelector('[data-test="rpc-execute"]')).not.toBeDisabled())
    fireEvent.click(document.querySelector('[data-test="rpc-execute"]')!)
    await confirmDialog(true)
    await waitFor(() =>
      expect(vi.mocked(apiModule.executeRpc)).toHaveBeenCalledWith('10.0.0.1', 'vlan', 'clear-if-counters', { 'if-name': 'GE0/0/1' }),
    )
  })
})
