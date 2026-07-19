import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useDeviceStore } from '../../src/stores/device'

vi.mock('../../src/api')

// 全局设备上下文（device-first-config-context / FE-10）：
// 设备作用域配置页的唯一选中态，IP 口径，跨页面共享。
describe('device store · 全局设备上下文 selectedDeviceIp', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('初始为空串（未选设备）', () => {
    const store = useDeviceStore()
    expect(store.selectedDeviceIp).toBe('')
  })

  it('selectDevice(ip) 写入上下文', () => {
    const store = useDeviceStore()
    store.selectDevice('192.168.1.2')
    expect(store.selectedDeviceIp).toBe('192.168.1.2')
  })

  it('clearSelection() 清空上下文', () => {
    const store = useDeviceStore()
    store.selectDevice('192.168.1.2')
    store.clearSelection()
    expect(store.selectedDeviceIp).toBe('')
  })

  it('同一 pinia 下多次 useDeviceStore 共享同一上下文（跨页面语义）', () => {
    const a = useDeviceStore()
    a.selectDevice('10.0.0.1')
    const b = useDeviceStore()
    expect(b.selectedDeviceIp).toBe('10.0.0.1')
  })
})
