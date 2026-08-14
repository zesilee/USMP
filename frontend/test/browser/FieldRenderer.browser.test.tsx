import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import FieldRenderer from '../../src/components/config/FieldRenderer'
import type { Field } from '../../src/utils/crdSchemaParser'

// F3 真浏览器（真 Chromium）：happy-dom 伪造不了的两类交互——
// ① antd Select 弹层真实 teleport 到 body 并可点选；
// ② 嵌套 list / leaf-list 的行级增删改真实落地（add/edit/remove 全覆盖）。

const memberPortList: Field = {
  path: '/vlan/vlans/vlan/member-ports/member-port',
  type: 'list',
  label: '端口成员',
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
}

describe('FieldRenderer 枚举下拉（真浏览器 teleport）', () => {
  it('点开 Select 弹层挂到 body，点选项回调选中值', async () => {
    const onChange = vi.fn()
    const field: Field = {
      path: '/vlan/vlans/vlan/type',
      type: 'enum',
      label: 'type',
      options: [
        { label: 'common', value: 'common' },
        { label: 'super-vlan', value: 'super-vlan' },
      ],
    }
    const { container } = render(<FieldRenderer field={field} value={undefined} onChange={onChange} />)

    await userEvent.click(container.querySelector('.ant-select')!)
    // 弹层不在组件子树内，teleport 到 body（happy-dom 下拿不到真实布局）。
    const dropdown = await vi.waitFor(() => {
      const el = document.body.querySelector('.ant-select-dropdown')
      expect(el).toBeTruthy()
      return el!
    })
    expect(dropdown.contains(container.firstElementChild)).toBe(false)

    const option = Array.from(dropdown.querySelectorAll('.ant-select-item-option')).find(
      (o) => o.textContent === 'super-vlan',
    )
    expect(option).toBeTruthy()
    await userEvent.click(option!)
    expect(onChange).toHaveBeenCalledWith('super-vlan')
  })
})

describe('FieldRenderer 嵌套 list（真浏览器 add/edit/remove）', () => {
  it('已有行渲染成子表单（输入框+枚举下拉），含添加按钮', async () => {
    const { container } = render(
      <FieldRenderer
        field={memberPortList}
        value={[{ 'interface-name': 'GE0/0/1', 'access-type': 'trunk' }]}
        onChange={vi.fn()}
      />,
    )
    const row = container.querySelector('.list-row')!
    expect(row).toBeTruthy()
    expect(row.querySelector('input.ant-input')).toBeTruthy()
    expect(row.querySelector('.ant-select')).toBeTruthy()
    expect(screen.getByRole('button', { name: /添加端口成员/ })).toBeInTheDocument()
  })

  it('add：点添加上抛追加空行的数组', async () => {
    const onChange = vi.fn()
    render(<FieldRenderer field={memberPortList} value={[]} onChange={onChange} />)
    await userEvent.click(screen.getByRole('button', { name: /添加端口成员/ }))
    expect(onChange).toHaveBeenCalledWith([{}])
  })

  it('edit：改行内输入框上抛该行合并后的数组（不可变更新）', async () => {
    const onChange = vi.fn()
    const rows = [{ 'interface-name': 'GE0/0/1', 'access-type': 'trunk' }]
    const { container } = render(<FieldRenderer field={memberPortList} value={rows} onChange={onChange} />)
    const input = container.querySelector<HTMLInputElement>('.list-row input.ant-input')!
    await userEvent.clear(input)
    await userEvent.type(input, 'X')
    const last = onChange.mock.calls.at(-1)![0]
    // 受控组件重打字符逐击上抛：末次调用行对象已合并、原数组未被原地改。
    expect(last[0]['access-type']).toBe('trunk')
    expect(rows[0]['interface-name']).toBe('GE0/0/1')
  })

  it('remove：点行删除上抛剔除该行的数组', async () => {
    const onChange = vi.fn()
    render(
      <FieldRenderer
        field={memberPortList}
        value={[
          { 'interface-name': 'GE0/0/1' },
          { 'interface-name': 'GE0/0/2' },
        ]}
        onChange={onChange}
      />,
    )
    const delButtons = screen.getAllByRole('button', { name: /删\s*除/ })
    expect(delButtons.length).toBe(2)
    await userEvent.click(delButtons[0])
    expect(onChange).toHaveBeenCalledWith([{ 'interface-name': 'GE0/0/2' }])
  })
})

describe('FieldRenderer leaf-list（真浏览器 add/edit/remove）', () => {
  const leafList: Field = {
    path: '/acl/groups/group/rule-names/rule-name',
    type: 'leaf-list',
    label: '规则名',
  }

  it('add + edit + remove 全链路上抛不可变数组', async () => {
    const onChange = vi.fn()
    const { rerender, container } = render(<FieldRenderer field={leafList} value={[]} onChange={onChange} />)
    await userEvent.click(screen.getByRole('button', { name: /添加规则名/ }))
    expect(onChange).toHaveBeenCalledWith([''])

    rerender(<FieldRenderer field={leafList} value={['r1', 'r2']} onChange={onChange} />)
    const inputs = container.querySelectorAll<HTMLInputElement>('.leaf-list-row input.ant-input')
    expect(inputs.length).toBe(2)
    await userEvent.type(inputs[1], 'x')
    expect(onChange.mock.calls.at(-1)![0]).toEqual(['r1', 'r2x'])

    const delButtons = screen.getAllByRole('button', { name: /删\s*除/ })
    await userEvent.click(delButtons[0])
    expect(onChange.mock.calls.at(-1)![0]).toEqual(['r2'])
  })
})
