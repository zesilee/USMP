import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ElementPlus from 'element-plus'
import DryRunDialog from '../../src/components/config/DryRunDialog.vue'
import { useChangesetStore } from '../../src/stores/changeset'
import { previewChangeset } from '../../src/api'

vi.mock('../../src/api')

const DEV = '10.0.0.1'
const VLAN_PATH = '/vlan:vlan/vlan:vlans'

function seedOne() {
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

const previewOK = {
  data: {
    code: 0,
    success: true,
    data: {
      device: DEV,
      entries: [
        {
          op: 'update',
          path: VLAN_PATH,
          baseline_source: 'cache',
          forward_xml: '<vlan><id>10</id><description>x</description></vlan>',
          rollback_xml: '<vlan><id>10</id><description>old</description></vlan>',
          diff: [{ type: 'MODIFY', path: `${VLAN_PATH}/vlan[id=10]/description`, old: 'old', new: 'x' }],
        },
        {
          op: 'update',
          path: '/system:system',
          baseline_source: 'none',
          unsupported: true,
          unsupported_reason: '模块 system 无 XML 编码通道，不支持报文预览',
          diff: [],
        },
      ],
      summary: { adds: 0, deletes: 0, modifies: 1, total: 1 },
    },
  },
}

function mountDialog() {
  return mount(DryRunDialog, {
    props: { visible: true, device: DEV },
    global: { plugins: [ElementPlus] },
    attachTo: document.body,
  })
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.resetAllMocks()
})

describe('DryRunDialog · 试运行弹窗（FE-23，调 preview 接口）', () => {
  it('打开即调 preview（入参=store 序列化），Tab① 正向/回滚双栏 XML', async () => {
    seedOne()
    vi.mocked(previewChangeset).mockResolvedValue(previewOK as any)
    const w = mountDialog()
    await flushPromises()

    expect(previewChangeset).toHaveBeenCalledTimes(1)
    const req = vi.mocked(previewChangeset).mock.calls[0][0]
    expect(req.device).toBe(DEV)
    expect(req.entries).toHaveLength(1)

    const body = document.body.textContent!
    expect(body).toContain('正向报文')
    expect(body).toContain('回滚报文')
    expect(body).toContain('<description>x</description>')
    expect(body).toContain('<description>old</description>')
  })

  it('无 XML 通道条目：如实展示降级说明，不伪造报文（CS-03）', async () => {
    seedOne()
    vi.mocked(previewChangeset).mockResolvedValue(previewOK as any)
    const w = mountDialog()
    await flushPromises()
    expect(document.body.textContent).toContain('不支持报文预览')
  })

  it('Tab② 网元数据差异对比：diff 行 + 基线来源标注', async () => {
    seedOne()
    vi.mocked(previewChangeset).mockResolvedValue(previewOK as any)
    const w = mountDialog()
    await flushPromises()

    const tabs = document.body.querySelectorAll('.el-tabs__item')
    ;(Array.from(tabs).find((n) => n.textContent?.includes('网元数据差异对比')) as HTMLElement).click()
    await w.vm.$nextTick()

    const body = document.body.textContent!
    expect(body).toContain('description')
    expect(body).toContain('old')
    expect(body).toContain('缓存回读')
  })

  it('预览失败：如实展示后端错误、无任何报文内容（负路径）', async () => {
    seedOne()
    vi.mocked(previewChangeset).mockResolvedValue({
      data: { code: 400, success: false, message: '条目 0 (update /x): 解码失败' },
    } as any)
    const w = mountDialog()
    await flushPromises()
    const body = document.body.textContent!
    expect(body).toContain('解码失败')
    expect(body).not.toContain('正向报文')
  })

  it('接口异常（网络层 reject）：错误提示且不崩溃（R08）', async () => {
    seedOne()
    vi.mocked(previewChangeset).mockRejectedValue(new Error('network down'))
    const w = mountDialog()
    await flushPromises()
    expect(document.body.querySelector('[data-test="dryrun-error"]')).toBeTruthy()
  })
})
