import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import ElementPlus, { ElMessageBox } from 'element-plus'
import RpcExecuteTab from '../../src/components/config/RpcExecuteTab.vue'
import { executeRpc, getConfig } from '../../src/api'
import { deriveRpcTabs } from '../../src/utils/moduleConsole'

vi.mock('../../src/api')

const rpcs = [
  {
    name: 'reset-if-counters-by-name',
    label: '按接口名清除统计',
    highRisk: false,
    input: [{ path: 'if-name', type: 'string' as const, label: 'if-name', required: true, leafRef: '/ifm/interfaces/interface/name' }],
  },
  { name: 'restart-if', label: '重启接口', highRisk: true, input: [] as any[] },
]
const resetTab = deriveRpcTabs(rpcs)[0]
const restartTab = deriveRpcTabs(rpcs)[1]

function mountTab(tab = resetTab, device = '10.0.0.1') {
  return mount(RpcExecuteTab, {
    props: { tab, module: 'ifm', device },
    global: { plugins: [createPinia(), ElementPlus] },
  })
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(executeRpc).mockResolvedValue({ data: { success: true, data: { ok: true, reply: '', highRisk: false } } } as any)
  vi.mocked(getConfig).mockResolvedValue({ data: { data: {} } } as any)
})

describe('RpcExecuteTab（FE-19/20）', () => {
  it('渲染 rpc 输入字段', () => {
    const w = mountTab()
    expect(w.text()).toContain('按接口名清除统计')
    expect(w.find('input').exists()).toBe(true)
  })

  it('缺 mandatory input 时执行按钮禁用（§9 校验拦截）', async () => {
    const w = mountTab()
    const btn = w.find('[data-test="rpc-execute"]')
    expect((btn.element as HTMLButtonElement).disabled).toBe(true)
    // 填入后启用
    ;(w.vm as any).values['if-name'] = '200GE0/1/0'
    await flushPromises()
    expect((btn.element as HTMLButtonElement).disabled).toBe(false)
  })

  it('高危 rpc 显示高危标签', () => {
    const w = mountTab(restartTab)
    expect(w.find('[data-test="rpc-highrisk"]').exists()).toBe(true)
    // 非高危不显示
    expect(mountTab(resetTab).find('[data-test="rpc-highrisk"]').exists()).toBe(false)
  })

  it('执行前二次确认；确认后调执行 API 并回显结果', async () => {
    const confirm = vi.spyOn(ElMessageBox, 'confirm').mockResolvedValue('confirm' as any)
    const w = mountTab()
    ;(w.vm as any).values['if-name'] = '200GE0/1/0'
    await flushPromises()

    await w.find('[data-test="rpc-execute"]').trigger('click')
    await flushPromises()

    expect(confirm).toHaveBeenCalledTimes(1)
    expect(vi.mocked(executeRpc)).toHaveBeenCalledWith('10.0.0.1', 'ifm', 'reset-if-counters-by-name', { 'if-name': '200GE0/1/0' })
    expect(w.find('[data-test="rpc-result"]').exists()).toBe(true)
  })

  it('取消确认 → 不下发到设备', async () => {
    const confirm = vi.spyOn(ElMessageBox, 'confirm').mockRejectedValue('cancel')
    const w = mountTab()
    ;(w.vm as any).values['if-name'] = '200GE0/1/0'
    await flushPromises()

    await w.find('[data-test="rpc-execute"]').trigger('click')
    await flushPromises()

    expect(confirm).toHaveBeenCalled()
    expect(vi.mocked(executeRpc)).not.toHaveBeenCalled()
  })

  it('执行失败回显错误', async () => {
    vi.spyOn(ElMessageBox, 'confirm').mockResolvedValue('confirm' as any)
    vi.mocked(executeRpc).mockResolvedValue({ data: { success: false, message: 'interface busy' } } as any)
    const w = mountTab()
    ;(w.vm as any).values['if-name'] = '200GE0/1/0'
    await flushPromises()

    await w.find('[data-test="rpc-execute"]').trigger('click')
    await flushPromises()

    const alert = w.find('[data-test="rpc-result"]')
    expect(alert.exists()).toBe(true)
    expect(alert.text()).toContain('interface busy')
  })
})

describe('RpcExecuteTab · leafref 下拉（FE-19）', () => {
  it('leafref 输入解析目标列表 → 渲染为可搜索下拉，含接口名选项', async () => {
    vi.mocked(getConfig).mockResolvedValue({
      data: { data: { interface: [{ name: '200GE0/1/0' }, { name: '200GE0/1/1' }] } },
    } as any)
    const w = mountTab(resetTab)
    await flushPromises()

    // 拉取路径由 leafRef 解析：/ifm/interfaces/interface/name → /ifm/interfaces
    expect(vi.mocked(getConfig)).toHaveBeenCalledWith('10.0.0.1', '/ifm/interfaces')
    // if-name 渲染为 leafref 下拉（非文本框）
    expect(w.find('[data-test="leafref-select"]').exists()).toBe(true)
    // 选项来自设备接口列表
    const vm = w.vm as any
    expect(vm.resolvedInputs[0].options.map((o: any) => o.value)).toEqual(['200GE0/1/0', '200GE0/1/1'])
  })

  it('拉取失败/空列表 → 降级文本输入（R08，不阻断执行）', async () => {
    vi.mocked(getConfig).mockRejectedValue(new Error('offline'))
    const w = mountTab(resetTab)
    await flushPromises()
    expect(w.find('[data-test="leafref-select"]').exists()).toBe(false)
    expect(w.find('input').exists()).toBe(true) // 文本框兜底
  })
})
