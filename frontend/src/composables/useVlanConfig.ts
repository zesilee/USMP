import { ref } from 'vue'
import { getConfig, setConfig, getYangSchema } from '../api'
import type { Field } from '../utils/crdSchemaParser'

// 华为 VLAN 配置路径。含 "vlan:" → 后端 convertToTypedStruct 路由到 convertMapToHuaweiVlan。
export const VLAN_PATH = 'huawei-vlan:vlan/vlans'

// 从嵌套 schema（container vlans → list vlan）里取出「单个 VLAN」的字段集。
export function extractVlanItemFields(schema: any): Field[] {
  const top: Field[] = schema?.fields ?? []
  const vlansGroup = top.find((f) => f.path.endsWith('/vlans'))
  const vlanList = (vlansGroup?.fields ?? []).find((f) => f.path.endsWith('/vlan'))
  return vlanList?.fields ?? []
}

// 把运行配置里的 vlan 负载归一化为行数组（兼容 {vlans:[...]}、数组、以 id 为键的 map）。
export function extractVlanRows(data: any): Record<string, any>[] {
  const payload = data?.data ?? data
  const vlans = payload?.vlans ?? payload?.Vlan ?? payload
  if (Array.isArray(vlans)) return vlans
  if (vlans && typeof vlans === 'object') {
    return Object.entries(vlans).map(([k, v]) =>
      typeof v === 'object' && v !== null ? { id: Number(k) || k, ...(v as object) } : { id: k },
    )
  }
  return []
}

export function useVlanConfig() {
  const fields = ref<Field[]>([])
  const vlans = ref<Record<string, any>[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function loadSchema() {
    // 走 api 客户端（绝对 baseURL）——staging 的 nginx 不代理 /api，裸相对 fetch 会拿到 index.html。
    const res = await getYangSchema('vlan', 'nested')
    fields.value = extractVlanItemFields(res.data?.data)
  }

  async function loadVlans(ip: string) {
    loading.value = true
    error.value = null
    try {
      const res = await getConfig(ip, VLAN_PATH)
      vlans.value = extractVlanRows(res.data?.data)
    } catch (e: any) {
      // 设备离线/未连接：列表置空并透出错误（R08 降级，仍可新增）
      error.value = e?.response?.data?.message || e?.message || '读取失败'
      vlans.value = []
    } finally {
      loading.value = false
    }
  }

  // 下发单个 VLAN（声明式，后端对账到设备）。
  async function saveVlan(ip: string, vlan: Record<string, any>) {
    await setConfig(ip, VLAN_PATH, { vlans: [vlan] })
  }

  return { fields, vlans, loading, error, loadSchema, loadVlans, saveVlan }
}
