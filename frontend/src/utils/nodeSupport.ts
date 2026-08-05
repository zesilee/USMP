// FE-24：设备不支持节点（node-unsupported）判定。
// 判定一律以响应体结构化 reason 字段为准（design D5），禁止错误文案字符串匹配。
// 后端契约（BR-12）：GET/POST /config 命中不支持路径时返回 HTTP 200 统一信封
// `{code:!=0, success:false, message, data:{reason:"node-unsupported"}}`——axios
// 不 reject，调用方须在成功回调里检查信封；非 200 兜底形态经错误对象判定。

/** 判定响应信封是否「设备不支持该节点」：success=false 且 data.reason 命中。 */
export function nodeUnsupportedFromEnvelope(resData: any): boolean {
  return resData?.success === false && resData?.data?.reason === 'node-unsupported'
}

/** 判定 axios 错误对象（网络层包裹/非 200 兜底）是否同款语义。 */
export function nodeUnsupportedFromError(e: any): boolean {
  return e?.response?.data?.data?.reason === 'node-unsupported'
}
