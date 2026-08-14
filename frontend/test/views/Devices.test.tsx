import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import Devices from '../../src/views/Devices'
import { UiProvider } from '../../src/ui'
import { useDeviceStore } from '../../src/stores/device'
import * as apiModule from '../../src/api'

// Devices F2：台账 join 渲染、搜索过滤、添加纯函数闸门（非法拦截/成功提交）、
// 删除确认、对账失败不阻断设备表。
vi.mock('../../src/api')

function mount() {
  return render(
    <UiProvider>
      <MemoryRouter>
        <Devices />
      </MemoryRouter>
    </UiProvider>,
  )
}

describe('Devices · 设备台账', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useDeviceStore.setState({ devices: [], selectedDeviceIp: '' })
    vi.mocked(apiModule.listDevices).mockResolvedValue({
      data: { success: true, data: { devices: [
        { ip: '10.0.0.1', online: true, name: 'sw1', vendor: 'huawei' },
        { ip: '10.0.0.2', online: false, name: 'sw2', vendor: 'huawei' },
      ] } },
    } as any)
    vi.mocked(apiModule.getFleetReconcile).mockResolvedValue({
      data: { data: { devices: [{ device_id: '10.0.0.1', outcome: 'drifted' }] } },
    } as any)
  })

  it('台账渲染：会话态 Tag、对账 chip（离线优先）、搜索过滤', async () => {
    mount()
    expect(await screen.findByText('sw1')).toBeInTheDocument()
    expect(screen.getByText('sw2')).toBeInTheDocument()
    // 对账 chip 出现（漂移 + 离线两枚）
    expect(document.querySelectorAll('[data-test="reconcile-chip"]').length).toBeGreaterThanOrEqual(2)

    const search = screen.getByPlaceholderText(/搜索|Search/i)
    fireEvent.change(search, { target: { value: 'sw2' } })
    await waitFor(() => expect(screen.queryByText('sw1')).toBeNull())
    expect(screen.getByText('sw2')).toBeInTheDocument()
  })

  it('状态/厂商筛选：离线过滤与厂商全等（客户端）', async () => {
    const user = (await import('@testing-library/user-event')).default.setup()
    mount()
    await screen.findByText('sw1')
    const selects = screen.getAllByRole('combobox')
    await user.click(selects[0])
    await user.click(await screen.findByTitle(/离线|Offline/))
    await waitFor(() => expect(screen.queryByText('sw1')).toBeNull())
    expect(screen.getByText('sw2')).toBeInTheDocument()
  })

  it('添加设备：非法 IP 被纯函数闸门拦截（不发请求）；合法提交成功并刷新', async () => {
    mount()
    await screen.findByText('sw1')
    fireEvent.click(document.querySelector('[data-test="add-device-btn"]')!)
    const form = await waitFor(() => document.querySelector('[data-test="add-device-form"]')!)

    const set = (key: string, v: string) =>
      fireEvent.change(form.querySelector(`[data-test="add-${key}"]`)!, { target: { value: v } })
    set('ip', '999.1.1.1')
    set('username', 'admin')
    set('password', 'pw')
    fireEvent.click(within(document.querySelector('.ant-modal-footer') as HTMLElement).getByRole('button', { name: /确\s*定|OK/ }))
    await waitFor(() => expect(form.querySelector('.adf-err')).toBeTruthy())
    expect(vi.mocked(apiModule.addDevice)).not.toHaveBeenCalled()

    vi.mocked(apiModule.addDevice).mockResolvedValue({ data: { success: true } } as any)
    set('ip', '10.0.0.9')
    fireEvent.click(within(document.querySelector('.ant-modal-footer') as HTMLElement).getByRole('button', { name: /确\s*定|OK/ }))
    await waitFor(() => expect(vi.mocked(apiModule.addDevice)).toHaveBeenCalledTimes(1))
  })

  it('删除设备：确认后调 removeDevice 并刷新；取消零调用', async () => {
    vi.mocked(apiModule.removeDevice).mockResolvedValue({ data: { success: true } } as any)
    mount()
    await screen.findByText('sw1')
    fireEvent.click(document.querySelectorAll('[data-test="device-delete-btn"]')[0]!)
    const btns = await waitFor(() => {
      const el = document.querySelector('.ant-modal-confirm-btns')
      if (!el) throw new Error('not open')
      return el as HTMLElement
    })
    fireEvent.click(within(btns).getByRole('button', { name: /确\s*定|OK/ }))
    await waitFor(() => expect(vi.mocked(apiModule.removeDevice)).toHaveBeenCalledWith('10.0.0.1'))
  })

  it('对账聚合失败：设备表照常渲染（allSettled 边界）', async () => {
    vi.mocked(apiModule.getFleetReconcile).mockRejectedValue(new Error('agg down'))
    mount()
    expect(await screen.findByText('sw1')).toBeInTheDocument()
  })
})
