import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ElementPlus from 'element-plus'
import router from '../../src/router'
import Devices from '../../src/views/Devices.vue'
import { listDevices, getFleetReconcile, addDevice } from '../../src/api'

vi.mock('../../src/api')

// F3 —— 真 Chromium：添加设备对话框走真实 teleport/overlay 与 el-form 行内
// 错误渲染。happy-dom 下 el-dialog 会出现「ref 实例 ≠ 活实例」，validate() 因
// 字段列表为空而恒 resolve（实测），行内错误也不落到活 DOM——故行内错误态与
// 真实弹层交互只能在此层断言（§5.6 军规）。
// 提交闸门本身是纯函数（utils/deviceForm），F1 已全矩阵覆盖。

const devicesEnvelope = {
  data: {
    success: true,
    data: { devices: [{ ip: '192.168.1.1', port: 830, online: true }], stats: {} },
  },
}

function mountView() {
  return mount(Devices, {
    global: { plugins: [ElementPlus, createPinia(), router] },
    attachTo: document.body,
  })
}

describe('添加设备对话框（真浏览器）', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(listDevices).mockResolvedValue(devicesEnvelope as any)
    vi.mocked(getFleetReconcile).mockResolvedValue({ data: { success: true, data: { devices: [] } } } as any)
    vi.mocked(addDevice).mockResolvedValue({ data: { success: true } } as any)
    document.body.innerHTML = ''
  })

  async function openDialog() {
    const w = mountView()
    await flushPromises()
    await w.find('[data-test="add-device-btn"]').trigger('click')
    await flushPromises()
    await new Promise((r) => setTimeout(r, 300)) // 弹层过渡
    return w
  }

  it('对话框在真实 overlay 中可见且表单项可交互', async () => {
    await openDialog()
    const dialog = document.querySelector('[data-test="add-device-dialog"]') as HTMLElement
    expect(dialog).toBeTruthy()
    expect(dialog.getBoundingClientRect().height).toBeGreaterThan(0) // 真实可见，非零高度

    const ipInput = document.querySelector('[data-test="add-ip"] input') as HTMLInputElement
    ipInput.value = '7.225.21.14'
    ipInput.dispatchEvent(new Event('input', { bubbles: true }))
    await flushPromises()
    expect(ipInput.value).toBe('7.225.21.14')
  })

  it('空表单提交渲染行内错误（el-form 真实校验态）', async () => {
    await openDialog()
    const submit = document.querySelector('[data-test="add-submit"]') as HTMLElement
    submit.click()
    await flushPromises()
    await new Promise((r) => setTimeout(r, 300))

    const errors = document.querySelectorAll('.el-form-item__error')
    expect(errors.length).toBeGreaterThan(0) // IP/用户名/密码 必填提示
    expect(addDevice).not.toHaveBeenCalled()
  })

  it('填合法值后行内错误消失并可提交', async () => {
    await openDialog()
    for (const [id, val] of [
      ['add-ip', '7.225.21.14'],
      ['add-username', 'admin'],
      ['add-password', 'secret'],
    ] as const) {
      const input = document.querySelector(`[data-test="${id}"] input`) as HTMLInputElement
      input.value = val
      input.dispatchEvent(new Event('input', { bubbles: true }))
      input.dispatchEvent(new Event('blur', { bubbles: true }))
    }
    await flushPromises()
    await new Promise((r) => setTimeout(r, 200))

    const submit = document.querySelector('[data-test="add-submit"]') as HTMLElement
    submit.click()
    await flushPromises()
    expect(addDevice).toHaveBeenCalledWith(
      expect.objectContaining({ ip: '7.225.21.14', username: 'admin', password: 'secret' }),
    )
  })
})
