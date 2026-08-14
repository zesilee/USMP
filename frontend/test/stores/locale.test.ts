import { describe, it, expect, beforeEach } from 'vitest'
import { useLocaleStore, LOCALE_STORAGE_KEY } from '../../src/stores/locale'

// UI-01 F1：默认 zh-cn、切换、localStorage 持久化、非法存量值回退。
// zustand 惯用法：S() 取新鲜快照；__resetForTest 按当前 localStorage 重算初始态
// （对齐旧 Pinia 每用例新建 store 的语义）。
const S = () => useLocaleStore.getState()

describe('locale store（UI-01）', () => {
  beforeEach(() => {
    localStorage.clear()
    S().__resetForTest()
  })

  it('默认 zh-cn', () => {
    expect(S().locale).toBe('zh-cn')
  })

  it('切换 en-us 并持久化', () => {
    S().setLocale('en-us')
    expect(S().locale).toBe('en-us')
    expect(localStorage.getItem(LOCALE_STORAGE_KEY)).toBe('en-us')
  })

  it('从 localStorage 恢复', () => {
    localStorage.setItem(LOCALE_STORAGE_KEY, 'en-us')
    S().__resetForTest()
    expect(S().locale).toBe('en-us')
  })

  it('非法存量值回退 zh-cn（R08）', () => {
    localStorage.setItem(LOCALE_STORAGE_KEY, 'fr-fr')
    S().__resetForTest()
    expect(S().locale).toBe('zh-cn')
  })

  it('setLocale 拒绝非法值', () => {
    S().setLocale('xx' as any)
    expect(S().locale).toBe('zh-cn')
  })
})
