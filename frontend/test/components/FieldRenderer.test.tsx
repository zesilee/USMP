import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import FieldRenderer from '../../src/components/config/FieldRenderer'
import { UiProvider } from '../../src/ui'
import type { Field } from '../../src/utils/crdSchemaParser'

// FieldRenderer F2（FE-01/R05）：类型→控件分派、受控单向、嵌套整体上抛、
// FE-12/27 的 presence/choice 键语义。控件断言优先走可访问角色/文案。
function mount(field: Field, value: any, onChange = vi.fn(), disabled = false) {
  const utils = render(
    <UiProvider>
      <FieldRenderer field={field} value={value} onChange={onChange} disabled={disabled} />
    </UiProvider>,
  )
  return { ...utils, onChange }
}

describe('FieldRenderer · 类型→控件分派（FE-01）', () => {
  it('string → 文本框，输入即上抛新值', () => {
    const { onChange } = mount({ path: '/m/name', type: 'string', label: 'name' }, 'ge0')
    const input = screen.getByRole('textbox')
    expect(input).toHaveValue('ge0')
    fireEvent.change(input, { target: { value: 'ge1' } })
    expect(onChange).toHaveBeenCalledWith('ge1')
  })

  it('string + units → 单位后缀展示（FE-15）', () => {
    mount({ path: '/m/mtu', type: 'string', label: 'mtu', units: 'byte' }, '')
    expect(screen.getByText('byte')).toBeInTheDocument()
  })

  it('number → 数字框，min/max 透传，清空上抛 undefined', () => {
    const { onChange } = mount(
      { path: '/m/vid', type: 'number', label: 'vid', minimum: 1, maximum: 4094 },
      10,
    )
    const spin = screen.getByRole('spinbutton')
    expect(spin).toHaveValue('10')
    fireEvent.change(spin, { target: { value: '' } })
    fireEvent.blur(spin)
    expect(onChange).toHaveBeenLastCalledWith(undefined)
  })

  it('boolean →「打开/关闭」radio，值仍 true/false；未选两项均不选中', () => {
    const { onChange } = mount({ path: '/m/en', type: 'boolean', label: 'en' }, undefined)
    const radios = screen.getAllByRole('radio')
    expect(radios).toHaveLength(2)
    expect(radios.every((r) => !(r as HTMLInputElement).checked)).toBe(true)
    fireEvent.click(radios[0])
    expect(onChange).toHaveBeenCalledWith(true)
  })

  it('enum 必填且 ≤3 → 分段控件（FE-01 细分）', () => {
    const f: Field = {
      path: '/m/mode', type: 'enum', label: 'mode', required: true,
      options: [
        { label: 'access', value: 'access' },
        { label: 'trunk', value: 'trunk' },
      ],
    }
    const { onChange } = mount(f, 'access')
    // Segmented 渲染为 radio 组形态
    expect(screen.getByText('trunk')).toBeInTheDocument()
    fireEvent.click(screen.getByText('trunk'))
    expect(onChange).toHaveBeenCalledWith('trunk')
  })

  it('enum 可选 → 下拉（保留清空=键不入 payload 语义）', () => {
    const f: Field = {
      path: '/m/mode', type: 'enum', label: 'mode',
      options: [{ label: 'a', value: 'a' }, { label: 'b', value: 'b' }],
    }
    mount(f, 'a')
    expect(screen.getByRole('combobox')).toBeInTheDocument()
  })

  it('leafref 字段即使零选项也保持下拉、不降级文本框（FE-19 语义）', () => {
    mount(
      { path: '/m/if', type: 'string', label: 'if', leafRef: '/ifm:ifm/ifm:interfaces/ifm:interface/ifm:name' },
      undefined,
    )
    expect(document.querySelector('[data-test="leafref-select"]')).toBeTruthy()
    expect(screen.queryByRole('textbox')).toBeNull()
  })

  it('未知类型降级文本框不崩（R08）', () => {
    mount({ path: '/m/x', type: 'weird' as any, label: 'x' }, 'v')
    expect(screen.getByRole('textbox')).toHaveValue('v')
  })
})

describe('FieldRenderer · presence 容器（FE-12/FE-27）', () => {
  const f: Field = {
    path: '/m/tuning', type: 'group', label: 'tuning', presence: true,
    fields: [{ path: '/m/tuning/level', type: 'string', label: 'level' }],
  }

  it('关闭态：开关 off、子表单不渲染；开启 → 上抛空对象', () => {
    const { onChange } = mount(f, undefined)
    const sw = screen.getByRole('switch')
    expect(sw).not.toBeChecked()
    expect(screen.queryByRole('textbox')).toBeNull()
    fireEvent.click(sw)
    expect(onChange).toHaveBeenCalledWith({})
  })

  it('开启态：子表单渲染；关闭 → 上抛 undefined（键不入 payload）', () => {
    const { onChange } = mount(f, { level: 'high' })
    expect(screen.getByRole('textbox')).toHaveValue('high')
    fireEvent.click(screen.getByRole('switch'))
    expect(onChange).toHaveBeenCalledWith(undefined)
  })

  it('编辑子字段：整对象上抛（保留其余键）', () => {
    const { onChange } = mount(f, { level: 'high' })
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'low' } })
    expect(onChange).toHaveBeenCalledWith({ level: 'low' })
  })
})

