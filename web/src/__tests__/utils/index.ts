import type { VueWrapper } from '@vue/test-utils'
import { mount } from '@vue/test-utils'
import { createApp, type Component, type App } from 'vue'
import ElementPlus from 'element-plus'

// 全局配置
export function createTestWrapper<T extends Component>(
  component: T,
  options: Parameters<typeof mount>[1] = {}
): VueWrapper {
  return mount(component, {
    global: {
      plugins: [ElementPlus],
      ...options.global
    },
    ...options
  })
}

// 等待 Element Plus 组件的 DOM 更新
export async function waitForUpdate(wrapper: VueWrapper, ms = 100): Promise<void> {
  await wrapper.vm.$nextTick()
  await new Promise(resolve => setTimeout(resolve, ms))
}

// 获取 Element Plus 弹窗内容
export function getPopupContent(): HTMLElement | null {
  return document.querySelector('.el-popper')
}

// 模拟用户输入
export async function simulateInput(wrapper: VueWrapper, selector: string, value: string): Promise<void> {
  const input = wrapper.find(selector)
  await input.setValue(value)
  await input.trigger('input')
  await wrapper.vm.$nextTick()
}

// 模拟点击按钮
export async function simulateClick(wrapper: VueWrapper, selector: string): Promise<void> {
  const button = wrapper.find(selector)
  await button.trigger('click')
  await wrapper.vm.$nextTick()
}
