// 适配层桥接工具（FA-02 受控桥接层 / gate-conclusion 半受控档位，组 3.1）。
import { Children, isValidElement, useRef, useState, type ReactNode } from 'react'

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
/** JSX → 纯文本（label 文本化通道：Tab 标题/树节点 title/DropDown 项）。 */
export const textOf = (n: ReactNode): string => {
  if (typeof n === 'string' || typeof n === 'number') return String(n)
  if (isValidElement(n)) {
    const kids = (n.props as { children?: ReactNode }).children
    return Children.toArray(kids ?? [])
      .map((c) => textOf(c))
      .join('')
  }
  return n == null ? '' : String(n)
}

export function anchorId(dataTest?: string): string | undefined {
  return dataTest ? `dt-${dataTest}` : undefined
}

/** 由 anchorId 反推 DOM 选择器（测试侧配套，见 FA-05 守护）。 */
export const ANCHOR_SELECTOR = (dataTest: string): string => `[data-test="${dataTest}"], #dt-${dataTest}`

/**
 * 锚点属性回填观察器（组 7 E2E 定案）：桥把 data-test 经 anchorId 转成
 * id=dt-xxx 传给 eview 组件，但 eview 类组件不透传未知 props——DOM 上
 * data-test 属性系统性丢失，E2E/F4 的 data-test 契约（80 条）全部落空。
 * 单点修复：MutationObserver 监听新增节点，把 dt- 前缀 id 自动回填为
 * data-test 属性——22 桥与未来新桥零改动，契约零妥协。UiProvider 装配。
 */
export function installAnchorAttrObserver(doc: Document = document): () => void {
  const restore = (el: Element) => {
    if (el.id && el.id.startsWith('dt-') && !el.hasAttribute('data-test')) {
      el.setAttribute('data-test', el.id.slice(3))
    }
    el.querySelectorAll?.('[id^="dt-"]:not([data-test])').forEach((child) => {
      child.setAttribute('data-test', child.id.slice(3))
    })
  }
  restore(doc.body)
  const mo = new MutationObserver((muts) => {
    for (const m of muts) {
      m.addedNodes.forEach((n) => {
        if (n.nodeType === 1) restore(n as Element)
      })
    }
  })
  mo.observe(doc.body, { childList: true, subtree: true })
  return () => mo.disconnect()
}

/**
 * 半受控输入桥（gate 定案组合：②受控回写为常规路径 + ③key 重挂兜底）。
 * 底层组件 value 经 cWRP 同步（②档，gate 双轮实证）；但「父级拒写/改写为
 * 第三值」时 props.value 未变化 → cWRP 不触发 → 内部停留用户输入。本 hook
 * 以影子值检测该分歧并 bump key 强制重建（③档），使显示始终回到 props。
 *
 * 用法：const { key, onEmit } = useSemiControlledBridge(props.value)
 *   底层组件挂 key={key}；上抛回调里先 onEmit(新值) 再调业务 onChange。
 */
export function useSemiControlledBridge(value: unknown): {
  key: number
  onEmit: (emitted: unknown) => void
} {
  const ref = useRef<{ shadow: unknown; hasShadow: boolean; key: number; lastValue: unknown }>({
    shadow: null,
    hasShadow: false,
    key: 0,
    lastValue: value,
  })
  // 拒写场景父级不重渲染，检测逻辑（render 期）不会跑——onEmit 须主动
  // 触发一次桥渲染（与父级同批 setState 合并，接受路径零额外成本）。
  const [, force] = useState(0)
  const st = ref.current
  if (!Object.is(value, st.lastValue)) {
    // 父级把 value 改到了新值：若与影子一致=接受（正常受控回写）；
    // 不一致=改写为第三值——cWRP 会同步，无需重挂。两种情况影子都清。
    st.lastValue = value
    st.shadow = null
    st.hasShadow = false
  } else if (st.hasShadow && !Object.is(value, st.shadow)) {
    // 父级拒写（value 保持原值而用户已输入影子值）：重挂强制回到 props。
    st.key += 1
    st.shadow = null
    st.hasShadow = false
  }
  return {
    key: st.key,
    onEmit: (emitted: unknown) => {
      st.shadow = emitted
      st.hasShadow = true
      force((n) => n + 1)
    },
  }
}

/**
 * EviewUI 编译产物为 babel esModule interop（.default 承载组件）；stub/缺失
 * 场景（外网 skip 模式）安全返回 undefined，渲染前不崩（收集期防线）。
 */
export function pickDefault(mod: unknown): never {
  // vite 浏览器优化器对 CJS 的 interop 可产生多层 default 嵌套（F3-R3 实录：
  // happy-dom 单层剥即得组件，真浏览器下拿到 module 对象致「Element type is
  // invalid: got object」）——循环剥到组件（函数/forwardRef 对象无 default
  // 键自然停）为止。
  let cur: unknown = mod
  let guard = 0
  while (cur != null && typeof cur === 'object' && 'default' in (cur as object) && guard++ < 5) {
    const next = (cur as { default?: unknown }).default
    if (next == null) break
    cur = next
  }
  return (cur ?? undefined) as never
}
