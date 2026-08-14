import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router'
import ModuleConsolePage from '../../src/views/ModuleConsolePage'
import { UiProvider } from '../../src/ui'
import { useDeviceStore } from '../../src/stores/device'
import { useMenuStore } from '../../src/stores/menu'
import * as apiModule from '../../src/api'

// ModuleConsolePage F2（FE-10/18/19/24 宿主）：schema→Tab 派生、设备空态引导、
// 不支持预标记 Tab 淡化、rpc 直达与未知名报错、深链 device query 覆盖上下文。
vi.mock('../../src/api')
vi.mock('../../src/composables/useFieldLabels', () => ({
  localizeFields: async (f: any) => f,
  localizeRpcs: async (r: any) => r,
}))

const schemaData = {
  title: 'vlan',
  vendor: 'huawei',
  unsupported: ['oam'],
  fields: [
    {
      path: '/vlan/vlans', type: 'group', label: 'vlans',
      fields: [
        {
          path: '/vlan/vlans/vlan', type: 'list', label: 'vlan',
          fields: [{ path: '/vlan/vlans/vlan/id', type: 'number', label: 'id', isKey: true }],
        },
      ],
    },
    { path: '/vlan/oam', type: 'group', label: 'oam', fields: [{ path: '/vlan/oam/x', type: 'string', label: 'x' }] },
  ],
  rpcs: [{ name: 'reset-stat', label: '清统计', input: [] }],
}

function mount(path = '/module/vlan') {
  return render(
    <UiProvider>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/module/:module" element={<ModuleConsolePage />} />
          <Route path="/module/:module/rpc/:rpcName" element={<ModuleConsolePage />} />
        </Routes>
      </MemoryRouter>
    </UiProvider>,
  )
}

describe('ModuleConsolePage · 宿主编排', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useMenuStore.setState({ leftTree: [], leftTreeLoaded: true, nativeLoaded: true })
    useDeviceStore.setState({ devices: [{ id: '10.0.0.1', ip: '10.0.0.1' } as any], selectedDeviceIp: '10.0.0.1' })
    vi.mocked(apiModule.getYangSchema).mockResolvedValue({ data: { success: true, data: schemaData } } as any)
    vi.mocked(apiModule.getOwnership).mockResolvedValue({ data: { data: { claims: [] } } } as any)
    vi.mocked(apiModule.listDevices).mockResolvedValue({ data: { success: true, data: { devices: [] } } } as any)
    vi.mocked(apiModule.getConfig).mockResolvedValue({ data: { success: true, data: {} } } as any)
  })

  it('schema → 一级 Tab 派生渲染；不支持 Tab 头淡化（FE-24 预标记）', async () => {
    mount()
    expect(await screen.findByRole('tab', { name: /vlans/ })).toBeInTheDocument()
    expect(await screen.findByRole('tab', { name: /oam/ })).toBeInTheDocument()
    await waitFor(() => expect(document.querySelector('[data-test="tab-unsupported"]')).toBeTruthy())
    // 带设备取 schema（CN-05 预标记零额外请求）
    expect(vi.mocked(apiModule.getYangSchema)).toHaveBeenCalledWith('vlan', 'nested', '10.0.0.1')
  })

  it('未选设备：引导空态、不渲染 Tab（FE-10）', async () => {
    useDeviceStore.setState({ selectedDeviceIp: '' })
    mount()
    await waitFor(() => expect(document.querySelector('[data-test="select-device-empty"]')).toBeTruthy())
    expect(screen.queryByRole('tab')).toBeNull()
  })

  it('rpc 直达：仅渲染该 rpc 面板；未知 rpc 名明确报错（R08）', async () => {
    mount('/module/vlan/rpc/reset-stat')
    await waitFor(() => expect(document.querySelector('[data-test="rpc-execute-tab"]')).toBeTruthy())
    expect(screen.queryByRole('tab')).toBeNull()
  })

  it('rpc 名不存在 → rpc-not-found 报错条', async () => {
    mount('/module/vlan/rpc/nope')
    await waitFor(() => expect(document.querySelector('[data-test="rpc-not-found"]')).toBeTruthy())
  })

  it('深链 ?device= 覆盖全局设备上下文（query > store）', async () => {
    useDeviceStore.setState({
      devices: [{ id: '10.0.0.1', ip: '10.0.0.1' } as any, { id: '10.0.0.9', ip: '10.0.0.9' } as any],
      selectedDeviceIp: '10.0.0.1',
    })
    mount('/module/vlan?device=10.0.0.9')
    await waitFor(() => expect(useDeviceStore.getState().selectedDeviceIp).toBe('10.0.0.9'))
  })

  it('schema 拉取失败：明确报错不崩（§9）', async () => {
    vi.mocked(apiModule.getYangSchema).mockRejectedValue(new Error('schema boom'))
    mount()
    expect(await screen.findByText('schema boom')).toBeInTheDocument()
  })

  it('软归属徽标（FE-18）：命中模块的意图数展示', async () => {
    vi.mocked(apiModule.getOwnership).mockResolvedValue({
      data: { data: { claims: [{ module: 'vlan', intent: 'net-a' }, { module: 'ifm', intent: 'x' }] } },
    } as any)
    mount()
    await waitFor(() => expect(document.querySelector('[data-test="ownership-badge"]')).toBeTruthy())
  })
})
