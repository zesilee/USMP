import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import FieldRenderer from '../../src/components/config/FieldRenderer'
import type { Field } from '../../src/utils/crdSchemaParser'

// F3 真浏览器：choice 互斥分支的真实切换（FE-08）——
// 多字段 case 走 antd Tabs（真实 tab 轨道/激活态），切 case 清空其它分支成员键。

const tabsChoice: Field = {
  path: '/nat/rule/match',
  type: 'choice',
  label: 'match',
  cases: [
    {
      name: 'by-address',
      label: '按地址',
      fields: [
        { path: '/nat/rule/match/src-ip', type: 'string', label: 'src-ip' },
        { path: '/nat/rule/match/dst-ip', type: 'string', label: 'dst-ip' },
      ],
    },
    {
      name: 'by-acl',
      label: '按ACL',
      fields: [
        { path: '/nat/rule/match/acl-name', type: 'string', label: 'acl-name' },
        { path: '/nat/rule/match/acl-type', type: 'string', label: 'acl-type' },
      ],
    },
  ],
}

const radioChoice: Field = {
  path: '/qos/policy/action',
  type: 'choice',
  label: 'action',
  cases: [
    { name: 'permit', label: '放行', fields: [{ path: '/qos/policy/action/permit', type: 'boolean', label: 'permit' }] },
    { name: 'deny', label: '拒绝', fields: [{ path: '/qos/policy/action/deny', type: 'boolean', label: 'deny' }] },
  ],
}

describe('FieldRenderer choice（真浏览器）', () => {
  it('多字段 case 渲染为 Tabs，有数据的 case 自动激活', () => {
    render(
      <FieldRenderer field={tabsChoice} value={{ 'acl-name': 'a1' }} onChange={vi.fn()} />,
    )
    const active = document.querySelector('.ant-tabs-tab-active')
    expect(active?.textContent).toBe('按ACL')
    // 激活 case 的成员字段真实渲染
    expect(screen.getByText('acl-name')).toBeInTheDocument()
  })

  it('真实点击切 Tab：上抛清空其它分支成员键（YANG choice 互斥）', async () => {
    const onChange = vi.fn()
    render(
      <FieldRenderer field={tabsChoice} value={{ 'acl-name': 'a1', 'acl-type': 'basic' }} onChange={onChange} />,
    )
    await userEvent.click(screen.getByRole('tab', { name: '按地址' }))
    expect(onChange).toHaveBeenCalledWith({})
  })

  it('单叶 case 走 Radio 组，选分支后成员字段展开且互斥清理', async () => {
    const onChange = vi.fn()
    render(<FieldRenderer field={radioChoice} value={{ permit: true }} onChange={onChange} />)
    // 有数据的 permit case 激活。断 antd 受控 class 而非 DOM checked：test 环境
    // antd useId 恒为 test-id，choice 组与成员 boolean 组撞 name，真浏览器把两组
    // radio 视作一组、DOM checked 属性互斥失真（生产 useId 唯一，无此问题）。
    const permitRadio = screen.getByRole('radio', { name: '放行' })
    expect(permitRadio.closest('.ant-radio-wrapper')).toHaveClass('ant-radio-wrapper-checked')
    await userEvent.click(screen.getByRole('radio', { name: '拒绝' }))
    // 切到 deny：permit 成员键被清（键不存在=节点不存在，FE-27）
    expect(onChange).toHaveBeenCalledWith({})
  })
})
