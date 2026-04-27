import { describe, it, expect, vi, beforeEach } from 'vitest'
import InterfaceManager from '../components/interfaces/InterfaceManager.vue'
import { createTestWrapper, waitForUpdate } from './utils'
import { ElMessage } from 'element-plus'

// Mock API
vi.mock('../api', () => ({
  setConfig: vi.fn().mockResolvedValue({
    data: { success: true, message: '配置下发成功' }
  }),
  getConfig: vi.fn().mockResolvedValue({
    data: {
      success: true,
      data: {
        interface: {
          'GigabitEthernet0/0': {
            name: 'GigabitEthernet0/0',
            config: {
              name: 'GigabitEthernet0/0',
              type: 'PHYSICAL',
              mtu: 1500,
              enabled: true,
              description: 'Uplink port'
            }
          }
        }
      }
    }
  })
}))

// Mock YangRenderer 组件
vi.mock('../components/yang/YangRenderer.vue', () => ({
  default: {
    template: '<div class="yang-renderer-mock">Yang Renderer</div>',
    methods: {
      loadData: vi.fn()
    },
    data() {
      return {
        formData: {}
      }
    }
  }
}))

vi.mock('element-plus', async () => {
  const actual = await vi.importActual('element-plus')
  return {
    ...actual,
    ElMessage: {
      success: vi.fn(),
      error: vi.fn()
    }
  }
})

describe('InterfaceManager', () => {
  const deviceIp = '192.168.1.1'

  const createWrapper = (props = {}) => {
    return createTestWrapper(InterfaceManager, {
      props: {
        deviceIp,
        ...props
      }
    })
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('应正确渲染标题和设备 IP', async () => {
    const wrapper = createWrapper()
    await waitForUpdate(wrapper)

    expect(wrapper.text()).toContain('接口配置管理')
    expect(wrapper.text()).toContain(deviceIp)
  })

  it('应正确渲染刷新按钮', async () => {
    const wrapper = createWrapper()
    await waitForUpdate(wrapper)

    const refreshButton = wrapper.find('button')
    expect(refreshButton.exists()).toBe(true)
    expect(refreshButton.text()).toContain('刷新')
  })

  it('应正确渲染下发配置按钮', async () => {
    const wrapper = createWrapper()
    await waitForUpdate(wrapper)

    const submitButton = wrapper.find('.el-button--primary')
    expect(submitButton.exists()).toBe(true)
    expect(submitButton.text()).toContain('下发配置')
  })

  it('应渲染 YangRenderer 组件', async () => {
    const wrapper = createWrapper()
    await waitForUpdate(wrapper)

    const renderer = wrapper.find('.yang-renderer-mock')
    expect(renderer.exists()).toBe(true)
  })

  it('工具栏应包含两个按钮', async () => {
    const wrapper = createWrapper()
    await waitForUpdate(wrapper)

    const buttons = wrapper.findAll('button')
    expect(buttons.length).toBeGreaterThanOrEqual(2)
  })

  it('点击下发配置按钮应调用 setConfig API', async () => {
    const { setConfig } = await import('../api')
    const wrapper = createWrapper()
    await waitForUpdate(wrapper)

    const submitButton = wrapper.find('.el-button--primary')
    await submitButton.trigger('click')

    expect(setConfig).toHaveBeenCalledWith(deviceIp, '/interfaces', expect.any(Object))
  })

  it('配置下发成功应显示成功消息', async () => {
    const wrapper = createWrapper()
    await waitForUpdate(wrapper)

    const submitButton = wrapper.find('.el-button--primary')
    await submitButton.trigger('click')

    expect(ElMessage.success).toHaveBeenCalledWith('配置下发成功')
  })

  it('配置下发失败应显示错误消息', async () => {
    const { setConfig } = await import('../api')
    vi.mocked(setConfig).mockRejectedValueOnce(new Error('Network Error'))

    const wrapper = createWrapper()
    await waitForUpdate(wrapper)

    const submitButton = wrapper.find('.el-button--primary')
    await submitButton.trigger('click')

    expect(ElMessage.error).toHaveBeenCalledWith('Network Error')
  })
})
