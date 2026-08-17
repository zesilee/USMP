// EviewUI 桥 · 展示组（组 4.1 之一）：Tag/Badge/Breadcrumb/Empty/Drawer/Modal。
// 对外 props = antd 形态（业务调用点零改动），内部转换为 EviewUI props
// （映射依据=component-matrix，勿凭空改）。FA-05 锚点：闭合 props 组件走
// 组件 id prop（anchorId 映射）。
import { createElement, type ReactNode, type CSSProperties } from 'react'
import TagMod from '@nce/eview-react/Tag'
import BadgeMod from '@nce/eview-react/Badge'
import CrumbsMod from '@nce/eview-react/Crumbs'
import EmptyMod from '@nce/eview-react/Empty'
import DrawerMod from '@nce/eview-react/Drawer'
import DialogMod from '@nce/eview-react/Dialog'
import { anchorId } from '../../bridge'

function pick(mod: unknown): never {
  return ((mod as { default?: unknown }).default ?? mod) as never
}
const EvTag = pick(TagMod)
const EvBadge = pick(BadgeMod)
const EvCrumbs = pick(CrumbsMod)
const EvEmpty = pick(EmptyMod)
const EvDrawer = pick(DrawerMod)
const EvDialog = pick(DialogMod)

interface CommonProps {
  'data-test'?: string
  className?: string
  style?: CSSProperties
}

// ===== Tag：antd 语义色 → eview 色名（error→danger、processing→primary）=====
const TAG_COLOR: Record<string, string> = {
  success: 'success',
  warning: 'warning',
  error: 'danger',
  processing: 'primary',
  default: 'default',
}

export function Tag(props: CommonProps & { color?: string; children?: ReactNode }) {
  const { color, children } = props
  return createElement(
    EvTag,
    {
      id: anchorId(props['data-test']),
      color: color ? (TAG_COLOR[color] ?? color) : 'default',
      round: false, // antd 观感为微圆角，eview 默认全圆
      className: props.className,
      style: props.style,
    },
    children,
  )
}

// ===== Badge：count→content；size=small 用 badgeStyle 固化 =====
export function Badge(props: CommonProps & { count?: ReactNode; size?: string; children?: ReactNode }) {
  return createElement(
    EvBadge,
    {
      id: anchorId(props['data-test']),
      content: props.count,
      max: 99,
      badgeStyle: props.size === 'small' ? { transform: 'scale(0.85)' } : undefined,
      className: props.className,
    },
    props.children,
  )
}

// ===== Breadcrumb：items[{title}]→data[{title:string}]；separator→seprator（其拼写）=====
export function Breadcrumb(props: CommonProps & { items?: Array<{ title?: ReactNode }>; separator?: string }) {
  return createElement(EvCrumbs, {
    id: anchorId(props['data-test']),
    data: (props.items ?? []).map((i) => ({ title: typeof i.title === 'string' ? i.title : String(i.title ?? '') })),
    seprator: props.separator,
    className: props.className,
  })
}

// ===== Empty =====
export function Empty(props: CommonProps & { description?: ReactNode }) {
  return createElement(EvEmpty, {
    id: anchorId(props['data-test']),
    description: props.description,
    className: props.className,
  })
}

// ===== Drawer：open→visible；width 仅收 number（'50%' 折算视口）=====
export function toPx(width: number | string | undefined, viewport: number): number | undefined {
  if (width == null) return undefined
  if (typeof width === 'number') return width
  const pct = /^([\d.]+)%$/.exec(width)
  if (pct) return Math.round((parseFloat(pct[1]) / 100) * viewport)
  const n = parseFloat(width)
  return Number.isFinite(n) ? n : undefined
}

export function Drawer(
  props: CommonProps & {
    open?: boolean
    onClose?: () => void
    title?: ReactNode
    width?: number | string
    maskClosable?: boolean
    children?: ReactNode
  },
) {
  return createElement(
    EvDrawer,
    {
      id: anchorId(props['data-test']),
      visible: !!props.open,
      onClose: () => props.onClose?.(),
      title: props.title,
      width: toPx(props.width, typeof window !== 'undefined' ? window.innerWidth : 1280),
      isClickMask: props.maskClosable !== false,
      className: props.className,
    },
    props.children,
  )
}

// ===== Modal→Dialog：okText/onOk→buttons；footer=null→无底栏；width→size=[w,null] =====
export function Modal(
  props: CommonProps & {
    open?: boolean
    title?: ReactNode
    onCancel?: () => void
    onOk?: () => void
    okText?: ReactNode
    footer?: ReactNode | null
    confirmLoading?: boolean
    closable?: boolean
    destroyOnHidden?: boolean
    width?: number | string
    children?: ReactNode
  },
) {
  const showFooter = props.footer !== null && (props.onOk != null || props.okText != null)
  return createElement(
    EvDialog,
    {
      id: anchorId(props['data-test']),
      isOpen: !!props.open,
      title: props.title,
      onClose: () => props.onCancel?.(),
      // confirmLoading（eview 无对应）：loading 期间吞掉 onOk 防重复提交。
      buttons: showFooter
        ? [{ text: props.okText, onClick: () => (props.confirmLoading ? undefined : props.onOk?.()) }]
        : undefined,
      closable: props.closable !== false,
      destroyOnClose: props.destroyOnHidden !== false,
      size: props.width != null ? [toPx(props.width, typeof window !== 'undefined' ? window.innerWidth : 1280), null] : undefined,
      movable: false, // eview 默认可拖动，对齐 antd 不可拖
      className: props.className,
    },
    props.children,
  )
}
