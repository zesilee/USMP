import { ref, computed, onMounted, onUnmounted } from 'vue'
import {
  getSchema,
  createConfig,
  updateConfig,
  deleteConfig,
  watchConfigs,
  type Schema,
  type DeviceConfigCR,
  type ConfigPhase
} from '../api/crd'

export function useDeviceConfig(device: string, module: string) {
  const configCR = ref<DeviceConfigCR | null>(null)
  const schema = ref<Schema | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  let eventSource: EventSource | null = null

  const isSyncing = computed(() => {
    const phase = configCR.value?.status?.phase
    return phase === 'Pending' || phase === 'Updating'
  })

  const currentPhase = computed(() => configCR.value?.status?.phase || 'Pending')

  async function loadSchema() {
    isLoading.value = true
    error.value = null
    try {
      schema.value = await getSchema(module)
    } catch (e: any) {
      error.value = e.message || '加载 Schema 失败'
      console.error('Failed to load schema:', e)
    } finally {
      isLoading.value = false
    }
  }

  async function save(desiredConfig: Record<string, any>): Promise<DeviceConfigCR> {
    isLoading.value = true
    error.value = null
    try {
      const cr: DeviceConfigCR = {
        apiVersion: 'network.usmp.io/v1',
        kind: module,
        metadata: {
          name: configCR.value?.metadata.name || `${device}-${module}-config`,
          device,
          annotations: configCR.value?.metadata.annotations
        },
        spec: desiredConfig
      }

      if (configCR.value) {
        configCR.value = await updateConfig(cr.metadata.name, cr)
      } else {
        configCR.value = await createConfig(cr)
      }
      return configCR.value
    } catch (e: any) {
      error.value = e.message || '保存配置失败'
      throw e
    } finally {
      isLoading.value = false
    }
  }

  async function remove(): Promise<void> {
    if (!configCR.value) return
    isLoading.value = true
    error.value = null
    try {
      await deleteConfig(configCR.value.metadata.name)
      configCR.value = null
    } catch (e: any) {
      error.value = e.message || '删除配置失败'
      throw e
    } finally {
      isLoading.value = false
    }
  }

  function connectWatch() {
    if (eventSource) {
      eventSource.close()
    }

    eventSource = watchConfigs(device, module)

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        if (data.type === 'config') {
          configCR.value = data.payload
        }
      } catch (e) {
        console.error('Failed to parse SSE message:', e)
      }
    }

    eventSource.onerror = (e) => {
      console.error('SSE connection error:', e)
      error.value = '配置同步连接断开'
    }
  }

  function refresh() {
    loadSchema()
    connectWatch()
  }

  onMounted(() => {
    loadSchema()
    connectWatch()
  })

  onUnmounted(() => {
    if (eventSource) {
      eventSource.close()
      eventSource = null
    }
  })

  return {
    configCR,
    schema,
    isLoading,
    isSyncing,
    currentPhase,
    error,
    save,
    remove,
    refresh
  }
}
