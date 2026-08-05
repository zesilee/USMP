import type { Field } from './crdSchemaParser'

// 读通道拆分（真机回归）：列表读走 config-only 快通道后，行数据不含 config=false
// 状态；详情打开时按需单行读状态（include_state=true），经本函数只把「只读字段」
// 的设备值合并进表单展示。铁律：绝不触碰可编辑字段——用户未保存的草稿原样保留。
export function mergeReadonlyState(
  fields: Field[],
  formData: Record<string, any>,
  stateRow: Record<string, any> | undefined | null,
): void {
  if (!stateRow || typeof stateRow !== 'object') return
  for (const f of fields) {
    const key = f.path.split('/').filter(Boolean).pop() || f.path
    const incoming = stateRow[key]
    if (incoming === undefined) continue
    if (f.readonly) {
      formData[key] = incoming
      continue
    }
    // 可写 group：深合并其只读子叶（statistics 类容器常配置+状态混排）。
    if (f.type === 'group' && f.fields?.length && incoming && typeof incoming === 'object') {
      const cur = formData[key] && typeof formData[key] === 'object' ? formData[key] : {}
      mergeReadonlyState(f.fields, cur, incoming)
      if (Object.keys(cur).length > 0) formData[key] = cur
    }
  }
}
