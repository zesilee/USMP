// EviewUI 桥 · 交互控件组（组 4.1 之二）：Dropdown/Segmented/Radio/Checkbox/Switch。
// 对外 props = antd 形态；受控性按 gate 定案（Radio/Switch 有真受控开关，
// 其余控件本批不涉半受控输入类）。Radio onChange 参数序未实测——桥用
// 运行时自适应判别（新值 = ∈选项集且 ≠ 当前值），免疫两种参数顺序。
import { Children, createElement, isValidElement, type ReactNode, type CSSProperties } from 'react'
import DropDownMod from '@nce/eview-react/DropDown'
import SegmentedMod from '@nce/eview-react/Segmented'
import RadioGroupMod from '@nce/eview-react/RadioGroup'
import CheckboxMod from '@nce/eview-react/Checkbox'
import SwitchMod from '@nce/eview-react/Switch'
import { anchorId, pickDefault, textOf } from '../../bridge'

const EvDropDown = pickDefault(DropDownMod)
const EvSegmented = pickDefault(SegmentedMod)
const EvRadioGroup = pickDefault(RadioGroupMod)
const EvCheckbox = pickDefault(CheckboxMod)
const EvSwitch = pickDefault(SwitchMod)

interface CommonProps {
  'data-test'?: string
  className?: string
  style?: CSSProperties
}

// ===== Dropdown：menu{items,onClick}→data/onItemClick（key↔value）=====
export function Dropdown(
  props: CommonProps & {
    // selectedKeys：eview DropDown 无选中态高亮位，类型收下（视觉差异组 5 目视验收）。
    menu?: { items?: Array<{ key: string; label?: ReactNode; disabled?: boolean }>; onClick?: (info: { key: string }) => void; selectedKeys?: string[] }
    trigger?: string[] | string
    children?: ReactNode
  },
) {
  const items = props.menu?.items ?? []
  const trigger = Array.isArray(props.trigger) ? props.trigger[0] : props.trigger
  return createElement(
    EvDropDown,
    {
      id: anchorId(props['data-test']),
      // E2E diag5 实录：JSX label 经 String() 显示 [object Object]（语言菜单
      // 真 UI bug）——textOf 挖掘文本（label 内 data-test 锚随之丢失=已知
      // 锚点债，E2E 按文本选）。
      data: items.map((i) => ({ text: textOf(i.label), value: i.key, disabled: i.disabled })),
      onItemClick: (item: { value?: string }) => {
        if (item?.value != null) props.menu?.onClick?.({ key: item.value })
      },
      trigger: trigger === 'hover' ? 'hover' : 'click',
      className: props.className,
    },
    props.children,
  )
}

// ===== Segmented：options→data（label→text、逐项 disable 拼写）=====
export function Segmented(
  props: CommonProps & {
    options?: Array<{ label?: ReactNode; value: string | number; disabled?: boolean } | string | number>
    value?: string | number
    onChange?: (v: string | number) => void
    disabled?: boolean
  },
) {
  const data = (props.options ?? []).map((o) =>
    typeof o === 'object'
      ? { value: o.value, text: typeof o.label === 'string' ? o.label : String(o.label ?? o.value), disable: props.disabled || o.disabled }
      : { value: o, text: String(o), disable: props.disabled },
  )
  return createElement(EvSegmented, {
    id: anchorId(props['data-test']),
    data,
    value: props.value,
    onChange: (v: string | number) => props.onChange?.(v),
    className: props.className,
  })
}

// ===== Radio.Group：antd children 形态 → eview data 形态（isControlled 真受控）=====
interface RadioItemProps {
  value: string | number | boolean
  disabled?: boolean
  children?: ReactNode
}

/** antd 形态占位：仅作为 Radio.Group children 的数据载体，不独立渲染。 */
function RadioItem(_props: RadioItemProps) {
  return null
}

function RadioGroupBridge(
  props: CommonProps & {
    value?: string | number | boolean
    onChange?: (e: { target: { value: string | number | boolean } }) => void
    disabled?: boolean
    children?: ReactNode
  },
) {
  const opts: Array<{ value: string | number | boolean; text: string; disabled?: boolean }> = []
  Children.forEach(props.children, (child) => {
    if (isValidElement<RadioItemProps>(child) && child.props?.value !== undefined) {
      const label = child.props.children
      opts.push({
        value: child.props.value,
        text: typeof label === 'string' ? label : String(label ?? child.props.value),
        disabled: props.disabled || child.props.disabled,
      })
    }
  })
  const current = props.value
  // 参数序自适应（gate 未定案项）：从回调参数里找「∈选项集且≠当前值」者为新值。
  const pickNext = (...args: unknown[]): string | number | boolean | undefined => {
    const values = opts.map((o) => o.value)
    for (const a of args) {
      if (values.some((v) => Object.is(v, a)) && !Object.is(a, current)) return a as string | number | boolean
    }
    return undefined
  }
  return createElement(EvRadioGroup, {
    id: anchorId(props['data-test']),
    data: opts.map((o) => ({ value: o.value, text: o.text, disabled: o.disabled })),
    value: current,
    isControlled: true,
    onChange: (...args: unknown[]) => {
      const next = pickNext(...args)
      if (next !== undefined) props.onChange?.({ target: { value: next } })
    },
    className: props.className,
  })
}

export const Radio = Object.assign(RadioItem, { Group: RadioGroupBridge })

// ===== Checkbox：onChange 合成 antd e.target.checked 形态（onPreChange 兜底①档留桥内）=====
export function Checkbox(
  props: CommonProps & {
    checked?: boolean
    onChange?: (e: { target: { checked: boolean } }) => void
    disabled?: boolean
    children?: ReactNode
  },
) {
  return createElement(
    EvCheckbox,
    {
      id: anchorId(props['data-test']),
      checked: !!props.checked,
      // R3 侦察：真组件消费 label 出可点文本（children 不渲染）。
      label: typeof props.children === 'string' ? props.children : undefined,
      // eview onChange 第 2 参才是 checked（matrix 定案）。
      onChange: (_v: unknown, check: boolean) => props.onChange?.({ target: { checked: !!check } }),
      disabled: props.disabled,
      className: props.className,
    },
  )
}

// ===== Switch：checked→toggled + isControlToggled 真受控；必传 data 两态 =====
export function Switch(
  props: CommonProps & { checked?: boolean; onChange?: (checked: boolean) => void; disabled?: boolean },
) {
  return createElement(EvSwitch, {
    id: anchorId(props['data-test']),
    toggled: !!props.checked,
    isControlToggled: true,
    data: [false, true],
    onToggle: (v: unknown) => props.onChange?.(!!v),
    disabled: props.disabled,
    className: props.className,
  })
}
