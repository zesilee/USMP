import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useLocaleStore, LOCALE_STORAGE_KEY } from '../../src/stores/locale'

// UI-01 F1：默认 zh-cn、切换、localStorage 持久化、非法存量值回退。
describe('locale store（UI-01）', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
  })

  it('默认 zh-cn', () => {
    const store = useLocaleStore()
    expect(store.locale).toBe('zh-cn')
  })

  it('切换 en-us 并持久化', () => {
    const store = useLocaleStore()
    store.setLocale('en-us')
    expect(store.locale).toBe('en-us')
    expect(localStorage.getItem(LOCALE_STORAGE_KEY)).toBe('en-us')
  })

  it('从 localStorage 恢复', () => {
    localStorage.setItem(LOCALE_STORAGE_KEY, 'en-us')
    const store = useLocaleStore()
    expect(store.locale).toBe('en-us')
  })

  it('非法存量值回退 zh-cn（R08）', () => {
    localStorage.setItem(LOCALE_STORAGE_KEY, 'fr-fr')
    const store = useLocaleStore()
    expect(store.locale).toBe('zh-cn')
  })

  it('setLocale 拒绝非法值', () => {
    const store = useLocaleStore()
    store.setLocale('xx' as any)
    expect(store.locale).toBe('zh-cn')
  })
})
