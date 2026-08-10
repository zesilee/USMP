import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import router from '../../src/router'
import App from '../../src/App.vue'
import { getYangSchema } from '../../src/api'

vi.mock('../../src/api')

// 回归：/module/vlan 与 /module/ifm 是同一个 ModuleConsolePage 组件，vue-router 在
// 同组件路由间复用实例 → setup/onMounted 只跑一次 → 切换后 schema 不重载（接口表单显示
// VLAN 字段）。此测试在真 App+真路由下做 SPA 内导航，断言按各自 module 重新拉取 schema。
//（旧 /config/* 重定向已退役，入口直接走现役路由。）
describe('设备配置页在同组件路由间切换应按 module 重载 schema', () => {
  beforeEach(() => {
    vi.mocked(getYangSchema).mockResolvedValue({ data: { success: true, data: { fields: [] } } } as any)
  })

  it('从 VLAN 导航到接口应用 ifm 重新拉取 schema（而非沿用 vlan）', async () => {
    const wrapper = mount(App, { global: { plugins: [createPinia(), ElementPlus, router] } })

    await router.push('/module/vlan')
    await flushPromises()
    await router.push('/module/ifm')
    await flushPromises()

    const modules = vi.mocked(getYangSchema).mock.calls.map((c) => c[0])
    expect(modules).toContain('vlan')
    expect(modules).toContain('ifm')

    wrapper.unmount()
  })
})
