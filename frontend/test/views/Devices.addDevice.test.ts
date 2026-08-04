import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import router from '../../src/router'
import Devices from '../../src/views/Devices.vue'
import ElementPlus, { ElMessage } from 'element-plus'
import { listDevices, getFleetReconcile, addDevice, removeDevice } from '../../src/api'

vi.mock('../../src/api')
vi.mock('element-plus', async () => {
  const actual = await vi.importActual<typeof import('element-plus')>('element-plus')
  return {
    ...actual,
    ElMessage: Object.assign(vi.fn(), { success: vi.fn(), error: vi.fn(), warning: vi.fn(), info: vi.fn() }),
  }
})

const devicesEnvelope = {
  data: {
    success: true,
    data: {
      devices: [{ ip: '192.168.1.1', port: 830, online: true }],
      stats: { active_connections: 1, total_connections: 1, errors: 0 },
    },
  },
}
const fleetEnvelope = { data: { success: true, data: { devices: [] } } }

let pinia: ReturnType<typeof createPinia>
function mountView() {
  return mount(Devices, { global: { plugins: [ElementPlus, pinia, router] } })
}

// 查询约定：模板内的 el-dialog 在 test-utils 下留在 wrapper DOM 内；命令式的
// ElMessageBox 挂到 document.body。故先查 wrapper，再兜底查 body。
let wrapper: ReturnType<typeof mountView>

function el<T extends Element>(selector: string): T | null {
  return (wrapper?.element as HTMLElement)?.querySelector<T>(selector) ?? document.body.querySelector<T>(selector)
}

function bodyEl<T extends Element>(selector: string): T | null {
  return document.body.querySelector<T>(selector)
}

async function openDialog() {
  wrapper = mountView()
  await flushPromises()
  await wrapper.find('[data-test="add-device-btn"]').trigger('click')
  await flushPromises()
  return wrapper
}

async function setInput(testId: string, value: string) {
  const input = el<HTMLInputElement>(`[data-test="${testId}"] input`)
  expect(input, `缺输入框 ${testId}`).toBeTruthy()
  input!.value = value
  input!.dispatchEvent(new Event('input'))
  await flushPromises()
}

describe('Devices View · 添加/删除设备', () => {
  beforeEach(() => {
    pinia = createPinia()
    setActivePinia(pinia)
    vi.clearAllMocks()
    vi.mocked(listDevices).mockResolvedValue(devicesEnvelope as any)
    vi.mocked(getFleetReconcile).mockResolvedValue(fleetEnvelope as any)
    vi.mocked(addDevice).mockResolvedValue({ data: { success: true } } as any)
    vi.mocked(removeDevice).mockResolvedValue({ data: { success: true } } as any)
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('点「添加设备」弹出表单对话框', async () => {
    await openDialog()
    expect(el('[data-test="add-device-dialog"]')).toBeTruthy()
    expect(el('[data-test="add-ip"] input')).toBeTruthy()
    expect(el('[data-test="add-username"] input')).toBeTruthy()
    expect(el('[data-test="add-password"] input')).toBeTruthy()
  })

  it('空表单提交被校验拦截，不发请求且提示原因', async () => {
    await openDialog()
    el<HTMLButtonElement>('[data-test="add-submit"]')!.click()
    await flushPromises()
    expect(addDevice).not.toHaveBeenCalled()
    // 行内错误样式属 el-form/teleport 范畴，真浏览器断言见 F3
    // （test/browser/DeviceAddDialog.browser.test.ts）；此处断可靠的 toast 反馈。
    expect(ElMessage.error).toHaveBeenCalled()
  })

  it('非法 IP 被校验拦截', async () => {
    await openDialog()
    await setInput('add-ip', 'not-an-ip')
    await setInput('add-username', 'admin')
    await setInput('add-password', 'admin')
    el<HTMLButtonElement>('[data-test="add-submit"]')!.click()
    await flushPromises()
    expect(addDevice).not.toHaveBeenCalled()
  })

  it('合法表单提交调用 API 并刷新列表', async () => {
    await openDialog()
    await setInput('add-ip', '7.225.21.14')
    await setInput('add-port', '830')
    await setInput('add-username', 'admin')
    await setInput('add-password', 'secret')
    const callsBefore = vi.mocked(listDevices).mock.calls.length
    el<HTMLButtonElement>('[data-test="add-submit"]')!.click()
    await flushPromises()
    expect(addDevice).toHaveBeenCalledWith(
      expect.objectContaining({ ip: '7.225.21.14', port: 830, username: 'admin', password: 'secret' }),
    )
    expect(vi.mocked(listDevices).mock.calls.length).toBeGreaterThan(callsBefore) // 刷新
  })

  it('后端拒绝时展示错误且对话框保留', async () => {
    vi.mocked(addDevice).mockRejectedValue({ response: { data: { message: 'unsupported vendor' } } })
    await openDialog()
    await setInput('add-ip', '7.225.21.14')
    await setInput('add-username', 'admin')
    await setInput('add-password', 'secret')
    el<HTMLButtonElement>('[data-test="add-submit"]')!.click()
    await flushPromises()
    expect(el('[data-test="add-device-dialog"]')).toBeTruthy() // 不关框，可改重试
  })

  it('行内删除经确认后调用 API', async () => {
    wrapper = mountView()
    await flushPromises()
    await wrapper.find('[data-test="device-delete-btn"]').trigger('click')
    await flushPromises()
    const confirmBtn = bodyEl<HTMLButtonElement>('.el-message-box__btns .el-button--primary')
    expect(confirmBtn, '缺确认弹窗').toBeTruthy()
    confirmBtn!.click()
    await flushPromises()
    expect(removeDevice).toHaveBeenCalledWith('192.168.1.1')
  })
})
