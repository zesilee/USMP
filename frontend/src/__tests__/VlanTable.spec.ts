import { describe, it, expect, vi } from 'vitest'
import VlanTable from '../components/vlan/VlanTable.vue'
import { createTestWrapper, waitForUpdate } from './utils'
import type { VlanItem } from '../types/vlan'

describe('VlanTable', () => {
  const mockVlans: VlanItem[] = [
    {
      id: 1,
      name: 'default',
      adminStatus: 'UP',
      operStatus: 'ACTIVE',
      taggedPorts: [],
      untaggedPorts: ['GE0/1', 'GE0/2']
    },
    {
      id: 10,
      name: 'Management',
      adminStatus: 'UP',
      operStatus: 'ACTIVE',
      taggedPorts: ['GE0/3'],
      untaggedPorts: []
    },
    {
      id: 30,
      name: 'Guest',
      adminStatus: 'DOWN',
      operStatus: 'INACTIVE',
      taggedPorts: [],
      untaggedPorts: []
    }
  ]

  const createWrapper = (props = {}) => {
    return createTestWrapper(VlanTable, {
      props: {
        vlans: mockVlans,
        loading: false,
        ...props
      }
    })
  }

  it('应正确渲染 VLAN 列表数据', async () => {
    const wrapper = createWrapper()
    await waitForUpdate(wrapper)

    const rows = wrapper.findAll('.el-table__row')
    expect(rows.length).toBe(3)
    expect(wrapper.text()).toContain('default')
    expect(wrapper.text()).toContain('Management')
    expect(wrapper.text()).toContain('Guest')
  })

  it('VLAN ID 应正确显示', async () => {
    const wrapper = createWrapper()
    await waitForUpdate(wrapper)

    expect(wrapper.find('.vlan-id').text()).toBe('1')
  })

  it('应正确渲染状态徽章', async () => {
    const wrapper = createWrapper()
    await waitForUpdate(wrapper)

    const badges = wrapper.findAllComponents({ name: 'StatusBadge' })
    // 每一行有2个状态徽章
    expect(badges.length).toBe(6)
  })

  it('运行中状态应显示绿色圆点', async () => {
    const wrapper = createWrapper()
    await waitForUpdate(wrapper)

    const successDots = wrapper.findAll('.status-dot--success')
    // 前两个VLAN是ACTIVE状态
    expect(successDots.length).toBeGreaterThan(0)
  })

  it('端口标签应正确显示缩略名称', async () => {
    const wrapper = createWrapper()
    await waitForUpdate(wrapper)

    expect(wrapper.text()).toContain('GE0/1')
    expect(wrapper.text()).toContain('GE0/2')
  })

  it('点击编辑按钮应触发 edit 事件', async () => {
    const wrapper = createWrapper()
    await waitForUpdate(wrapper)

    const editButtons = wrapper.findAll('button').filter(btn => btn.text() === '编辑')
    await editButtons[0].trigger('click')

    const emitted = wrapper.emitted('edit')
    expect(emitted).toBeDefined()
    expect(emitted?.[0]?.[0]).toMatchObject({ id: 1, name: 'default' })
  })

  it('点击删除按钮应触发 delete 事件', async () => {
    const wrapper = createWrapper()
    await waitForUpdate(wrapper)

    const deleteButtons = wrapper.findAll('button').filter(btn => btn.text() === '删除')
    await deleteButtons[0].trigger('click')

    const emitted = wrapper.emitted('delete')
    expect(emitted).toBeDefined()
    expect(emitted?.[0]?.[0]).toMatchObject({ id: 1 })
  })

  it('表格应支持多选功能', async () => {
    const wrapper = createWrapper()
    await waitForUpdate(wrapper)

    // 验证多选列存在
    const checkbox = wrapper.find('.el-checkbox__input')
    expect(checkbox.exists()).toBe(true)
  })

  it('loading 状态应显示加载动画', async () => {
    const wrapper = createWrapper({ loading: true })
    await waitForUpdate(wrapper)

    expect(wrapper.find('.el-loading-mask').exists()).toBe(true)
  })

  it('底部应显示 VLAN 总数', async () => {
    const wrapper = createWrapper()
    await waitForUpdate(wrapper)

    expect(wrapper.find('.total-info').text()).toContain('3 个 VLAN')
  })
})
