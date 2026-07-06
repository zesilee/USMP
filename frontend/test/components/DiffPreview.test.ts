import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import DiffPreview from '../../src/components/config/DiffPreview.vue'
import type { DiffEntry } from '../../src/utils/configDiff'

describe('DiffPreview · 下发差异预览', () => {
  it('改动项渲染 was→now', () => {
    const diff: DiffEntry[] = [{ key: 'name', label: '名称', was: 'old', now: 'new', isNew: false }]
    const w = mount(DiffPreview, { props: { diff } })
    expect(w.find('.was').text()).toBe('old')
    expect(w.find('.now').text()).toBe('new')
    expect(w.find('.arrow').exists()).toBe(true)
    expect(w.find('.preview-head b').text()).toBe('1')
  })

  it('新增项渲染 now + 新增标签，无 was/arrow', () => {
    const diff: DiffEntry[] = [{ key: 'id', label: 'VLAN ID', was: '', now: 200, isNew: true }]
    const w = mount(DiffPreview, { props: { diff } })
    expect(w.find('.now').text()).toBe('200')
    expect(w.find('.tag-new').text()).toBe('新增')
    expect(w.find('.was').exists()).toBe(false)
    expect(w.find('.arrow').exists()).toBe(false)
  })

  it('无改动显示空态占位', () => {
    const w = mount(DiffPreview, { props: { diff: [] } })
    expect(w.find('.preview-empty').exists()).toBe(true)
    expect(w.find('.dva').exists()).toBe(false)
    expect(w.find('.preview-head b').text()).toBe('0')
  })

  it('空值 was 显示占位 —', () => {
    const diff: DiffEntry[] = [{ key: 'x', label: 'X', was: null, now: 'v', isNew: false }]
    const w = mount(DiffPreview, { props: { diff } })
    expect(w.find('.was').text()).toBe('—')
  })
})