describe('FieldRenderer · choice 互斥分支（FE-08）', () => {
  const f: Field = {
    path: '/m/mode', type: 'choice', label: 'mode',
    cases: [
      { name: 'a', label: 'caseA', fields: [{ path: '/m/speed', type: 'number', label: 'speed' }] },
      { name: 'b', label: 'caseB', fields: [{ path: '/m/auto', type: 'boolean', label: 'auto' }] },
    ],
  }

  it('全单叶 case → RadioGroup；按数据推断激活分支', () => {
    mount(f, { speed: 100 })
    expect(screen.getByRole('radio', { name: 'caseA' })).toBeChecked()
    expect(screen.getByRole('spinbutton')).toHaveValue('100')
  })

  it('切分支：非激活 case 成员键被清（上抛对象不含旧成员，FE-27 源头）', () => {
    const { onChange } = mount(f, { speed: 100 })
    fireEvent.click(screen.getByRole('radio', { name: 'caseB' }))
    const next = onChange.mock.calls.at(-1)![0]
    expect('speed' in next).toBe(false)
  })

  it('编辑激活成员：保留 scope 其它键整体上抛', () => {
    const { onChange } = mount(f, { speed: 100, other: 'x' })
    fireEvent.change(screen.getByRole('spinbutton'), { target: { value: '200' } })
    fireEvent.blur(screen.getByRole('spinbutton'))
    const next = onChange.mock.calls.at(-1)![0]
    expect(next.speed).toBe(200)
    expect(next.other).toBe('x')
  })
})

describe('FieldRenderer · choice Tabs 形态（任一 case 多字段/容器）', () => {
  const f: Field = {
    path: '/m/addr', type: 'choice', label: 'addr',
    cases: [
      {
        name: 'static', label: 'Static',
        fields: [
          { path: '/m/ip', type: 'string', label: 'ip' },
          { path: '/m/mask', type: 'string', label: 'mask' },
        ],
      },
      { name: 'dhcp', label: 'DHCP', fields: [{ path: '/m/dhcp-en', type: 'boolean', label: 'dhcp-en' }] },
    ],
  }

  it('多字段 case → Tabs 渲染；切 Tab 清空非激活成员', () => {
    const { onChange } = mount(f, { ip: '1.1.1.1', mask: '24' })
    expect(screen.getByRole('tab', { name: 'Static' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('tab', { name: 'DHCP' }))
    const next = onChange.mock.calls.at(-1)![0]
    expect('ip' in next).toBe(false)
    expect('mask' in next).toBe(false)
  })

  it('Tabs 内编辑成员：整 scope 上抛', () => {
    const { onChange } = mount(f, { ip: '1.1.1.1', mask: '24' })
    const inputs = screen.getAllByRole('textbox')
    fireEvent.change(inputs[0], { target: { value: '2.2.2.2' } })
    const next = onChange.mock.calls.at(-1)![0]
    expect(next.ip).toBe('2.2.2.2')
    expect(next.mask).toBe('24')
  })
})

describe('FieldRenderer · group 与枚举补充分支', () => {
  it('普通 group：子字段编辑整对象上抛（保留其余键）', () => {
    const g: Field = {
      path: '/m/cfg', type: 'group', label: 'cfg',
      fields: [{ path: '/m/cfg/a', type: 'string', label: 'a' }],
    }
    const { onChange } = mount(g, { a: 'x', keep: 1 })
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'y' } })
    expect(onChange).toHaveBeenLastCalledWith({ a: 'y', keep: 1 })
  })

  it('enum 必填但零选项 → 降级下拉不崩（R08 边界）', () => {
    mount({ path: '/m/e', type: 'enum', label: 'e', required: true, options: [] }, undefined)
    expect(screen.getByRole('combobox')).toBeInTheDocument()
  })

  it('leaf-list 带 options → 元素渲染为下拉，展开选值定点更新', async () => {
    const user = userEvent.setup()
    const ll: Field = {
      path: '/m/vlans', type: 'leaf-list', label: 'vlans',
      options: [{ label: 'v10', value: '10' }, { label: 'v20', value: '20' }],
    }
    const { onChange } = mount(ll, ['10'])
    await user.click(screen.getByRole('combobox'))
    await user.click(await screen.findByTitle('v20'))
    expect(onChange).toHaveBeenLastCalledWith(['20'])
  })

  it('enum 下拉展开选值上抛；清空按钮上抛 undefined（键不入 payload 源头）', async () => {
    const user = userEvent.setup()
    const f: Field = {
      path: '/m/mode', type: 'enum', label: 'mode',
      options: [{ label: 'optA', value: 'a' }, { label: 'optB', value: 'b' }],
    }
    const { onChange, container } = mount(f, 'a')
    await user.click(screen.getByRole('combobox'))
    await user.click(await screen.findByTitle('optB'))
    expect(onChange).toHaveBeenLastCalledWith('b')

    const clearBtn = container.querySelector('.ant-select-clear')
    expect(clearBtn, 'allowClear 须存在（可选枚举清空语义）').toBeTruthy()
    fireEvent.mouseDown(clearBtn!)
    fireEvent.click(clearBtn!)
    expect(onChange).toHaveBeenLastCalledWith(undefined)
  })

  it('leafref 下拉选值上抛（选项由调用方注入）', async () => {
    const user = userEvent.setup()
    const f: Field = {
      path: '/m/if', type: 'string', label: 'if', leafRef: '/x/y/name',
      options: [{ label: 'GE0/0/1', value: 'GE0/0/1' }],
    }
    const { onChange } = mount(f, undefined)
    await user.click(screen.getByRole('combobox'))
    await user.click(await screen.findByTitle('GE0/0/1'))
    expect(onChange).toHaveBeenLastCalledWith('GE0/0/1')
  })

  it('segmented（必填短枚举）切换上抛新值', () => {
    const f: Field = {
      path: '/m/dir', type: 'enum', label: 'dir', required: true,
      options: [{ label: 'in', value: 'inbound' }, { label: 'out', value: 'outbound' }],
    }
    const { onChange } = mount(f, 'inbound')
    fireEvent.click(screen.getByText('out'))
    expect(onChange).toHaveBeenCalledWith('outbound')
  })

  it('string length 约束合成占位（FE-22）', () => {
    mount({ path: '/m/desc', type: 'string', label: 'desc', minLength: 1, maxLength: 242 }, undefined)
    const ph = (screen.getByRole('textbox') as HTMLInputElement).placeholder
    expect(ph).toContain('242')
  })
})

