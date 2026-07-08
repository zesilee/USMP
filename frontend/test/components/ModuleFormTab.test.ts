import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import ModuleFormTab from '../../src/components/config/ModuleFormTab.vue'
import { getConfig, setConfig } from '../../src/api'
import { deriveTabs } from '../../src/utils/moduleConsole'
import { ifmNestedSchema } from '../views/moduleConsole.fixture'

vi.mock('../../src/api')

const globalTab = deriveTabs(ifmNestedSchema.fields).find((t) => t.name === 'global')!

function mountTab() {
  return mount(ModuleFormTab, {
    props: { tab: globalTab, rootName: 'ifm', device: '10.0.0.1' },
    global: { plugins: [createPinia(), ElementPlus] },
  })
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(getConfig).mockResolvedValue({ data: { data: { data: {} } } } as any)
  vi.mocked(setConfig).mockResolvedValue({ data: { data: {} } } as any)
})

describe('ModuleFormTab · presence 容器（FE-12）', () => {
  it('presence group 渲染为开关；ignore-primary-sub=true 时 must 不满足 → 禁用并强制关', async () => {
    const w = mountTab()
    await flushPromises()
    const vm = w.vm as any

    // must 满足（false）：开关可用，可开启
    vm.form.formData['ipv4-ignore-primary-sub'] = false
    await flushPromises()
    const conflictField = vm.form.visibleFields.value.find((f: any) => f.label === 'ipv4-conflict-enable')
    expect(conflictField?.presence).toBe(true)
    expect(vm.presenceBlocked(conflictField)).toBe(false)
    vm.form.formData['ipv4-conflict-enable'] = {}
    await flushPromises()
    expect(vm.form.visiblePayload()['ipv4-conflict-enable']).toEqual({})

    // 切 true：must 违例 → 开关禁用 + 键被强制删除（节点不存在）
    vm.form.formData['ipv4-ignore-primary-sub'] = true
    await flushPromises()
    expect(vm.presenceBlocked(conflictField)).toBe(true)
    expect(vm.form.formData['ipv4-conflict-enable']).toBeUndefined()
    expect('ipv4-conflict-enable' in vm.form.visiblePayload()).toBe(false)

    // DOM 证据：该表单项内的 switch 处于禁用态
    const switches = w.findAll('.el-switch.is-disabled')
    expect(switches.length).toBeGreaterThan(0)
  })

  it('must 表达式非法时降级为可用（R08）', async () => {
    const brokenTab = JSON.parse(JSON.stringify(globalTab))
    brokenTab.field.fields.find((f: any) => f.label === 'ipv4-conflict-enable').must = [{ expr: '((' }]
    const w = mount(ModuleFormTab, {
      props: { tab: brokenTab, rootName: 'ifm', device: '10.0.0.1' },
      global: { plugins: [createPinia(), ElementPlus] },
    })
    await flushPromises()
    const vm = w.vm as any
    const f = vm.form.visibleFields.value.find((x: any) => x.label === 'ipv4-conflict-enable')
    expect(vm.presenceBlocked(f)).toBe(false)
  })
})

describe('ModuleFormTab · 全局属性校验（statistic-interval，FE-11/FE-07）', () => {
  it('must (interval mod 10 = 0)：违例阻断提交、合规放行', async () => {
    const w = mountTab()
    await flushPromises()
    const vm = w.vm as any

    vm.form.formData['statistic-interval'] = 305
    await flushPromises()
    expect(vm.form.mustViolations.value.length).toBeGreaterThan(0)
    expect(vm.form.submittable.value).toBe(false)
    await vm.submit()
    expect(setConfig).not.toHaveBeenCalled()

    vm.form.formData['statistic-interval'] = 300
    await flushPromises()
    expect(vm.form.mustViolations.value).toHaveLength(0)
    expect(vm.form.submittable.value).toBe(true)
    await vm.submit()
    expect(setConfig).toHaveBeenCalledWith('10.0.0.1', 'ifm:ifm/ifm:global', expect.objectContaining({ 'statistic-interval': 300 }))
  })

  it('回读回填：GET 值 seed 进表单；GET 失败降级为空表单 + 告警（§9）', async () => {
    vi.mocked(getConfig).mockResolvedValue({
      data: { data: { data: { 'statistic-interval': 120, 'ipv4-ignore-primary-sub': false } } },
    } as any)
    const w = mountTab()
    await flushPromises()
    expect((w.vm as any).form.formData['statistic-interval']).toBe(120)

    vi.mocked(getConfig).mockRejectedValue(new Error('unsupported path'))
    const w2 = mountTab()
    await flushPromises()
    expect(w2.find('.el-alert').exists()).toBe(true)
    expect((w2.vm as any).form.formData['statistic-interval']).toBeUndefined()
  })

  it('下发失败原样透出后端错误（§9，不伪装成功）', async () => {
    vi.mocked(setConfig).mockRejectedValue({ response: { data: { message: 'no converter for path' } } })
    const w = mountTab()
    await flushPromises()
    const vm = w.vm as any
    vm.form.formData['statistic-interval'] = 300
    await flushPromises()
    await vm.submit()
    await flushPromises()
    expect(w.text()).toContain('no converter for path')
  })
})

