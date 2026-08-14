import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useDeviceStore } from '../../src/stores/device'
import * as apiModule from '../../src/api'
import { resetStores } from './reset'

// device store 取数与探活（F1，zustand 重建补层）：旧栈由 Devices.vue F2 间接
// 覆盖，React 重建期表格页未回归，先以 F1 钉住信封兼容/归一化/降级语义。
const S = () => useDeviceStore.getState()

describe('device store · fetchDevices 信封兼容与归一化', () => {
  beforeEach(() => {
    resetStores()
    vi.restoreAllMocks()
  })

  it('新契约信封 {data:{devices:[...]}}：归一化 + 缺字段 ip 兜底（R08）', async () => {
    vi.spyOn(apiModule, 'listDevices').mockResolvedValue({
      data: { success: true, data: { devices: [{ ip: '10.0.0.1', online: true }] } },
    } as any)
    await S().fetchDevices()
    expect(S().devices).toEqual([
      {
        id: '10.0.0.1', ip: '10.0.0.1', name: '10.0.0.1', vendor: '', model: '', role: '',
        status: 'online', lastSync: '',
      },
    ])
    expect(S().onlineCount()).toBe(1)
    expect(S().offlineCount()).toBe(0)
    expect(S().isLoading).toBe(false)
  })

  it('旧二进制扁平信封 {data:[...]}：status 字符串口径兼容', async () => {
    vi.spyOn(apiModule, 'listDevices').mockResolvedValue({
      data: { success: true, data: [{ ip: '10.0.0.2', status: 'offline', name: 'sw2' }] },
    } as any)
    await S().fetchDevices()
    expect(S().devices[0].status).toBe('offline')
    expect(S().devices[0].name).toBe('sw2')
    expect(S().offlineCount()).toBe(1)
  })

  it('请求失败：清空列表不抛错（R08 负路径），loading 复位', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {})
    vi.spyOn(apiModule, 'listDevices').mockRejectedValue(new Error('down'))
    await expect(S().fetchDevices()).resolves.toBeUndefined()
    expect(S().devices).toEqual([])
    expect(S().isLoading).toBe(false)
  })
})

describe('device store · testConnection 探活三态', () => {
  beforeEach(() => {
    resetStores()
    vi.restoreAllMocks()
  })

  it('connected=true → success', async () => {
    vi.spyOn(apiModule, 'getDeviceStatus').mockResolvedValue({
      data: { data: { connected: true } },
    } as any)
    const r = await S().testConnection('10.0.0.1')
    expect(r.success).toBe(true)
    expect(r.message).toBeTruthy()
  })

  it('connected=false → 失败文案', async () => {
    vi.spyOn(apiModule, 'getDeviceStatus').mockResolvedValue({
      data: { data: { connected: false } },
    } as any)
    const r = await S().testConnection('10.0.0.1')
    expect(r.success).toBe(false)
  })

  it('请求异常 → 失败不抛（R08）', async () => {
    vi.spyOn(apiModule, 'getDeviceStatus').mockRejectedValue(new Error('timeout'))
    const r = await S().testConnection('10.0.0.1')
    expect(r.success).toBe(false)
    expect(r.message).toBeTruthy()
  })
})
