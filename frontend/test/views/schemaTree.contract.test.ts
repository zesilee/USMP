import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ElementPlus from 'element-plus'
import router from '../../src/router'
import App from '../../src/App.vue'
import { useDeviceStore } from '../../src/stores/device'
import { getYangSchema } from '../../src/api'

vi.mock('../../src/api')

// 嵌套 schema：container vlans → list vlan → 叶子（含主键 id）
const vlanNested = {
  fields: [
    {
      path: '/vlan/vlans',
      type: 'group',
      label: 'vlans',
      fields: [
        {
          path: '/vlan/vlans/vlan',
          type: 'list',
          label: 'vlan',
          fields: [
            { path: '/vlan/vlans/vlan/id', type: 'number', label: 'id', required: true },
            { path: '/vlan/vlans/vlan/name', type: 'string', label: 'name' },
          ],
        },
      ],
    },
  ],
}

// FE-13：/config/vlan 重定向到通用模块控制台，页面由 schema 派生 Tab 渲染
//（原「左侧 YANG 架构树」随 DeviceConfigPage 退出路由，SchemaTree 组件契约
// 由 test/components/SchemaTree.test.ts 单测继续兜底）。
describe('旧配置路由重定向到通用模块控制台（真路由挂载）', () => {
  beforeEach(() => {
    vi.mocked(getYangSchema).mockResolvedValue({ data: { success: true, data: vlanNested } } as any)
  })

  it('/config/vlan → /module/vlan，schema 派生出 vlans 列表 Tab', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    // 全局设备上下文：控制台内容区以已选设备为前提（未选走引导空态）。
    useDeviceStore().selectDevice('192.168.1.1')
    const wrapper = mount(App, { global: { plugins: [pinia, ElementPlus, router] } })
    await router.push('/config/vlan')
    await flushPromises()

    expect(router.currentRoute.value.fullPath).toBe('/module/vlan')
    const tabs = wrapper.findAll('.el-tabs__item').map((n) => n.text().trim())
    expect(tabs).toContain('vlans')

    wrapper.unmount()
  })
})
