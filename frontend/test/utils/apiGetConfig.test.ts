import { describe, it, expect, vi, beforeEach } from 'vitest'

// F1 回归（FE-14 状态通道）：getConfig 的请求形状——include_state 参数与
// 状态读超时（对齐后端设备侧 30s，默认 15s 会在真机大状态子树上先超时）。
const mocks = vi.hoisted(() => ({ get: vi.fn() }))

vi.mock('axios', () => ({
  default: {
    create: vi.fn(() => ({
      get: mocks.get,
      post: vi.fn(),
      put: vi.fn(),
      delete: vi.fn(),
    })),
  },
}))

import { getConfig } from '../../src/api'

beforeEach(() => {
  mocks.get.mockReset()
  mocks.get.mockResolvedValue({ data: {} })
})

describe('getConfig 请求形状（F1）', () => {
  it('默认配置读：无 include_state、不覆写超时（走全局 15s）', () => {
    getConfig('10.0.0.1', '/ifm:ifm/ifm:interfaces')
    const [url, cfg] = mocks.get.mock.calls.at(-1)!
    expect(url).toBe('/config/10.0.0.1/ifm:ifm/ifm:interfaces')
    expect(cfg.params.include_state).toBeUndefined()
    expect(cfg.timeout).toBeUndefined()
  })

  it('includeState=true：带 include_state 且放宽超时到 30s（对齐后端设备侧）', () => {
    getConfig('10.0.0.1', '/devm:devm/physical-entitys', false, true)
    const [, cfg] = mocks.get.mock.calls.at(-1)!
    expect(cfg.params.include_state).toBe(true)
    expect(cfg.timeout).toBe(30000)
  })

  it('force_refresh 透传且与 includeState 正交', () => {
    getConfig('10.0.0.1', '/devm:devm/schedule-reboot', true, true)
    const [, cfg] = mocks.get.mock.calls.at(-1)!
    expect(cfg.params.force_refresh).toBe(true)
    expect(cfg.params.include_state).toBe(true)
  })
})