describe('ModuleFormTab · readonly 降级（FE-14）', () => {
  const mixedTab = deriveTabs([
    {
      path: '/ifm/mixed',
      type: 'group' as const,
      label: 'mixed',
      fields: [
        { path: '/ifm/mixed/mtu', type: 'number' as const, label: 'mtu' },
        { path: '/ifm/mixed/oper-status', type: 'string' as const, label: 'oper-status', readonly: true },
      ],
    },
  ])[0]

  it('混合容器内 readonly 叶：渲染禁用态、不入 diff/payload、无必填规则', async () => {
    vi.mocked(getConfig).mockResolvedValue({
      data: { data: { data: { mtu: 1500, 'oper-status': 'up' } } },
    } as any)
    const w = mount(ModuleFormTab, {
      props: { tab: mixedTab, rootName: 'ifm', device: '10.0.0.1' },
      global: { plugins: [createPinia(), ElementPlus] },
    })
    await flushPromises()
    const vm = w.vm as any

    // readonly 叶可见但禁用（回显设备 state 值）
    const inputs = w.findAll('.el-form-item')
    const operItem = inputs.find((n) => n.text().includes('oper-status'))!
    expect(operItem.find('input').attributes('disabled')).toBeDefined()

    // 改可编辑叶后：diff/payload 只含可编辑键，readonly 键不入
    vm.form.formData['mtu'] = 9000
    await flushPromises()
    const payload = vm.form.visiblePayload()
    expect(payload['mtu']).toBe(9000)
    expect('oper-status' in payload).toBe(false)
    expect(vm.form.diff.value.every((d: any) => d.key !== 'oper-status')).toBe(true)
    expect(vm.form.rules.value['oper-status']).toBeUndefined()
  })

  it('整 Tab readonly：无「下发」按钮', async () => {
    const roTab = deriveTabs([
      {
        path: '/ifm/ipv4-interface-count',
        type: 'group' as const,
        label: 'ipv4-interface-count',
        readonly: true,
        fields: [
          { path: '/ifm/ipv4-interface-count/protocol-up-count', type: 'number' as const, label: 'protocol-up-count', readonly: true },
        ],
      },
    ])[0]
    vi.mocked(getConfig).mockResolvedValue({ data: { data: { data: { 'protocol-up-count': 3 } } } } as any)
    const w = mount(ModuleFormTab, {
      props: { tab: roTab, rootName: 'ifm', device: '10.0.0.1' },
      global: { plugins: [createPinia(), ElementPlus] },
    })
    await flushPromises()
    expect(w.text()).not.toContain('下发')
    // state 值照常回显
    expect((w.vm as any).form.formData['protocol-up-count']).toBe(3)
  })
})

describe('ModuleFormTab · dynamicDefault 空值语义（FE-15）', () => {
  const dynTab = deriveTabs([
    {
      path: '/ifm/dyncfg',
      type: 'group' as const,
      label: 'dyncfg',
      fields: [
        { path: '/ifm/dyncfg/mtu', type: 'number' as const, label: 'mtu' },
        { path: '/ifm/dyncfg/admin-status', type: 'string' as const, label: 'admin-status', required: true, dynamicDefault: true },
      ],
    },
  ])[0]

  it('空值不入 payload/不报必填；显式覆写正常入 payload（边界）', async () => {
    vi.mocked(getConfig).mockResolvedValue({ data: { data: { data: {} } } } as any)
    const w = mount(ModuleFormTab, {
      props: { tab: dynTab, rootName: 'ifm', device: '10.0.0.1' },
      global: { plugins: [createPinia(), ElementPlus] },
    })
    await flushPromises()
    const vm = w.vm as any

    // 空值：required+dynamicDefault 不拦截（blocked=false 前提是有其它有效改动）
    vm.form.formData['mtu'] = 9000
    vm.form.formData['admin-status'] = ''
    await flushPromises()
    expect(vm.form.blocked.value).toBe(false)
    let payload = vm.form.visiblePayload()
    expect(payload['mtu']).toBe(9000)
    expect('admin-status' in payload).toBe(false)

    // 显式覆写：正常进入 payload
    vm.form.formData['admin-status'] = 'down'
    await flushPromises()
    payload = vm.form.visiblePayload()
    expect(payload['admin-status']).toBe('down')
  })
})
