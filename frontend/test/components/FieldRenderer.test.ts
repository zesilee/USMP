import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import FieldRenderer from '../../src/components/config/FieldRenderer.vue'
import ElementPlus from 'element-plus'

const baseField = {
  path: 'test',
  type: 'string' as const,
  label: '测试字段',
  placeholder: '请输入'
}

describe('FieldRenderer Component', () => {
  it('should render ElInput for string type', () => {
    const wrapper = mount(FieldRenderer, {
      props: { field: baseField, modelValue: '' },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.findComponent({ name: 'ElInput' }).exists()).toBe(true)
  })

  it('should render ElInputNumber for number type', () => {
    const wrapper = mount(FieldRenderer, {
      props: { field: { ...baseField, type: 'number' }, modelValue: 0 },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.findComponent({ name: 'ElInputNumber' }).exists()).toBe(true)
  })

  it('boolean 渲染「打开/关闭」radio 而非开关（FE-01 NCE 形态）', () => {
    const wrapper = mount(FieldRenderer, {
      props: { field: { ...baseField, type: 'boolean' }, modelValue: false },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.findComponent({ name: 'ElRadioGroup' }).exists()).toBe(true)
    expect(wrapper.findComponent({ name: 'ElSwitch' }).exists()).toBe(false)
    const labels = wrapper.findAll('.el-radio').map((r) => r.text())
    expect(labels).toEqual(['打开', '关闭'])
  })

  it('should render ElSelect for enum type', () => {
    const wrapper = mount(FieldRenderer, {
      props: {
        field: {
          ...baseField,
          type: 'enum',
          options: [{ label: '选项1', value: '1' }, { label: '选项2', value: '2' }]
        },
        modelValue: ''
      },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.findComponent({ name: 'ElSelect' }).exists()).toBe(true)
  })
})

describe('FieldRenderer · boolean radio 值语义（FE-01）', () => {
  it('选「打开」emit true、「关闭」emit false；禁用态透传', async () => {
    const w = mount(FieldRenderer, {
      props: { field: { path: '/x/en', type: 'boolean' as const, label: 'en' }, modelValue: undefined },
      global: { plugins: [ElementPlus] },
    })
    const radios = w.findAll('.el-radio input[type="radio"]')
    await radios[0].setValue()
    expect(w.emitted('update:modelValue')?.at(-1)).toEqual([true])
    await radios[1].setValue()
    expect(w.emitted('update:modelValue')?.at(-1)).toEqual([false])

    const disabled = mount(FieldRenderer, {
      props: { field: { path: '/x/en', type: 'boolean' as const, label: 'en', readonly: true }, modelValue: true },
      global: { plugins: [ElementPlus] },
    })
    expect(disabled.findAll('.el-radio.is-disabled')).toHaveLength(2)
  })

  it('可选 boolean 未选：两个 radio 均未选中（不入 payload 语义由表单层保证）', () => {
    const w = mount(FieldRenderer, {
      props: { field: { path: '/x/en', type: 'boolean' as const, label: 'en' }, modelValue: undefined },
      global: { plugins: [ElementPlus] },
    })
    expect(w.findAll('.el-radio.is-checked')).toHaveLength(0)
  })
})

describe('FieldRenderer · 约束合成占位（FE-22）', () => {
  it('数值 range 合成「整数 合法范围」占位', () => {
    const w = mount(FieldRenderer, {
      props: {
        field: { path: '/x/mac-age', type: 'number' as const, label: 'mac-age', minimum: 60, maximum: 1000000 },
        modelValue: undefined,
      },
      global: { plugins: [ElementPlus] },
    })
    expect(w.find('input').attributes('placeholder')).toBe('整数 合法范围: [60, 1000000]')
  })

  it('单边界 range 合成 ≥/≤ 形态；无约束无占位', () => {
    const minOnly = mount(FieldRenderer, {
      props: { field: { path: '/x/a', type: 'number' as const, label: 'a', minimum: 10 }, modelValue: undefined },
      global: { plugins: [ElementPlus] },
    })
    expect(minOnly.find('input').attributes('placeholder')).toContain('≥ 10')
    const none = mount(FieldRenderer, {
      props: { field: { path: '/x/b', type: 'number' as const, label: 'b' }, modelValue: undefined },
      global: { plugins: [ElementPlus] },
    })
    expect(none.find('input').attributes('placeholder') || '').toBe('')
  })

  it('优先级：显式 placeholder > dynamicDefault > range 合成', () => {
    const dyn = mount(FieldRenderer, {
      props: {
        field: { path: '/x/c', type: 'number' as const, label: 'c', minimum: 1, maximum: 9, dynamicDefault: true },
        modelValue: undefined,
      },
      global: { plugins: [ElementPlus] },
    })
    expect(dyn.find('input').attributes('placeholder')).toContain('系统自动分配')
    const explicit = mount(FieldRenderer, {
      props: {
        field: { path: '/x/d', type: 'number' as const, label: 'd', minimum: 1, maximum: 9, placeholder: '自定义' },
        modelValue: undefined,
      },
      global: { plugins: [ElementPlus] },
    })
    expect(explicit.find('input').attributes('placeholder')).toBe('自定义')
  })
})

describe('FieldRenderer · 默认值并入合成占位（FE-22 扩展，NCE waterMark 对齐）', () => {
  const g = { global: { plugins: [ElementPlus] } }

  it('range + default：范围段后追加「，默认值: X」', () => {
    const w = mount(FieldRenderer, {
      props: {
        field: { path: '/x/iv', type: 'number' as const, label: 'iv', minimum: 10, maximum: 600, default: 300 },
        modelValue: undefined,
      },
      ...g,
    })
    expect(w.find('input').attributes('placeholder')).toBe('整数 合法范围: [10, 600]，默认值: 300')
  })

  it('length + default：长度段后追加默认值', () => {
    const w = mount(FieldRenderer, {
      props: {
        field: { path: '/x/nm', type: 'string' as const, label: 'nm', minLength: 1, maxLength: 31, default: 'vlan1' },
        modelValue: '',
      },
      ...g,
    })
    expect(w.find('input').attributes('placeholder')).toBe('合法长度: [1..31]，默认值: vlan1')
  })

  it('仅 default（无 range/length）：单独展示「默认值: X」，值原样不本地化', () => {
    const w = mount(FieldRenderer, {
      props: {
        field: { path: '/x/mode', type: 'string' as const, label: 'mode', default: 'dot1q' },
        modelValue: '',
      },
      ...g,
    })
    expect(w.find('input').attributes('placeholder')).toBe('默认值: dot1q')
  })

  it('enum 下拉空值展示默认值占位', () => {
    const w = mount(FieldRenderer, {
      props: {
        field: {
          path: '/x/proto', type: 'enum' as const, label: 'proto', default: 'ethernet',
          options: [
            { label: 'ethernet', value: 'ethernet' },
            { label: 'hdlc', value: 'hdlc' },
            { label: 'ppp', value: 'ppp' },
            { label: 'fr', value: 'fr' },
          ],
        },
        modelValue: undefined,
      },
      ...g,
    })
    expect(w.find('.el-select__placeholder').text()).toBe('默认值: ethernet')
  })

  it('dynamicDefault 优先于 default（边界）', () => {
    const w = mount(FieldRenderer, {
      props: {
        field: { path: '/x/dd', type: 'number' as const, label: 'dd', default: 5, dynamicDefault: true },
        modelValue: undefined,
      },
      ...g,
    })
    expect(w.find('input').attributes('placeholder')).toContain('系统自动分配')
    expect(w.find('input').attributes('placeholder')).not.toContain('默认值')
  })

  it('boolean 不合成占位（负路径）', () => {
    const w = mount(FieldRenderer, {
      props: {
        field: { path: '/x/en', type: 'boolean' as const, label: 'en', default: false },
        modelValue: undefined,
      },
      ...g,
    })
    expect(w.find('input[placeholder]').exists()).toBe(false)
  })
})

describe('FieldRenderer · 动态缺省占位与单位后缀（FE-15）', () => {
  it('dynamicDefault 字段空值展示「系统自动分配」占位', () => {
    const w = mount(FieldRenderer, {
      props: {
        field: { path: '/ifm/x/admin-status', type: 'string', label: 'admin-status', dynamicDefault: true },
        modelValue: undefined,
      },
      global: { plugins: [ElementPlus] },
    })
    expect(w.find('input').attributes('placeholder')).toContain('系统自动分配')
  })

  it('显式 placeholder 优先于动态缺省占位', () => {
    const w = mount(FieldRenderer, {
      props: {
        field: { path: '/x/a', type: 'string', label: 'a', dynamicDefault: true, placeholder: '自定义' },
        modelValue: undefined,
      },
      global: { plugins: [ElementPlus] },
    })
    expect(w.find('input').attributes('placeholder')).toBe('自定义')
  })

  it('units 在 string 输入框渲染单位后缀', () => {
    const w = mount(FieldRenderer, {
      props: {
        field: { path: '/x/bw', type: 'string', label: 'bw', units: 'bit/s' },
        modelValue: '100',
      },
      global: { plugins: [ElementPlus] },
    })
    expect(w.find('.field-units').text()).toBe('bit/s')
  })

  it('units 在 number 输入框渲染单位后缀；无 units 不渲染', () => {
    const withUnits = mount(FieldRenderer, {
      props: {
        field: { path: '/x/mtu', type: 'number', label: 'mtu', units: 'octets' },
        modelValue: 1500,
      },
      global: { plugins: [ElementPlus] },
    })
    expect(withUnits.find('.field-units').text()).toBe('octets')

    const without = mount(FieldRenderer, {
      props: {
        field: { path: '/x/mtu', type: 'number', label: 'mtu' },
        modelValue: 1500,
      },
      global: { plugins: [ElementPlus] },
    })
    expect(without.find('.field-units').exists()).toBe(false)
  })
})

describe('FieldRenderer · group 内 readonly 状态叶回显（NS-08/BR-01，FE-14 对齐）', () => {
  const dynGroup = {
    path: '/ifm/interfaces/interface/dynamic',
    type: 'group' as const,
    label: '接口动态信息',
    readonly: true,
    fields: [
      { path: '/ifm/interfaces/interface/dynamic/mac-address', type: 'string' as const, label: 'mac-address', readonly: true },
      { path: '/ifm/interfaces/interface/dynamic/oper-status', type: 'string' as const, label: 'oper-status', readonly: true },
    ],
  }

  it('readonly 子叶应渲染为禁用输入并回显状态值（不得被过滤成空组）', () => {
    const w = mount(FieldRenderer, {
      props: {
        field: dynGroup,
        modelValue: { 'mac-address': '00:e0:fc:12:34:01', 'oper-status': 'up' },
      },
      global: { plugins: [ElementPlus] },
    })
    const inputs = w.findAll('input')
    expect(inputs.length).toBeGreaterThanOrEqual(2)
    const mac = inputs.find((i) => (i.element as HTMLInputElement).value === '00:e0:fc:12:34:01')
    expect(mac, 'mac-address 状态值应回显').toBeTruthy()
    expect((mac!.element as HTMLInputElement).disabled).toBe(true)
  })

  it('混合 group：readonly 叶与可写叶并存时两者都渲染，仅 readonly 禁用', () => {
    const mixed = {
      path: '/x/g',
      type: 'group' as const,
      label: 'g',
      fields: [
        { path: '/x/g/writable', type: 'string' as const, label: 'writable' },
        { path: '/x/g/state', type: 'string' as const, label: 'state', readonly: true },
      ],
    }
    const w = mount(FieldRenderer, {
      props: { field: mixed, modelValue: { writable: 'a', state: 'b' } },
      global: { plugins: [ElementPlus] },
    })
    const inputs = w.findAll('input')
    const writable = inputs.find((i) => (i.element as HTMLInputElement).value === 'a')
    const state = inputs.find((i) => (i.element as HTMLInputElement).value === 'b')
    expect(writable, '可写叶应渲染').toBeTruthy()
    expect((writable!.element as HTMLInputElement).disabled).toBe(false)
    expect(state, 'readonly 叶应渲染回显').toBeTruthy()
    expect((state!.element as HTMLInputElement).disabled).toBe(true)
  })
})

describe('FieldRenderer · 字符串 length 合成占位（FE-22/D9）', () => {
  it('minLength+maxLength → 合法长度占位；显式 placeholder 与 dynamicDefault 优先', () => {
    const f = (extra: any = {}) =>
      mount(FieldRenderer, {
        props: { field: { path: '/x/name', type: 'string', label: 'name', minLength: 1, maxLength: 31, ...extra } as any, modelValue: '' },
        global: { plugins: [ElementPlus] },
      })
    expect(f().find('input').attributes('placeholder')).toBe('合法长度: [1..31]')
    expect(f({ placeholder: '显式占位' }).find('input').attributes('placeholder')).toBe('显式占位')
    expect(f({ dynamicDefault: true }).find('input').attributes('placeholder')).toContain('系统自动分配')
  })

  it('仅一侧界与数值字段不受影响（边界）', () => {
    const one = mount(FieldRenderer, {
      props: { field: { path: '/x/desc', type: 'string', label: 'desc', maxLength: 80 } as any, modelValue: '' },
      global: { plugins: [ElementPlus] },
    })
    expect(one.find('input').attributes('placeholder')).toBe('合法长度: ≤ 80')
    const num = mount(FieldRenderer, {
      props: { field: { path: '/x/mtu', type: 'number', label: 'mtu', minimum: 60, maximum: 9600 } as any, modelValue: undefined },
      global: { plugins: [ElementPlus] },
    })
    expect(num.find('input').attributes('placeholder')).toBe('整数 合法范围: [60, 9600]')
  })
})

describe('FieldRenderer · leafref 禁自由文本（FE-19）', () => {
  const leafrefField = {
    path: 'if-name',
    type: 'string' as const,
    label: 'if-name',
    leafRef: '/ifm:ifm/ifm:interfaces/ifm:interface/ifm:name',
  }

  it('leafref 字段无 options（拉取失败/为空）→ 仍渲染下拉，非文本框', () => {
    const w = mount(FieldRenderer, {
      props: { field: leafrefField as any, modelValue: '' },
      global: { plugins: [ElementPlus] },
    })
    expect(w.find('[data-test="leafref-select"]').exists()).toBe(true)
    expect(w.find('.field-scalar').exists()).toBe(false)
  })

  it('leafref 字段有 options → 下拉含选项（既有行为回归）', () => {
    const w = mount(FieldRenderer, {
      props: {
        field: { ...leafrefField, options: [{ label: '200GE0/1/0', value: '200GE0/1/0' }] } as any,
        modelValue: '',
      },
      global: { plugins: [ElementPlus] },
    })
    expect(w.find('[data-test="leafref-select"]').exists()).toBe(true)
  })

  it('非 leafref 普通 string 字段仍渲染文本框（回归不破）', () => {
    const w = mount(FieldRenderer, {
      props: { field: { path: 'desc', type: 'string' as const, label: 'desc' } as any, modelValue: '' },
      global: { plugins: [ElementPlus] },
    })
    expect(w.find('.field-scalar').exists()).toBe(true)
    expect(w.find('[data-test="leafref-select"]').exists()).toBe(false)
  })
})
