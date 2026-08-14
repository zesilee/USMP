import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useDeviceStore } from '../../src/stores/device'
import { resetStores } from './reset'

vi.mock('../../src/api')

const S = () => useDeviceStore.getState()

// 全局设备上下文（device-first-config-context / FE-10）：
// 设备作用域配置页的唯一选中态，IP 口径，跨页面共享。
describe('device store · 全局设备上下文 selectedDeviceIp', () => {
  beforeEach(() => {
    resetStores()
  })

  it('初始为空串（未选设备）', () => {
    expect(S().selectedDeviceIp).toBe('')
  })

  it('selectDevice(ip) 写入上下文', () => {
    S().selectDevice('192.168.1.2')
    expect(S().selectedDeviceIp).toBe('192.168.1.2')
  })

  it('clearSelection() 清空上下文', () => {
    S().selectDevice('192.168.1.2')
    S().clearSelection()
    expect(S().selectedDeviceIp).toBe('')
  })
})
