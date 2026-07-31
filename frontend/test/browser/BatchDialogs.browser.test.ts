import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ElementPlus from 'element-plus'
import BatchToolbar from '../../src/components/config/BatchToolbar.vue'
import DryRunDialog from '../../src/components/config/DryRunDialog.vue'
import { useChangesetStore } from '../../src/stores/changeset'
import { previewChangeset } from '../../src/api'

vi.mock('../../src/api')

// F3（FE-23）——真 Chromium：工具栏弹窗开合与试运行 Tab 切换走真实
// teleport/overlay（happy-dom 对 el-dialog/el-tabs 的伪造最重区域，§5.6 军规）。

const DEV = '10.0.0.1'
const VLAN_PATH = '/vlan:vlan/vlan:vlans'

function seed(device = DEV) {
  const s = useChangesetStore()
  s.upsert(device, {
    op: 'update',
    path: VLAN_PATH,
    listKey: 'vlan',
    keyValue: '10',
    payload: { id: 10, description: 'after-x' },
    cleared: [],
    baseline: { id: 10, description: 'before-x' },
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
          baseline_source: 'desired',
          forward_xml: '<vlan><id>10</id><description>after-x</description></vlan>',
          rollback_xml: '<vlan><id>10</id><description>before-x</description></vlan>',
          diff: [{ type: 'MODIFY', path: `${VLAN_PATH}/vlan[id=10]/description`, old: 'before-x', new: 'after-x' }],
        },
      ],
      summary: { adds: 0, deletes: 0, modifies: 1, total: 1 },
    },
  },
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.resetAllMocks()
  document.body.innerHTML = ''
})

describe('BatchToolbar F3 · 真浏览器弹窗开合（FE-23）', () => {
  it('变更内容弹窗：点击打开（真实 teleport 到 body）、树表内容可见、关闭收起', async () => {
    seed()
    const w = mount(BatchToolbar, {
      props: { device: DEV },
      global: { plugins: [ElementPlus] },
      attachTo: document.body,
    })
    await w.vm.$nextTick()
    await w.find('[data-test="batch-changes"]').trigger('click')
    await flushPromises()
    await w.vm.$nextTick()

    const dialog = document.body.querySelector('.el-dialog')
    expect(dialog).toBeTruthy()
    expect(document.body.textContent).toContain('vlan 10')
    expect(document.body.textContent).toContain('before-x')
    expect(document.body.textContent).toContain('after-x')

    const closeBtn = document.body.querySelector('.el-dialog__headerbtn') as HTMLElement
    closeBtn.click()
    await w.vm.$nextTick()
    await flushPromises()
    const visibleDialog = Array.from(document.body.querySelectorAll('.el-overlay')).find(
      (n) => (n as HTMLElement).style.display !== 'none',
    )
    expect(visibleDialog).toBeFalsy()
    w.unmount()
  })

  it('试运行弹窗：打开调 preview、双栏报文可见、真实 Tab 切换到差异对比', async () => {
    seed()
    vi.mocked(previewChangeset).mockResolvedValue(previewOK as any)
    const w = mount(DryRunDialog, {
      props: { visible: true, device: DEV },
      global: { plugins: [ElementPlus] },
      attachTo: document.body,
    })
    await flushPromises()

    expect(previewChangeset).toHaveBeenCalledTimes(1)
    expect(document.body.textContent).toContain('正向报文')
    expect(document.body.textContent).toContain('after-x')

    const diffTab = Array.from(document.body.querySelectorAll('.el-tabs__item')).find((n) =>
      n.textContent?.includes('网元数据差异对比'),
    ) as HTMLElement
    expect(diffTab).toBeTruthy()
    diffTab.click()
    await flushPromises()
    await w.vm.$nextTick()

    expect(document.body.textContent).toContain('before-x')
    expect(document.body.textContent).toContain('控制器期望配置')
    w.unmount()
  })
})
