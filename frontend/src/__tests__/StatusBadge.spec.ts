import { describe, it, expect } from 'vitest'
import StatusBadge from '../components/vlan/StatusBadge.vue'
import { createTestWrapper } from './utils'

describe('StatusBadge', () => {
  describe('管理状态 (admin)', () => {
    it('UP 状态应显示"启用"并使用 success 样式', async () => {
      const wrapper = createTestWrapper(StatusBadge, {
        props: {
          type: 'admin',
          value: 'UP'
        }
      })

      expect(wrapper.text()).toContain('启用')
      expect(wrapper.classes()).toContain('status-badge--success')
    })

    it('DOWN 状态应显示"禁用"并使用 info 样式', async () => {
      const wrapper = createTestWrapper(StatusBadge, {
        props: {
          type: 'admin',
          value: 'DOWN'
        }
      })

      expect(wrapper.text()).toContain('禁用')
      expect(wrapper.classes()).toContain('status-badge--info')
    })
  })

  describe('运行状态 (oper)', () => {
    it('ACTIVE 状态应显示"运行中"并带圆点指示器', async () => {
      const wrapper = createTestWrapper(StatusBadge, {
        props: {
          type: 'oper',
          value: 'ACTIVE'
        }
      })

      expect(wrapper.text()).toContain('运行中')
      expect(wrapper.find('.status-dot--success').exists()).toBe(true)
    })

    it('INACTIVE 状态应显示"未激活"', async () => {
      const wrapper = createTestWrapper(StatusBadge, {
        props: {
          type: 'oper',
          value: 'INACTIVE'
        }
      })

      expect(wrapper.text()).toContain('未激活')
    })

    it('SUSPENDED 状态应显示"已暂停"', async () => {
      const wrapper = createTestWrapper(StatusBadge, {
        props: {
          type: 'oper',
          value: 'SUSPENDED'
        }
      })

      expect(wrapper.text()).toContain('已暂停')
    })
  })
})
