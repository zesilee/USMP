import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import BusinessConsolePage from '../../src/views/BusinessConsolePage.vue'
import * as apiModule from '../../src/api'

// FE-17（F2）——平台作用域业务控制台：实例列表 + status 聚合呈现 + 三态
// （列表/新建表单/详情）+ 无集群降级告警 + 删除确认。

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { module: 'business-vlan-service' } }),
}))

const schemaFixture = {
  title: 'business-vlan-service',
  vendor: 'usmp',
  description: '跨设备 VLAN 打通',
  fields: [
    { path: '/business-vlan-service/vlan-id', type: 'number', label: '业务 VLAN', required: true, minimum: 1, maximum: 4094 },
    { path: '/business-vlan-service/name', type: 'string', label: '业务名称' },
    {
      path: '/business-vlan-service/devices',
      type: 'list',
      label: '设备清单',
      fields: [
        { path: '/business-vlan-service/devices/ip', type: 'string', label: '设备 IP', required: true, isKey: true },
        { path: '/business-vlan-service/devices/access-ports', type: 'leaf-list', label: 'Access 口' },
        { path: '/business-vlan-service/devices/trunk-ports', type: 'leaf-list', label: 'Trunk 口' },
      ],
    },
  ],
}

const itemsFixture = [
  {
    name: 'biz-100',
    spec: { 'vlan-id': 100, name: 'office', devices: [{ ip: '10.0.0.1' }, { ip: '10.0.0.2' }] },
    status: {
      conditions: [
        { type: 'Validated', status: 'True' },
        { type: 'Converged', status: 'True' },
      ],
      deviceStates: [
        { device: '10.0.0.1', phase: 'synced', reason: '' },
        { device: '10.0.0.2', phase: 'synced', reason: '' },
      ],
      claims: [{ device: '10.0.0.1', module: 'vlan', path: '/vlan:vlan/vlan:vlans/vlan[id=100]' }],
    },
  },
  {
    name: 'biz-200',
    spec: { 'vlan-id': 200, devices: [{ ip: '10.0.0.1' }] },
    status: {
      conditions: [
        { type: 'Validated', status: 'True' },
        { type: 'Converged', status: 'False', reason: 'PushFailed' },
      ],
      deviceStates: [{ device: '10.0.0.1', phase: 'failed', reason: 'device offline' }],
    },
  },
]

function mountPage() {
  return mount(BusinessConsolePage, {
    global: { plugins: [createPinia(), ElementPlus] },
  })
}

describe('BusinessConsolePage (FE-17)', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    vi.spyOn(apiModule, 'getYangSchema').mockResolvedValue({
      data: { success: true, data: schemaFixture },
    } as any)
  })

  it('渲染实例列表与收敛状态聚合（全收敛/部分失败）', async () => {
    vi.spyOn(apiModule, 'listBusinessVlanServices').mockResolvedValue({
      data: { success: true, data: { items: itemsFixture } },
    } as any)
    const wrapper = mountPage()
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('biz-100')
    expect(text).toContain('biz-200')
    expect(wrapper.find('[data-test="converge-biz-100"]').text()).toContain('已收敛')
    expect(wrapper.find('[data-test="converge-biz-200"]').text()).toContain('部分失败 0/1')
  })

  it('校验失败的实例呈现校验失败态', async () => {
    vi.spyOn(apiModule, 'listBusinessVlanServices').mockResolvedValue({
      data: {
        success: true,
        data: {
          items: [
            {
              name: 'bad',
              spec: { 'vlan-id': 100, devices: [] },
              status: { conditions: [{ type: 'Validated', status: 'False', message: 'devices empty' }] },
            },
          ],
        },
      },
    } as any)
    const wrapper = mountPage()
    await flushPromises()
    expect(wrapper.find('[data-test="converge-bad"]').text()).toContain('校验失败')
  })

  it('无集群降级：信封错误呈现告警而非崩溃（R08）', async () => {
    vi.spyOn(apiModule, 'listBusinessVlanServices').mockResolvedValue({
      data: { success: false, code: 503, message: '业务网络配置不可用：未连接 Kubernetes 集群（意图持久化载体）' },
    } as any)
    const wrapper = mountPage()
    await flushPromises()
    expect(wrapper.find('[data-test="business-unavailable"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('未连接 Kubernetes 集群')
  })

  it('新建抽屉：schema 驱动渲染意图字段（vlan-id/设备清单），缺实例名禁提交', async () => {
    vi.spyOn(apiModule, 'listBusinessVlanServices').mockResolvedValue({
      data: { success: true, data: { items: [] } },
    } as any)
    const wrapper = mountPage()
    await flushPromises()

    await wrapper.find('[data-test="business-create"]').trigger('click')
    await flushPromises()
    const text = wrapper.text()
    expect(text).toContain('业务 VLAN')
    expect(text).toContain('设备清单')

    const submit = wrapper.find('[data-test="business-submit"]')
    expect(submit.exists()).toBe(true)
    expect(submit.attributes('disabled')).toBeDefined() // 缺实例名 + 必填 vlan-id
  })

  it('编辑：spec 回填表单并按原名提交更新', async () => {
    vi.spyOn(apiModule, 'listBusinessVlanServices').mockResolvedValue({
      data: { success: true, data: { items: itemsFixture } },
    } as any)
    const applySpy = vi.spyOn(apiModule, 'applyBusinessVlanService').mockResolvedValue({
      data: { success: true, data: itemsFixture[0] },
    } as any)
    const wrapper = mountPage()
    await flushPromises()

    await wrapper.find('[data-test="business-edit-biz-100"]').trigger('click')
    await flushPromises()
    // el-input 透传未知属性到内部 input 本体。
    const nameInput = wrapper.find('input[data-test="business-name-input"], [data-test="business-name-input"] input, [data-test="business-name-input"]')
    expect((nameInput.element as HTMLInputElement).value).toBe('biz-100')
    expect((nameInput.element as HTMLInputElement).disabled).toBe(true) // 编辑不可改名（CR 名不可变）

    // 未修改任何字段时不可提交（diff 为空的权威门禁）。
    const submit = wrapper.find('[data-test="business-submit"]')
    expect(submit.attributes('disabled')).toBeDefined()

    // 修改业务名称后可提交，按原名调用更新。
    const nameField = wrapper.findAll('input').filter((i) => (i.element as HTMLInputElement).value === 'office')[0]
    await nameField.setValue('office-2')
    await flushPromises()
    expect(submit.attributes('disabled')).toBeUndefined()
    await submit.trigger('click')
    await flushPromises()
    expect(applySpy).toHaveBeenCalledWith('biz-100', expect.objectContaining({ 'vlan-id': 100, name: 'office-2' }))
  })

  it('详情抽屉：每设备状态与失败原因', async () => {
    vi.spyOn(apiModule, 'listBusinessVlanServices').mockResolvedValue({
      data: { success: true, data: { items: itemsFixture } },
    } as any)
    const wrapper = mountPage()
    await flushPromises()

    const detailBtns = wrapper.findAll('button').filter((b) => b.text() === '详情')
    await detailBtns[1].trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('device offline')
  })
})