describe('FieldRenderer · leaf-list 与嵌套 list（add/edit/remove 全径）', () => {
  it('leaf-list：add 追加空项、edit 定点改、remove 删指定项', async () => {
    const user = userEvent.setup()
    const ll: Field = { path: '/m/tags', type: 'leaf-list', label: 'tags' }
    const { onChange, rerender } = mount(ll, ['a', 'b'])

    await user.click(screen.getByRole('button', { name: /tags/ }))
    expect(onChange).toHaveBeenLastCalledWith(['a', 'b', ''])

    const inputs = screen.getAllByRole('textbox')
    fireEvent.change(inputs[1], { target: { value: 'B' } })
    expect(onChange).toHaveBeenLastCalledWith(['a', 'B'])

    rerender(
      <UiProvider>
        <FieldRenderer field={ll} value={['a', 'b']} onChange={onChange} />
      </UiProvider>,
    )
    const delBtns = screen.getAllByRole('button').filter((b) => b.className.includes('dangerous') || b.textContent?.includes('删除') || b.textContent?.toLowerCase().includes('delete'))
    await user.click(delBtns[0])
    expect(onChange).toHaveBeenLastCalledWith(['b'])
  })

  it('嵌套 list：add 追加空行、行内 edit 整行上抛、remove 删行', async () => {
    const user = userEvent.setup()
    const lf: Field = {
      path: '/m/members', type: 'list', label: 'members',
      fields: [{ path: '/m/members/member/id', type: 'string', label: 'id' }],
    }
    const { onChange } = mount(lf, [{ id: 'm1' }])

    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'm9' } })
    expect(onChange).toHaveBeenLastCalledWith([{ id: 'm9' }])

    await user.click(screen.getByRole('button', { name: /members/ }))
    expect(onChange).toHaveBeenLastCalledWith([{ id: 'm1' }, {}])

    const delBtn = screen.getAllByRole('button').find((b) => b.textContent?.includes('删除') || b.textContent?.toLowerCase().includes('delete'))!
    await user.click(delBtn)
    expect(onChange).toHaveBeenLastCalledWith([])
  })
})

describe('FieldRenderer · 占位与禁用（FE-15/FE-14）', () => {
  it('数值 range + default 合成占位', () => {
    mount({ path: '/m/vid', type: 'number', label: 'vid', minimum: 1, maximum: 4094, default: 1 }, undefined)
    const ph = (screen.getByRole('spinbutton') as HTMLInputElement).placeholder
    expect(ph).toContain('1')
    expect(ph).toContain('4094')
  })

  it('dynamicDefault 优先展示「系统自动分配」占位', () => {
    mount({ path: '/m/idx', type: 'string', label: 'idx', dynamicDefault: true }, undefined)
    expect((screen.getByRole('textbox') as HTMLInputElement).placeholder).not.toBe('')
  })

  it('readonly 叶控件禁用（FE-14 可见可回显不可改）', () => {
    mount({ path: '/m/oper', type: 'string', label: 'oper', readonly: true }, 'up')
    expect(screen.getByRole('textbox')).toBeDisabled()
  })
})
