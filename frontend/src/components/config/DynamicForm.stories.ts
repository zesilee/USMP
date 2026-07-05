import type { Meta, StoryObj } from '@storybook/vue3'
import DynamicForm from './DynamicForm.vue'
import type { Field } from '../../utils/crdSchemaParser'

// YANG 模型 → 动态表单（R05）。喂不同 field 集，无需后端即可开发/调参/回归渲染。
const meta: Meta<typeof DynamicForm> = {
  title: 'Config/DynamicForm',
  component: DynamicForm,
}
export default meta

type Story = StoryObj<typeof DynamicForm>

const vlanFields: Field[] = [
  { path: 'vlanId', type: 'number', label: 'VLAN ID', required: true, minimum: 1, maximum: 4094, group: '基本信息' },
  { path: 'vlanName', type: 'string', label: 'VLAN 名称', placeholder: '例如 VLAN-100', group: '基本信息' },
  { path: 'enabled', type: 'boolean', label: '启用', default: true, group: '基本信息' },
  { path: 'mode', type: 'enum', label: '端口模式', group: '高级设置', options: [{ label: 'Access', value: 'access' }, { label: 'Trunk', value: 'trunk' }] },
]

// boolean→开关、enum→下拉、number/string→输入，分组折叠 —— YANG 类型自动映射
export const VLAN配置: Story = {
  args: {
    fields: vlanFields,
    modelValue: { vlanId: 100, vlanName: 'VLAN-100', enabled: true, mode: 'trunk' },
  },
}

export const 空表单: Story = {
  args: {
    fields: vlanFields,
    modelValue: {},
  },
}

export const 单字段: Story = {
  args: {
    fields: [{ path: 'description', type: 'string', label: '描述', placeholder: '接口描述' }],
    modelValue: {},
  },
}
