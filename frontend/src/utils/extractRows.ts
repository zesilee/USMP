// 从运行配置归一化出行数组（兼容 {listKey:[...]}、数组、以主键为键的 map）。
// 自 useDeviceConfig（已退役）迁入；现存消费方：RpcExecuteTab 的 leafref 下拉取数。
export function extractRows(data: any, listKey: string, keyField: string): Record<string, any>[] {
  const payload = data?.data ?? data
  const rows = payload?.[listKey] ?? payload
  if (Array.isArray(rows)) return rows
  if (rows && typeof rows === 'object') {
    return Object.entries(rows).map(([k, v]) =>
      typeof v === 'object' && v !== null
        ? { [keyField]: isNaN(Number(k)) ? k : Number(k), ...(v as object) }
        : { [keyField]: k },
    )
  }
  return []
}
