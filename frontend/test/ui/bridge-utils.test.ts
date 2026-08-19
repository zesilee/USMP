import { describe, it, expect } from 'vitest'
import { textOf, pickDefault } from '../../src/ui/bridge'

// bridge 工具函数 F1（覆盖率回填，#393 红灯复盘：textOf 空值分支与
// pickDefault 的 default:null 提前停分支未覆盖）。
describe('textOf（JSX→纯文本）', () => {
  it('null/undefined → 空串；数字/布尔字符串化', () => {
    expect(textOf(null)).toBe('')
    expect(textOf(undefined)).toBe('')
    expect(textOf(42)).toBe('42')
    expect(textOf(true as unknown as string)).toBe('true')
  })
})

describe('pickDefault（多层 default 剥离）', () => {
  it('default 显式为 null 时提前停、返回当前层', () => {
    const mod = { default: null, other: 1 }
    expect(pickDefault(mod)).toBe(mod)
  })
  it('多层嵌套剥到组件；guard 防无限（自引用 5 层截断不挂死）', () => {
    const fn = () => null
    expect(pickDefault({ default: { default: fn } })).toBe(fn)
    const loop: { default?: unknown } = {}
    loop.default = loop
    expect(pickDefault(loop)).toBe(loop)
  })
  it('null 入参返回 undefined', () => {
    expect(pickDefault(null)).toBeUndefined()
  })
})
