import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ElementPlus, { ElMessageBox } from 'element-plus'
import BatchToolbar from '../../src/components/config/BatchToolbar.vue'
import { useChangesetStore } from '../../src/stores/changeset'

const DEV_A = '10.0.0.1'
const DEV_B = '10.0.0.2'
const VLAN_PATH = '/vlan:vlan/vlan:vlans'

// 弹窗为独立组件（4.3/4.5 另测），此处 stub 只断言开合状态。
const stubs = {
  ChangesContentDialog: { template: '<div data-test="stub-changes" :data-open="visible" />', props: ['visible', 'device'] },
  DryRunDialog: { template: '<div data-test="stub-dryrun" :data-open="visible" />', props: ['visible', 'device'] },
}

function seedEntry(device: string, keyValue: string, op: 'create' | 'update' = 'update') {
  const s = useChangesetStore()
  s.upsert(device, {
    op,
    path: VLAN_PATH,
    listKey: 'vlan',
    keyValue,
    payload: { id: Number(keyValue) },
    cleared: [],
    baseline: null,
    label: `vlan ${keyValue}`,
  })
}

function mountToolbar(device = DEV_A) {
  return mount(BatchToolbar, {
    props: { device },
    global: { plugins: [ElementPlus], stubs },
  })
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.restoreAllMocks()
})

describe('BatchToolbar · 攒批工具栏（FE-23）', () => {
  it('空变更集：四按钮渲染、后三者禁用、无徽标、无提示条', () => {
    const w = mountToolbar()
    for (const t of ['batch-changes', 'batch-dryrun', 'batch-reset', 'batch-commit']) {
      expect(w.find(`[data-test="${t}"]`).exists(), t).toBe(true)
    }
    expect(w.find('[data-test="batch-dryrun"]').attributes('disabled')).toBeDefined()
    expect(w.find('[data-test="batch-reset"]').attributes('disabled')).toBeDefined()
    expect(w.find('[data-test="batch-commit"]').attributes('disabled')).toBeDefined()
    expect(w.find('[data-test="batch-changes"]').attributes('disabled')).toBeUndefined()
    expect(w.find('.el-badge__content').exists()).toBe(false)
    expect(w.find('[data-test="batch-hint"]').exists()).toBe(false)
  })

  it('有变更：徽标计数、按钮解禁、提示条出现（FE-23 联动）', async () => {
    seedEntry(DEV_A, '10')
    seedEntry(DEV_A, '20', 'create')
    const w = mountToolbar()
    await w.vm.$nextTick()
    expect(w.find('.el-badge__content').text()).toBe('2')
    expect(w.find('[data-test="batch-dryrun"]').attributes('disabled')).toBeUndefined()
    expect(w.find('[data-test="batch-commit"]').attributes('disabled')).toBeUndefined()
    expect(w.find('[data-test="batch-hint"]').exists()).toBe(true)
  })

  it('提示条可关闭；清空后再攒新变更 → 提示条重新出现', async () => {
    seedEntry(DEV_A, '10')
    const w = mountToolbar()
    await w.vm.$nextTick()
    await w.find('[data-test="batch-hint"] .el-alert__close-btn').trigger('click')
    expect(w.find('[data-test="batch-hint"]').exists()).toBe(false)

    const s = useChangesetStore()
    s.clear(DEV_A)
    await w.vm.$nextTick()
    seedEntry(DEV_A, '30')
    await w.vm.$nextTick()
    expect(w.find('[data-test="batch-hint"]').exists()).toBe(true)
  })

  it('切设备隔离：徽标随 device prop 切换（FE-23）', async () => {
    seedEntry(DEV_A, '10')
    seedEntry(DEV_A, '20')
    seedEntry(DEV_B, '30')
    const w = mountToolbar(DEV_A)
    await w.vm.$nextTick()
    expect(w.find('.el-badge__content').text()).toBe('2')
    await w.setProps({ device: DEV_B })
    expect(w.find('.el-badge__content').text()).toBe('1')
  })

  it('变更内容/试运行点击 → 各自弹窗打开', async () => {
    seedEntry(DEV_A, '10')
    const w = mountToolbar()
    await w.vm.$nextTick()
    await w.find('[data-test="batch-changes"]').trigger('click')
    expect(w.find('[data-test="stub-changes"]').attributes('data-open')).toBe('true')
    await w.find('[data-test="batch-dryrun"]').trigger('click')
    expect(w.find('[data-test="stub-dryrun"]').attributes('data-open')).toBe('true')
  })

  it('重置：确认后清空当前设备并 emit reset；取消保留（FE-23）', async () => {
    seedEntry(DEV_A, '10')
    seedEntry(DEV_B, '30')
    const w = mountToolbar()
    await w.vm.$nextTick()

    vi.spyOn(ElMessageBox, 'confirm').mockRejectedValueOnce('cancel')
    await w.find('[data-test="batch-reset"]').trigger('click')
    await new Promise((r) => setTimeout(r))
    expect(useChangesetStore().countFor(DEV_A)).toBe(1)
    expect(w.emitted('reset')).toBeUndefined()

    vi.spyOn(ElMessageBox, 'confirm').mockResolvedValueOnce('confirm')
    await w.find('[data-test="batch-reset"]').trigger('click')
    await new Promise((r) => setTimeout(r))
    expect(useChangesetStore().countFor(DEV_A)).toBe(0)
    expect(useChangesetStore().countFor(DEV_B)).toBe(1, )
    expect(w.emitted('reset')).toHaveLength(1)
  })

  it('提交配置点击 → 仅 emit commit-request（编排由页面层承担，PR-5）', async () => {
    seedEntry(DEV_A, '10')
    const w = mountToolbar()
    await w.vm.$nextTick()
    await w.find('[data-test="batch-commit"]').trigger('click')
    expect(w.emitted('commit-request')).toHaveLength(1)
  })
})
