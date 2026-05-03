import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import DetailDrawer from '../../src/components/layout/DetailDrawer.vue'
import ElementPlus from 'element-plus'

describe('DetailDrawer Component', () => {
  it('should show when visible is true', () => {
    const wrapper = mount(DetailDrawer, {
      props: { modelValue: true, title: '测试标题' },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.text()).toContain('测试标题')
  })

  it('should emit update event when close', async () => {
    const wrapper = mount(DetailDrawer, {
      props: { modelValue: true, title: '测试' },
      global: { plugins: [ElementPlus] }
    })
    const drawer = wrapper.findComponent({ name: 'ElDrawer' })
    drawer.vm.$emit('close')
    await wrapper.vm.$nextTick()
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('should display submit button when showFooter is true', () => {
    const wrapper = mount(DetailDrawer, {
      props: { modelValue: true, title: '测试', showFooter: true, submitText: '确认' },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.text()).toContain('确认')
    expect(wrapper.text()).toContain('取消')
  })
})
