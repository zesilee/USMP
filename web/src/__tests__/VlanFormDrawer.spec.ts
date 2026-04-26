import { describe, it, expect, vi } from 'vitest'
import { nextTick } from 'vue'
import VlanFormDrawer from '../components/vlan/VlanFormDrawer.vue'
import { createTestWrapper, waitForUpdate, simulateInput } from './utils'
import type { VlanFormData } from '../types/vlan'

describe('VlanFormDrawer', () => {
  const createWrapper = (props = {}) => {
    return createTestWrapper(VlanFormDrawer, {
      props: {
        modelValue: true,
        mode: 'create' as const,
        formData: {
          id: null,
          name: '',
          adminStatus: 'UP',
          taggedPorts: [],
          untaggedPorts: []
        } as VlanFormData,
        deviceIp: '192.168.1.1',
        ...props
      }
    })
  }

  it('新建模式下 VLAN ID 输入框应可编辑', async () => {
    const wrapper = createWrapper()
    await waitForUpdate(wrapper)

    const vlanIdInput = wrapper.find('input[type="number"]')
    expect(vlanIdInput.exists()).toBe(true)
    expect(vlanIdInput.attributes('disabled')).toBeUndefined()
  })

  it('编辑模式下 VLAN ID 输入框应禁用', async () => {
    const wrapper = createWrapper({
      mode: 'edit',
      formData: {
        id: 100,
        name: 'Test',
        adminStatus: 'UP',
        taggedPorts: [],
        untaggedPorts: []
      } as VlanFormData
    })
    await waitForUpdate(wrapper)

    const vlanIdInput = wrapper.find('input[type="number"]')
    expect(vlanIdInput.attributes('disabled')).toBeDefined()
  })

  it('提交空表单应触发表单验证错误', async () => {
    const wrapper = createWrapper()
    await waitForUpdate(wrapper)

    // 直接点击提交
    const submitButton = wrapper.findAll('.el-button--primary').at(-1)
    await submitButton?.trigger('click')
    await nextTick()

    // 表单验证失败时不会触发 submit 事件
    expect(wrapper.emitted('submit')).toBeUndefined()
  })

  it('应包含提交和取消按钮', async () => {
    const wrapper = createWrapper()
    await waitForUpdate(wrapper)

    const buttons = wrapper.findAll('.el-button')
    const hasPrimaryButton = buttons.some(btn => btn.classes().includes('el-button--primary'))

    expect(buttons.length).toBeGreaterThan(1)
    expect(hasPrimaryButton).toBe(true)
  })

  it('应正确显示端口关联分组', async () => {
    const wrapper = createWrapper()
    await waitForUpdate(wrapper)

    // 检查是否有 Tagged 和 Untagged 端口分组
    const formItems = wrapper.findAll('.el-form-item')
    const hasTagged = formItems.some(item => item.text().includes('Tagged'))
    const hasUntagged = formItems.some(item => item.text().includes('Untagged'))

    expect(hasTagged).toBe(true)
    expect(hasUntagged).toBe(true)
  })

  it('关闭抽屉时应触发 update:modelValue 事件', async () => {
    const wrapper = createWrapper()
    await waitForUpdate(wrapper)

    // 点击取消按钮
    const cancelButton = wrapper.find('.el-button:not(.el-button--primary)')
    await cancelButton.trigger('click')
    await nextTick()

    expect(wrapper.emitted('update:modelValue')).toBeDefined()
    expect(wrapper.emitted('update:modelValue')?.[0]?.[0]).toBe(false)
  })
})
