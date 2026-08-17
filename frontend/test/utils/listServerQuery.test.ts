import { describe, it, expect, vi, beforeEach } from 'vitest'

// F1（FE-25/BR-13）：服务端 list 查询的请求形状与搜索条件下推映射。
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

import { getConfig } from '../../src/api'
import { buildServerFilters } from '../../src/utils/moduleConsole'
import type { Field } from '../../src/utils/crdSchemaParser'

beforeEach(() => {
  mocks.get.mockReset()
  mocks.get.mockResolvedValue({ data: {} })
})

// 编码承载迁移（inula-request 波次）：filter 可重复且不得出现 filter[]= 的
// 后端契约不变，但承载从 cfg.params(URLSearchParams) 改为直接拼进 URL
// （inula-request 的 params 类型不收 URLSearchParams）。断言解析 URL query。
const queryOf = (url: string) => new URLSearchParams(url.split('?')[1] ?? '')

describe('getConfig 分页查询参数编码（F1）', () => {
  it('query 存在：limit/offset 编码进 URL，filter 可重复不带 []', () => {
    getConfig('10.0.0.1', '/ifm:ifm/ifm:interfaces', false, false, {
      limit: 10,
      offset: 20,
      filters: ['admin-status==up', 'name~=GE'],
    })
    const [url] = mocks.get.mock.calls.at(-1)!
    expect(url.startsWith('/config/10.0.0.1/ifm:ifm/ifm:interfaces?')).toBe(true)
    const p = queryOf(url)
    expect(p.get('limit')).toBe('10')
    expect(p.get('offset')).toBe('20')
    expect(p.getAll('filter')).toEqual(['admin-status==up', 'name~=GE'])
    expect(url).not.toContain('%5B%5D') // 不得出现 filter[]=
  })

  it('sort 存在才带 sort_dir（缺省 asc）；offset=0 省略', () => {
    getConfig('10.0.0.1', '/p', false, false, { limit: 50, sort: 'mtu' })
    const [url] = mocks.get.mock.calls.at(-1)!
    const p = queryOf(url)
    expect(p.get('sort')).toBe('mtu')
    expect(p.get('sort_dir')).toBe('asc')
    expect(p.has('offset')).toBe(false)
    expect(p.has('filter')).toBe(false)
  })

  it('include_state/force_refresh 与分页参数正交组合，状态读放宽超时', () => {
    getConfig('10.0.0.1', '/fib:fib', true, true, { limit: 50, offset: 100, sortDir: 'desc' })
    const [url, cfg] = mocks.get.mock.calls.at(-1)!
    const p = queryOf(url)
    expect(p.get('include_state')).toBe('true')
    expect(p.get('force_refresh')).toBe('true')
    expect(p.get('limit')).toBe('50')
    expect(p.has('sort')).toBe(false) // 无 sort 不带 sort_dir
    expect(p.has('sort_dir')).toBe(false)
    expect(cfg.timeout).toBe(30000)
  })

  it('无 query：请求形状与旧契约完全一致（回归锚点）', () => {
    getConfig('10.0.0.1', '/p')
    const [, cfg] = mocks.get.mock.calls.at(-1)!
    expect(cfg.params).toEqual({ force_refresh: false })
  })
})

const fields = [
  { name: 'admin-status', path: 'x/admin-status', type: 'enum' },
  { name: 'name', path: 'x/name', type: 'string' },
  { name: 'mtu', path: 'x/mtu', type: 'number' },
] as unknown as Field[]

describe('buildServerFilters 搜索条件下推（F1）', () => {
  it('enum 全等（==）、其余包含（~=），与 filterRows 本地语义一一对应', () => {
    expect(
      buildServerFilters({ 'admin-status': 'up', name: 'GE', mtu: 1500 }, fields),
    ).toEqual(['admin-status==up', 'name~=GE', 'mtu~=1500'])
  })

  it('空串与 null/undefined 条件跳过', () => {
    expect(buildServerFilters({ name: '', 'admin-status': null, mtu: undefined }, fields)).toEqual([])
  })
})
