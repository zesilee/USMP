// EviewUI 桥 · 展示组（组 4.1 之一）：Tag/Badge/Breadcrumb/Empty/Drawer/Modal。
// 对外 props = antd 形态（业务调用点零改动），内部转换为 EviewUI props
// （映射依据=component-matrix，勿凭空改）。FA-05 锚点：闭合 props 组件走
// 组件 id prop（anchorId 映射）。
import { createElement, type ReactNode, type CSSProperties } from 'react'
import BadgeMod from '@nce/eview-react/Badge'
import CrumbsMod from '@nce/eview-react/Crumbs'
import EmptyMod from '@nce/eview-react/Empty'
import DrawerMod from '@nce/eview-react/Drawer'
import DialogMod from '@nce/eview-react/Dialog'
import { anchorId, pickDefault } from '../../bridge'

const EvBadge = pickDefault(BadgeMod)
const EvCrumbs = pickDefault(CrumbsMod)
const EvEmpty = pickDefault(EmptyMod)
const EvDrawer = pickDefault(DrawerMod)
const EvDialog = pickDefault(DialogMod)

interface CommonProps {
  'data-test'?: string
  className?: string
  style?: CSSProperties
}

// ===== Tag：桥自绘柔和标签（内网目视定案）=====
// eview Tag 直名色（green/red/orange…）=饱和实底、块面突兀且文字对比不足
// ——Tag 为纯展示元素，改自绘 span.ub-tag（浅底+同色系深字，theme.scss
// 承样式），不再经 EvTag、零 eview CSS 对抗。antd 语义色与直名色统一归
// 六档色调类。
const TAG_TONE: Record<string, string> = {
  success: 'green',
  green: 'green',
  warning: 'orange',
  orange: 'orange',
  error: 'red',
  red: 'red',
  processing: 'blue',
  blue: 'blue',
  cyan: 'cyan',
}

export function Tag(props: CommonProps & { color?: string; children?: ReactNode }) {
  const tone = TAG_TONE[props.color ?? ''] ?? 'default'
  return createElement(
    'span',
    {
      id: anchorId(props['data-test']),
      className: ['ub-tag', `ub-tag-${tone}`, props.className].filter(Boolean).join(' '),
      style: props.style,
    },
    props.children,
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

// ===== Breadcrumb：items[{title}]→data[{title:string}]；separator→seprator（eview 原拼写）=====
export function Breadcrumb(props: CommonProps & { items?: Array<{ title?: ReactNode }>; separator?: string }) {
  return createElement(EvCrumbs, {
    id: anchorId(props['data-test']),
    data: (props.items ?? []).map((i) => ({ title: typeof i.title === 'string' ? i.title : String(i.title ?? '') })),
    seprator: props.separator,
    className: props.className,
  })
}

// ===== Empty =====
export function Empty(props: CommonProps & { description?: ReactNode; children?: ReactNode }) {
  const empty = createElement(EvEmpty, {
    id: anchorId(props['data-test']),
    description: props.description,
    className: props.className,
  })
  // antd Empty 支持 children（描述下操作位，如「去新建」按钮）——eview Empty
  // 无槽位，桥在其后追加渲染。
  if (props.children == null) return empty
  return createElement('div', { className: 'ub-empty-wrap' }, empty,
    createElement('div', { className: 'ub-empty-extra' }, props.children))
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
    // eview Dialog 无遮罩点击关闭配置（默认即不关，对齐 antd maskClosable:false
    // 的效果）；类型收下保调用点兼容。
    maskClosable?: boolean
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
    // 自定义 footer（JSX）：eview Dialog 无 footer 槽位，随 children 尾部渲染
    // （视觉由 .ub-modal-footer 类衔接）。
    props.footer != null && props.footer !== false
      ? createElement('div', { key: 'ub-body' }, props.children,
          createElement('div', { className: 'ub-modal-footer', key: 'ub-footer' }, props.footer))
      : props.children,
  )
}
