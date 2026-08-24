// EviewUI 桥 · 收尾组（组 4.4）：Button/Spin/Tooltip/Popover/Alert。
// 对外 props = antd 形态；映射依据 = component-matrix + gate（勿凭空改）。
import { createElement, useEffect, useRef, useState, type ReactNode, type CSSProperties } from 'react'
import ButtonMod from '@nce/eview-react/Button'
import LoadingMod from '@nce/eview-react/Loading'
import TipBoxMod from '@nce/eview-react/TipBox'
import DivMessageMod from '@nce/eview-react/DivMessage'
import PageMessageMod from '@nce/eview-react/PageMessage'
import { anchorId, pickDefault } from '../../bridge'

const EvButton = pickDefault(ButtonMod)
const EvLoading = pickDefault(LoadingMod)
const EvTipBox = pickDefault(TipBoxMod)
const EvDivMessage = pickDefault(DivMessageMod)
const EvPageMessage = pickDefault(PageMessageMod)

interface CommonProps {
  'data-test'?: string
  className?: string
  style?: CSSProperties
}

// ===== Button：type→status；danger 优先 risk（primary+danger 组合 eview 表达不了）；
// loading/ghost 为 eview 缺口——loading=自绘 CSS spinner+禁点，ghost=样式类。=====
export function Button(
  props: CommonProps & {
    type?: string
    danger?: boolean
    ghost?: boolean
    icon?: ReactNode
    loading?: boolean
    disabled?: boolean
    size?: string
    title?: string
    onClick?: (e: unknown) => void
    children?: ReactNode
  },
) {
  // link+danger 不走 risk（内网实证：risk=实心红块，antd 语义=红字链接）
  // ——保持 text 形态 + ub-btn-link-danger 红字类。
  const status =
    props.danger && props.type !== 'link'
      ? 'risk'
      : props.type === 'primary'
        ? 'primary'
        : props.type === 'link'
          ? 'text'
          : 'default'
  return createElement(
    EvButton,
    {
      id: anchorId(props['data-test']),
      status,
      size: props.size === 'small' ? 'small' : props.size === 'large' ? 'large' : 'normal',
      disabled: props.disabled || props.loading,
      leftIcon: props.loading
        ? createElement('span', { className: 'ub-btn-spin', 'aria-label': 'loading' })
        : (props.icon as never),
      title: props.title,
      // eview onClick(event, additionalData) 双参——吞第二参对齐 antd 单参。
      onClick: (e: unknown) => {
        if (!props.loading) props.onClick?.(e)
      },
      className:
        [
          props.className,
          props.ghost ? 'ub-btn-ghost' : '',
          props.danger && props.type === 'link' ? 'ub-btn-link-danger' : '',
        ]
          .filter(Boolean)
          .join(' ') || undefined,
      style: props.style,
    },
    props.children,
  )
}

// ===== Spin → Loading（type=local；gate 实证缺 iconUrl 正常渲染 CSS 图标）=====
export function Spin(props: CommonProps & { tip?: string }) {
  return createElement(EvLoading, {
    id: anchorId(props['data-test']),
    isOpen: true,
    type: 'local',
    desc: props.tip,
    className: props.className,
  })
}

// ===== Tooltip → TipBox(hover)：antd title→content（TipBox 的 title 是弹层内标题，勿映错）=====
export function Tooltip(props: CommonProps & { title?: ReactNode; children?: ReactNode }) {
  return createElement(
    EvTipBox,
    {
      content: props.title,
      trigger: 'hover',
      type: 'simple',
      className: props.className,
    },
    props.children,
  )
}

// ===== Popover → TipBox(click)：matrix 已知限制——有 children 时 display 受控失效、
// 无 onOpenChange 等价；桥退化为非受控 + onClose 尽力通知（集成点重点校准，
// 业务侧（高级搜索面板）若受控确不可行则在组 5 接线时改非受控用法。=====
export function Popover(
  props: CommonProps & {
    content?: ReactNode
    trigger?: string
    open?: boolean
    onOpenChange?: (open: boolean) => void
    children?: ReactNode
  },
) {
  // 自绘气泡（组 7 E2E 定案）：TipBox 的 display 受控在真组件被忽略（matrix
  // 早标记的已知限制，高级搜索面板 E2E 直击：受控 open=true 面板不弹）——
  // 桥自绘：锚点 wrapper + 受控/非受控双态 + 点外关闭（window capture，
  // Tree 委托同款先例）。hover 触发不在业务用（列设置/高级搜索均 click）。
  const [innerOpen, setInnerOpen] = useState(false)
  const isControlled = props.open !== undefined
  const open = isControlled ? !!props.open : innerOpen
  const setOpen = (v: boolean) => {
    if (!isControlled) setInnerOpen(v)
    props.onOpenChange?.(v)
  }
  const wrapRef = useRef<HTMLSpanElement | null>(null)
  const openRef = useRef(open)
  openRef.current = open
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (!openRef.current) return
      const root = wrapRef.current
      const t = e.target as Node | null
      // teleport 到 body 的下拉弹层（ev_popup）视为「内」——点面板里下拉的
      // 选项曾被误判点外把面板关掉（E2E 238 实录）。
      if (t && (t as Element).closest?.('.ev_popup')) return
      if (root && t && !root.contains(t)) setOpen(false)
    }
    window.addEventListener('click', handler, true)
    return () => window.removeEventListener('click', handler, true)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])
  return createElement(
    'span',
    { className: ['ub-popover-anchor', props.className].filter(Boolean).join(' '), ref: wrapRef },
    createElement(
      'span',
      { className: 'ub-popover-trigger', onClick: () => setOpen(!open) },
      props.children,
    ),
    open
      ? createElement('div', { className: 'ub-popover', 'data-test': props['data-test'] }, props.content)
      : null,
  )
}

// ===== Alert：closable→DivMessage（关自动消失）；不可关→PageMessage（有 info 型）=====
const DIV_TYPE: Record<string, string> = { success: 'success', error: 'error', warning: 'warn', info: 'default' }
const PAGE_TYPE: Record<string, string> = { success: 'success', error: 'error', warning: 'warn', info: 'info' }

export function Alert(
  props: CommonProps & {
    type?: string
    message?: ReactNode
    showIcon?: boolean
    closable?: boolean
    onClose?: () => void
  },
) {
  const kind = props.type ?? 'info'
  if (props.closable) {
    return createElement(EvDivMessage, {
      id: anchorId(props['data-test']),
      text: typeof props.message === 'string' ? props.message : undefined,
      children: typeof props.message === 'string' ? undefined : props.message,
      type: DIV_TYPE[kind] ?? 'default',
      showIcon: props.showIcon !== false,
      closeIconDisplay: true,
      // gate 定案陷阱：默认 10s 自动消失——常驻横幅必须显式关闭。
      enableDisposeTimeOut: false,
      onClose: () => props.onClose?.(),
      className: props.className,
    })
  }
  return createElement(EvPageMessage, {
    id: anchorId(props['data-test']),
    text: props.message,
    type: PAGE_TYPE[kind] ?? 'info',
    className: props.className,
  })
}
