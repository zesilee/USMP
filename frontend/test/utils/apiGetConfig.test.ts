import { describe, it, expect, vi, beforeEach } from 'vitest'

// F1 回归（FE-14 状态通道）：getConfig 的请求形状——include_state 参数与
// 状态读超时（对齐后端设备侧 30s，默认 15s 会在真机大状态子树上先超时）。
const mocks = vi.hoisted(() => ({ get: vi.fn() }))

vi.mock('inula-request', () => ({
  default: {
    create: vi.fn(() => ({
      get: mocks.get,
      post: vi.fn(),
      put: vi.fn(),
      delete: vi.fn(),
    })),
  },
}))

import { getConfig, getYangSchema } from '../../src/api'

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

// FE-24/CN-05：getYangSchema 可选 device 参数——带 device 时后端响应附
// unsupported 预标记（该设备已学习的不支持子路径，相对模块根首段）。
describe('getYangSchema 请求形状（FE-24）', () => {
  it('device 参数透传：params 同时含 form 与 device', () => {
    getYangSchema('devm', 'nested', '10.0.0.1')
    const [url, cfg] = mocks.get.mock.calls.at(-1)!
    expect(url).toBe('/yang/schema/devm')
    expect(cfg.params.form).toBe('nested')
    expect(cfg.params.device).toBe('10.0.0.1')
  })

  it('不带 device：契约不变（无 device 键，向后兼容）', () => {
    getYangSchema('devm', 'nested')
    const [, cfg] = mocks.get.mock.calls.at(-1)!
    expect(cfg.params.form).toBe('nested')
    expect('device' in cfg.params).toBe(false)
  })

  it('仅 device 无 form：不带 form 键', () => {
    getYangSchema('devm', undefined, '10.0.0.1')
    const [, cfg] = mocks.get.mock.calls.at(-1)!
    expect('form' in cfg.params).toBe(false)
    expect(cfg.params.device).toBe('10.0.0.1')
  })
})
