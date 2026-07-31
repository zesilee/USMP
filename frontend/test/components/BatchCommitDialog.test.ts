import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia, type Pinia } from 'pinia'
import ElementPlus from 'element-plus'
import BatchCommitDialog from '../../src/components/config/BatchCommitDialog.vue'
import { useChangesetStore } from '../../src/stores/changeset'
import { commitChangeset, getConfig, getDeviceReconcile } from '../../src/api'

vi.mock('../../src/api')

const DEV = '10.0.0.1'
const VLAN_PATH = '/vlan:vlan/vlan:vlans'

let pinia: Pinia

function seed() {
  useChangesetStore().upsert(DEV, {
    op: 'update',
    path: VLAN_PATH,
    listKey: 'vlan',
    keyValue: '10',
    payload: { id: 10, description: 'x' },
    cleared: [],
    baseline: null,
    label: 'vlan 10',
  })
}

function mountDialog() {
  return mount(BatchCommitDialog, {
    props: { visible: true, device: DEV },
    global: { plugins: [pinia, ElementPlus] },
    attachTo: document.body,
  })
}

beforeEach(() => {
  pinia = createPinia()
  setActivePinia(pinia)
  vi.resetAllMocks()
  vi.mocked(getConfig).mockResolvedValue({ data: { data: {} } } as any)
})

describe('BatchCommitDialog · 提交进度弹窗（FE-03 攒批）', () => {
  it('打开即执行提交编排：成功链 emit committed、关闭按钮解禁', async () => {
    seed()
    vi.mocked(commitChangeset).mockResolvedValue({ data: { code: 0, success: true, data: {} } } as any)
    vi.mocked(getDeviceReconcile)
      .mockResolvedValueOnce({ data: { data: { statuses: [] } } } as any)
      .mockResolvedValue({
        data: { data: { statuses: [{ path: 'vlan:vlan/vlan:vlans', last_run: '2026-07-31T12:00:00Z', outcome: 'converged' }] } },
      } as any)

    const w = mountDialog()
    await flushPromises()
    await flushPromises()

    expect(vi.mocked(commitChangeset)).toHaveBeenCalledTimes(1)
    expect(w.emitted('committed')).toHaveLength(1)
    expect(useChangesetStore().countFor(DEV)).toBe(0)
    const closeBtn = document.body.querySelector('[data-test="commit-close"]') as HTMLButtonElement
    expect(closeBtn.disabled).toBe(false)
  })

  it('提交失败：如实展示错误、不 emit committed、变更集保留（R08/§9）', async () => {
    seed()
    vi.mocked(commitChangeset).mockResolvedValue({
      data: { code: 502, success: false, message: '提交失败（设备已整体回退）: rejected' },
    } as any)
    vi.mocked(getDeviceReconcile).mockResolvedValue({ data: { data: { statuses: [] } } } as any)

    const w = mountDialog()
    await flushPromises()

    expect(w.emitted('committed')).toBeUndefined()
    expect(document.body.querySelector('[data-test="commit-error"]')).toBeTruthy()
    expect(document.body.textContent).toContain('整体回退')
    expect(useChangesetStore().countFor(DEV)).toBe(1)
  })

  it('空变更集打开：静默收起（update:visible false），零请求', async () => {
    const w = mountDialog()
    await flushPromises()
    expect(vi.mocked(commitChangeset)).not.toHaveBeenCalled()
    expect(w.emitted('update:visible')?.some((e) => e[0] === false)).toBe(true)
  })
})
