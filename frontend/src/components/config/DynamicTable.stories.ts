import type { Meta, StoryObj } from '@storybook/vue3'
import DynamicTable from './DynamicTable.vue'
import type { Field } from '../../utils/crdSchemaParser'

// YANG list → 动态表格（R05）。列由 YANG field 定义驱动。
const meta: Meta<typeof DynamicTable> = {
  title: 'Config/DynamicTable',
  component: DynamicTable,
}
export default meta

type Story = StoryObj<typeof DynamicTable>

const columns: Field[] = [
  { path: 'vlanId', type: 'number', label: 'VLAN ID' },
  { path: 'vlanName', type: 'string', label: 'VLAN 名称' },
  { path: 'enabled', type: 'boolean', label: '启用' },
]

export const VLAN列表: Story = {
  args: {
    columns,
    data: [
      { vlanId: 100, vlanName: 'VLAN-100', enabled: true },
      { vlanId: 200, vlanName: 'VLAN-200', enabled: false },
      { vlanId: 300, vlanName: 'Mgmt', enabled: true },
    ],
  },
}

export const 空列表: Story = {
  args: {
    columns,
    data: [],
  },
}
