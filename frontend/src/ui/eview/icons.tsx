// EviewUI 图标（FA-04 / 组 5.3）：语义名（与 antd-backend/icons.ts 集合严格
// 一致，守护测试钉住）→ @nce/icon-plus 组件。具名导出的确切形态外网不可验
// （icon-plus 无 vendor d.ts）——运行时按候选名自适应（IconPlus 前缀 / 裸名），
// 缺名回落问号占位（R12 规范占位，SHALL NOT 空白）；实名由内网校准侦察用例
// 验证（RENDER 同款方法论）。实心变体经 type="filled"（矩阵定案）。
import { createElement, Component, type ComponentType, type ReactNode } from 'react'
import * as IconPlusNS from '@nce/icon-plus'
import { ICON_MAP } from './iconMap'

// 语义图标对外 props（antd 版对等面：className/onClick/data-test 透传）。
type IconProps = {
  className?: string
  onClick?: (e: { stopPropagation: () => void }) => void
} & Record<string, unknown>
// E2E 实证：icon-plus=纯具名导出 ESM（无 default），rolldown 生产构建严格
// 拦默认导入——命名空间导入取用。
const NS = (((IconPlusNS as { default?: unknown })?.default ?? IconPlusNS) ?? {}) as unknown as Record<
  string,
  ComponentType<IconProps>
>

function resolveIcon(name: string): ComponentType<IconProps> | undefined {
  return NS[`IconPlus${name}`] ?? NS[name]
}

// 图标级错误边界（R08）：icon-plus 组件 ref 阶段抛错走 React 不捕获的恢复
// 循环路径直至页面崩溃（E2E 实录）——单图标失败降级占位 span，不炸树不循环。
class IconBoundary extends Component<{ semantic: string; children?: ReactNode }, { failed: boolean }> {
  state = { failed: false }
  static getDerivedStateFromError() {
    return { failed: true }
  }
  render() {
    if (this.state.failed) return createElement('span', { 'data-icon-error': this.props.semantic })
    return this.props.children
  }
}

function makeIcon(semantic: string): ComponentType<IconProps> {
  const meta = ICON_MAP[semantic]
  return function EvIcon(props: IconProps) {
    const C = meta ? resolveIcon(meta.name) : undefined
    if (C)
      return createElement(
        IconBoundary,
        { semantic },
        createElement(C, meta!.filled ? { ...props, type: 'filled' } : props),
      )
    const Q = resolveIcon('IcPublicQuestionmarkCircle')
    if (Q) return createElement(Q, { ...props, 'data-icon-missing': semantic })
    return createElement('span', { ...props, 'data-icon-missing': semantic })
  }
}

export const ArrowDownIcon = makeIcon('ArrowDownIcon')
export const ArrowUpIcon = makeIcon('ArrowUpIcon')
export const BellIcon = makeIcon('BellIcon')
export const ConnectionIcon = makeIcon('ConnectionIcon')
export const DataLineIcon = makeIcon('DataLineIcon')
export const DeleteIcon = makeIcon('DeleteIcon')
export const DocumentIcon = makeIcon('DocumentIcon')
export const ExpandIcon = makeIcon('ExpandIcon')
export const FoldIcon = makeIcon('FoldIcon')
export const KeyIcon = makeIcon('KeyIcon')
export const MonitorIcon = makeIcon('MonitorIcon')
export const PlusIcon = makeIcon('PlusIcon')
export const RefreshIcon = makeIcon('RefreshIcon')
export const RefreshRightIcon = makeIcon('RefreshRightIcon')
export const SearchIcon = makeIcon('SearchIcon')
export const SettingIcon = makeIcon('SettingIcon')
export const ShareIcon = makeIcon('ShareIcon')
export const ToolsIcon = makeIcon('ToolsIcon')
export const WarningFilledIcon = makeIcon('WarningFilledIcon')
export const CheckIcon = makeIcon('CheckIcon')
export const CloseIcon = makeIcon('CloseIcon')
export const PlaceholderIcon = makeIcon('PlaceholderIcon')
