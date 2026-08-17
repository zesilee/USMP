// EviewUI 桥 · 表单输入组（组 4.2）：Input/InputNumber/Select。三件底层均为
// 半受控（gate 实证）——统一走 useSemiControlledBridge（②受控回写+③key 重挂）。
// 对外 props = antd 形态；映射依据 = component-matrix + gate R1/R2（勿凭空改）。
import { createElement, type ReactNode, type CSSProperties } from 'react'
import TextFieldMod from '@nce/eview-react/TextField'
import SpinnerMod from '@nce/eview-react/Spinner'
import InputSelectMod from '@nce/eview-react/InputSelect'
import { anchorId, useSemiControlledBridge, pickDefault } from '../../bridge'

const EvTextField = pickDefault(TextFieldMod)
const EvSpinner = pickDefault(SpinnerMod)
const EvInputSelect = pickDefault(InputSelectMod)

interface CommonProps {
  'data-test'?: string
  className?: string
  style?: CSSProperties
}

// ===== Input → TextField =====
// 缺口补齐：allowClear 自绘清除钮挂 suffix；prefix 叠放容器；size 走 className。
export function Input(
  props: CommonProps & {
    value?: string
    onChange?: (e: { target: { value: string } }) => void
    placeholder?: string
    disabled?: boolean
    allowClear?: boolean
    prefix?: ReactNode
    type?: string
    size?: string
  },
) {
  const { key, onEmit } = useSemiControlledBridge(props.value ?? '')
  const emit = (nv: string) => {
    onEmit(nv)
    props.onChange?.({ target: { value: nv } })
  }
  const field = createElement(EvTextField, {
    key,
    id: anchorId(props['data-test']),
    value: props.value ?? '',
    // eview 参数序=（新值, 旧值, event）——gate/matrix 定案。
    onChange: (nv: string) => emit(nv),
    placeholder: props.placeholder,
    disabled: props.disabled,
    type: props.type === 'password' ? 'password' : 'text',
    // 自带 validator 体系一律不传（FA-06：校验权威在自研引擎）。
    suffix:
      props.allowClear && (props.value ?? '') !== '' && !props.disabled
        ? createElement(
            'span',
            {
              className: 'ub-input-clear',
              role: 'button',
              'aria-label': 'clear',
              onClick: () => emit(''),
            },
            '×',
          )
        : undefined,
    className: [props.className, props.size === 'small' ? 'ub-size-small' : ''].filter(Boolean).join(' ') || undefined,
    style: props.style,
  })
  if (props.prefix == null) return field
  return createElement(
    'span',
    { className: 'ub-input-affix' },
    createElement('span', { className: 'ub-input-prefix' }, props.prefix),
    field,
  )
}

// ===== InputNumber → Spinner =====
// 缺口：min/max 默认 0/100 非无界（必须显式传）；无效输入走 onInputError（不上抛，
// 对齐 antd 无效输入不回调）；无 placeholder/controls 隐藏（记录，样式层再收）。
export function InputNumber(
  props: CommonProps & {
    value?: number | null
    onChange?: (v: number | undefined) => void
    min?: number
    max?: number
    controls?: boolean
    placeholder?: string
    disabled?: boolean
  },
) {
  const { key, onEmit } = useSemiControlledBridge(props.value ?? null)
  return createElement(EvSpinner, {
    key,
    id: anchorId(props['data-test']),
    value: props.value ?? undefined,
    min: props.min ?? Number.MIN_SAFE_INTEGER,
    max: props.max ?? Number.MAX_SAFE_INTEGER,
    onChange: (v: number | string) => {
      const n = typeof v === 'number' ? v : Number(v)
      const next = Number.isFinite(n) ? n : undefined
      onEmit(next ?? null)
      props.onChange?.(next)
    },
    onInputError: () => undefined, // 无效输入不上抛（antd 对齐）
    disabled: props.disabled,
    className: props.className,
    style: props.style,
  })
}

// ===== Select → InputSelect =====
// showSearch=false → onlySelect:true（禁输入纯下拉）；清空=onClear+onChange(undefined)。
export function Select(
  props: CommonProps & {
    options?: Array<{ label?: ReactNode; value: string | number }>
    value?: string | number
    onChange?: (v: string | number | undefined) => void
    allowClear?: boolean
    onClear?: () => void
    showSearch?: boolean
    placeholder?: string
    disabled?: boolean
    size?: string
  },
) {
  const { key, onEmit } = useSemiControlledBridge(props.value)
  return createElement(EvInputSelect, {
    key,
    id: anchorId(props['data-test']),
    options: (props.options ?? []).map((o) => ({
      text: typeof o.label === 'string' ? o.label : String(o.label ?? o.value),
      value: o.value,
    })),
    value: props.value,
    // gate R1/R2 定案参数序：(新值, 旧值, 'select'|'input')。
    onChange: (nv: string | number) => {
      onEmit(nv)
      props.onChange?.(nv)
    },
    enableClear: props.allowClear,
    onClear: () => {
      onEmit(undefined)
      props.onClear?.()
      props.onChange?.(undefined)
    },
    onlySelect: !props.showSearch,
    placeholder: props.placeholder,
    disabled: props.disabled,
    className: [props.className, props.size === 'small' ? 'ub-size-small' : ''].filter(Boolean).join(' ') || undefined,
    style: props.style,
  })
}
