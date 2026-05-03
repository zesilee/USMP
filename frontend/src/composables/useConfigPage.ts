import { ref, computed } from 'vue'
import { useK8sCRD } from './useK8sCRD'
import type { Field } from '../utils/crdSchemaParser'
import { parseCRDSchemaToFields } from '../utils/crdSchemaParser'

// Business CRD configuration mapping
const BUSINESS_CRDS: Record<string, { group: string; version: string; plural: string; title: string }> = {
  vlan: { group: 'biz.usmp.io', version: 'v1', plural: 'businessvlans', title: 'VLAN 配置' },
  interface: { group: 'biz.usmp.io', version: 'v1', plural: 'businessinterfaces', title: '接口配置' },
  route: { group: 'biz.usmp.io', version: 'v1', plural: 'businessroutes', title: '路由配置' },
  switch: { group: 'biz.usmp.io', version: 'v1', plural: 'businessswitches', title: '设备管理' },
}

export function useConfigPage(module: string) {
  const isBusinessConfig = !!BUSINESS_CRDS[module]
  const schema = ref<Field[]>([])
  const schemaLoading = ref(false)
  const schemaError = ref<string | null>(null)

  if (isBusinessConfig) {
    // Business config: use corresponding CRD
    const crdInfo = BUSINESS_CRDS[module]
    const crd = useK8sCRD(crdInfo.group, crdInfo.version, crdInfo.plural)
    const title = ref(crdInfo.title)

    // Wrapper for getSchema that also parses
    const getSchema = async (): Promise<Field[]> => {
      schemaLoading.value = true
      schemaError.value = null
      try {
        const rawSchema = await crd.getSchema()
        schema.value = parseCRDSchemaToFields(rawSchema)
        return schema.value
      } catch (e: any) {
        schemaError.value = e.message
        throw e
      } finally {
        schemaLoading.value = false
      }
    }

    // Convenience method to filter by device
    const listByDevice = async (deviceID: string) => {
      await crd.list()
      return crd.items.value.filter(item => item.spec.deviceID === deviceID)
    }

    return {
      ...crd,
      title,
      configType: 'business' as const,
      schema,
      schemaLoading,
      schemaError,
      getSchema,
      listByDevice,
    }
  }

  // Native config: use unified NativeDeviceConfig CRD + YANG schema API
  const crd = useK8sCRD('core.usmp.io', 'v1', 'nativedeviceconfigs')
  const title = ref(module)

  // Native configs get schema from backend YANG API (via proxy)
  const getSchema = async (): Promise<Field[]> => {
    schemaLoading.value = true
    schemaError.value = null
    try {
      const res = await fetch(`/api/v1/yang/schema/${module}`)
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}: Failed to fetch schema`)
      }
      const yangSchema = await res.json()
      title.value = yangSchema.title || module
      schema.value = yangSchema.fields || []
      return schema.value
    } catch (e: any) {
      schemaError.value = e.message
      throw e
    } finally {
      schemaLoading.value = false
    }
  }

  const listByDevice = async (deviceID: string) => {
    await crd.list()
    return crd.items.value.filter(
      item => item.spec.deviceID === deviceID && item.spec.module === module
    )
  }

  return {
    ...crd,
    title,
    configType: 'native' as const,
    schema,
    schemaLoading,
    schemaError,
    getSchema,
    listByDevice,
  }
}

// Convenience hook for getting available native modules
export function useNativeModules() {
  const modules = ref<{ name: string; title: string; vendor: string }[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  const loadModules = async () => {
    loading.value = true
    try {
      const res = await fetch('/api/v1/yang/modules')
      const data = await res.json()
      modules.value = data.models || []
    } catch (e: any) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  // Group modules by vendor
  const groupedByVendor = computed(() => {
    const groups = new Map<string, typeof modules.value>()
    for (const m of modules.value) {
      const vendor = m.vendor || '其他'
      if (!groups.has(vendor)) {
        groups.set(vendor, [])
      }
      groups.get(vendor)!.push(m)
    }
    return groups
  })

  return {
    modules,
    loading,
    error,
    loadModules,
    groupedByVendor,
  }
}

export type { Field } from '../utils/crdSchemaParser'
