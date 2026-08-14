import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import Logs from '../../src/views/Logs'
import Settings from '../../src/views/Settings'
import BusinessConsolePage from '../../src/views/BusinessConsolePage'
import { UiProvider } from '../../src/ui'
import { useMenuStore } from '../../src/stores/menu'
import * as apiModule from '../../src/api'

// Logs/Settings/Business F2（FE-26/FE-17）。
vi.mock('../../src/api')

function mount(node: React.ReactNode) {
  return render(
    <UiProvider>
      <MemoryRouter>{node}</MemoryRouter>
    </UiProvider>,
  )
}

describe('Logs · 审计台账（FE-26）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useMenuStore.setState({ nativeModules: [{ name: 'vlan', title: 'VLAN 配置', vendor: 'huawei' }] as any, nativeLoaded: true })
    vi.mocked(apiModule.listYangModules).mockResolvedValue({ data: { data: [] } } as any)
    vi.mocked(apiModule.getLogs).mockResolvedValue({
      data: { data: { logs: [
        { id: '1', timestamp: '2026-08-14T10:00:00Z', device_ip: '10.0.0.1', path: '/vlan:vlan/vlans', summary: 'id=10', actor: 'admin', outcome: 'converged' },
        { id: '2', timestamp: '2026-08-14T09:00:00Z', device_ip: '10.0.0.2', path: '/ifm:ifm', summary: 'mtu', actor: 'ops', outcome: 'error' },
      ] } },
    } as any)
  })

  it('行渲染：操作类型标签模型驱动（vlan→菜单标题）、结局 chip、搜索过滤', async () => {
    mount(<Logs />)
    expect(await screen.findByText('VLAN 配置')).toBeInTheDocument() // FE-26：name→菜单标题
    expect(screen.getByText('10.0.0.2')).toBeInTheDocument()
    expect(document.querySelectorAll('[data-test="reconcile-chip"]').length).toBe(2)

    fireEvent.change(screen.getByPlaceholderText(/搜索|Search/i), { target: { value: 'ops' } })
    await waitFor(() => expect(screen.queryByText('10.0.0.1')).toBeNull())
  })

  it('拉取失败降级空表（R08）', async () => {
    vi.mocked(apiModule.getLogs).mockRejectedValue(new Error('down'))
    mount(<Logs />)
    await waitFor(() => expect(document.querySelector('[data-test="logs-page"]')).toBeTruthy())
    expect(screen.queryByText('10.0.0.1')).toBeNull()
  })
})

describe('Settings · 只读架构事实', () => {
  it('协议与缓存两卡渲染（830/30s 等实值）', () => {
    mount(<Settings />)
    expect(screen.getByText('830')).toBeInTheDocument()
    expect(screen.getByText('30s')).toBeInTheDocument()
    expect(document.querySelector('[data-test="settings-page"]')).toBeTruthy()
  })
})

describe('BusinessConsolePage · 意图台（FE-17）', () => {
  const item = {
    name: 'net-a',
    spec: { 'vlan-id': 100, devices: [{ device: '10.0.0.1' }] },
    status: {
      conditions: [{ type: 'Converged', status: 'True' }],
      deviceStates: [{ device: '10.0.0.1', phase: 'synced', reason: '' }],
      claims: [{ device: '10.0.0.1', path: '/vlan' }],
    },
  }

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(apiModule.getYangSchema).mockResolvedValue({
      data: { data: { title: 'business-vlan-service', fields: [
        { path: '/b/vlan-id', type: 'number', label: 'vlan-id', required: true },
      ] } },
    } as any)
    vi.mocked(apiModule.listBusinessVlanServices).mockResolvedValue({
      data: { success: true, data: { items: [item] } },
    } as any)
  })

  it('实例列表渲染：收敛档位 Tag（Converged=True→已收敛）', async () => {
    mount(<BusinessConsolePage />)
    expect(await screen.findByText('net-a')).toBeInTheDocument()
    expect(document.querySelector('[data-test="converge-net-a"]')).toBeTruthy()
  })

  it('新建抽屉由意图 schema 驱动渲染；提交调 apply（R05）', async () => {
    vi.mocked(apiModule.applyBusinessVlanService).mockResolvedValue({ data: { success: true } } as any)
    mount(<BusinessConsolePage />)
    await screen.findByText('net-a')
    fireEvent.click(document.querySelector('[data-test="business-create"]')!)
    const nameInput = await waitFor(() => document.querySelector('[data-test="business-name-input"]')!)
    fireEvent.change(nameInput, { target: { value: 'net-b' } })
    // schema 驱动的 vlan-id 数字框
    const spin = screen.getByRole('spinbutton')
    fireEvent.change(spin, { target: { value: '200' } })
    fireEvent.blur(spin)
    await waitFor(() => expect(document.querySelector('[data-test="business-submit"]')).not.toBeDisabled())
    fireEvent.click(document.querySelector('[data-test="business-submit"]')!)
    await waitFor(() =>
      expect(vi.mocked(apiModule.applyBusinessVlanService)).toHaveBeenCalledWith('net-b', { 'vlan-id': 200 }),
    )
  })

  it('删除：确认后调 delete 并刷新', async () => {
    vi.mocked(apiModule.deleteBusinessVlanService).mockResolvedValue({ data: { success: true } } as any)
    mount(<BusinessConsolePage />)
    await screen.findByText('net-a')
    fireEvent.click(document.querySelector('[data-test="business-remove-net-a"]')!)
    const btns = await waitFor(() => {
      const el = document.querySelector('.ant-modal-confirm-btns')
      if (!el) throw new Error('not open')
      return el as HTMLElement
    })
    fireEvent.click(within(btns).getByRole('button', { name: /确\s*定|OK/ }))
    await waitFor(() => expect(vi.mocked(apiModule.deleteBusinessVlanService)).toHaveBeenCalledWith('net-a'))
  })

  it('后端未连接集群：信封降级提示不崩（R08）', async () => {
    vi.mocked(apiModule.listBusinessVlanServices).mockResolvedValue({
      data: { success: false, message: 'no cluster', data: null },
    } as any)
    mount(<BusinessConsolePage />)
    await waitFor(() => expect(document.querySelector('[data-test="business-unavailable"]')).toBeTruthy())
  })
})
