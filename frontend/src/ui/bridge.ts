// 适配层桥接工具（FA-02 受控桥接层 / gate-conclusion 半受控档位，组 3.1）。
import { useRef } from 'react'

/**
 * 半受控兜底③档：key 重挂。dep 变化时返回新 key，强制底层半受控组件重建
 * （丢内部 state，以 props 当前值重新初始化）。用于「父级拒写/程序化重置」
 * 等受控回写（②档）压不住的场景——gate 结论：②为常规路径，③为兜底。
 */
export function useRemountKey(dep: unknown): number {
  const ref = useRef({ dep, key: 0 })
  if (!Object.is(ref.current.dep, dep)) ref.current = { dep, key: ref.current.key + 1 }
  return ref.current.key
}

/**
 * FA-05 测试锚点落点（三路约定）：
 * - 底层支持属性透传（Tab/SearchInput/LabelField/icon-plus）→ 桥直接透传 data-test；
 * - 其余闭合 props 组件 → 桥把 data-test 映射到组件 `id`（EviewUI 全组件有 id prop），
 *   或外包 wrapper div 承载（弹层类组件 wrapper 定位不到浮层本体时用 id）。
 * 本工具生成 id 映射形态；wrapper 形态由各桥按组件结构选用。
 */
export function anchorId(dataTest?: string): string | undefined {
  return dataTest ? `dt-${dataTest}` : undefined
}

/** 由 anchorId 反推 DOM 选择器（测试侧配套，见 FA-05 守护）。 */
export const ANCHOR_SELECTOR = (dataTest: string): string => `[data-test="${dataTest}"], #dt-${dataTest}`
