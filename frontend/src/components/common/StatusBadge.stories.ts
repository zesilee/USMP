import type { Meta, StoryObj } from '@storybook/vue3'
import StatusBadge from './StatusBadge.vue'

// 配置对账阶段徽标（Pending/Updating/Ready/Failed）。
const meta: Meta<typeof StatusBadge> = {
  title: 'Common/StatusBadge',
  component: StatusBadge,
  argTypes: {
    phase: { control: 'select', options: ['Pending', 'Updating', 'Ready', 'Failed'] },
  },
}
export default meta

type Story = StoryObj<typeof StatusBadge>

export const Ready: Story = { args: { phase: 'Ready' } }
export const Updating: Story = { args: { phase: 'Updating' } }
export const Failed: Story = { args: { phase: 'Failed' } }
export const Pending: Story = { args: { phase: 'Pending' } }
