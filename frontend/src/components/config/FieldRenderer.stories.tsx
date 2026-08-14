import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import FieldRenderer from './FieldRenderer'
import type { Field } from '../../utils/crdSchemaParser'

// 框架冒烟故事（tasks 13.6）：验证 React 化后 Storybook 构建链路可用。
// 旧 Vue 栈故事内容不迁移（Non-Goal），组件故事按需重建。
const meta: Meta<typeof FieldRenderer> = {
  title: 'Config/FieldRenderer',
  component: FieldRenderer,
}
export default meta

type Story = StoryObj<typeof FieldRenderer>

function Controlled({ field, initial }: { field: Field; initial?: unknown }) {
  const [value, setValue] = useState<unknown>(initial)
  return <FieldRenderer field={field} value={value} onChange={setValue} />
}

export const Enum: Story = {
  render: () => (
    <Controlled
      field={{
        path: '/vlan/vlans/vlan/type',
        type: 'enum',
        label: 'type',
        options: [
          { label: 'common', value: 'common' },
          { label: 'super-vlan', value: 'super-vlan' },
        ],
      }}
    />
  ),
}

export const NestedList: Story = {
  render: () => (
    <Controlled
      field={{
        path: '/vlan/vlans/vlan/member-ports/member-port',
        type: 'list',
        label: 'member-port',
        fields: [
          { path: '/vlan/vlans/vlan/member-ports/member-port/interface-name', type: 'string', label: 'interface-name' },
          {
            path: '/vlan/vlans/vlan/member-ports/member-port/access-type',
            type: 'enum',
            label: 'access-type',
            options: [
              { label: 'access', value: 'access' },
              { label: 'trunk', value: 'trunk' },
            ],
          },
        ],
      }}
      initial={[{ 'interface-name': 'GE0/0/1', 'access-type': 'trunk' }]}
    />
  ),
}
