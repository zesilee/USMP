import { ref, onMounted, onUnmounted } from 'vue'
import { KubeConfig, CustomObjectsApi } from '@kubernetes/client-node'

export type ConfigPhase = 'Pending' | 'Updating' | 'Ready' | 'Failed'

export interface CRDItem {
  apiVersion: string
  kind: string
  metadata: {
    name: string
    creationTimestamp: string
    annotations?: Record<string, string>
    labels?: Record<string, string>
  }
  spec: Record<string, any>
  status?: {
    phase?: ConfigPhase
    lastSyncTime?: string
    error?: string
    [key: string]: any
  }
}

export interface UseK8sCRDOptions {
  autoWatch?: boolean
  autoList?: boolean
  namespace?: string
}

export function useK8sCRD(group: string, version: string, plural: string, options: UseK8sCRDOptions = {}) {
  const { autoWatch = true, autoList = true, namespace } = options

  const items = ref<CRDItem[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const watchAbortController = ref<AbortController | null>(null)

  // Initialize KubeConfig
  const kc = new KubeConfig()

  // In development, try to load from default kubeconfig
  // In production, this will automatically load from the service account
  try {
    kc.loadFromDefault()
  } catch (e) {
    console.warn('Failed to load kubeconfig, K8s API may not be available:', e)
  }

  const client = kc.makeApiClient(CustomObjectsApi)

  // Get base URL for fetch requests that aren't supported by the client
  const getBaseURL = () => {
    const currentCluster = (kc as any).getCurrentCluster()
    return currentCluster?.server || 'http://localhost:8001'
  }

  // List CRD items
  const list = async () => {
    loading.value = true
    error.value = null
    try {
      if (namespace) {
        const res = await client.listNamespacedCustomObject(group, version, namespace, plural)
        items.value = (res.body as any).items || []
      } else {
        const res = await client.listClusterCustomObject(group, version, plural)
        items.value = (res.body as any).items || []
      }
    } catch (e: any) {
      error.value = e.message || 'Failed to list CRD items'
      console.error('List CRD error:', e)
    } finally {
      loading.value = false
    }
  }

  // Get a single CRD item
  const get = async (name: string) => {
    if (namespace) {
      const res = await client.getNamespacedCustomObject(group, version, namespace, plural, name)
      return res.body as CRDItem
    }
    const res = await client.getClusterCustomObject(group, version, plural, name)
    return res.body as CRDItem
  }

  // Create a new CRD item
  const create = async (body: Partial<CRDItem>) => {
    const fullBody = {
      apiVersion: `${group}/${version}`,
      kind: plural.charAt(0).toUpperCase() + plural.slice(1).replace(/s$/, ''),
      metadata: {
        name: body.metadata?.name || `item-${Date.now()}`,
        ...body.metadata,
      },
      spec: body.spec || {},
      ...body,
    }

    if (namespace) {
      const res = await client.createNamespacedCustomObject(group, version, namespace, plural, fullBody)
      return res.body as CRDItem
    }
    const res = await client.createClusterCustomObject(group, version, plural, fullBody)
    return res.body as CRDItem
  }

  // Update an existing CRD item
  const update = async (name: string, body: CRDItem) => {
    if (namespace) {
      const res = await client.replaceNamespacedCustomObject(group, version, namespace, plural, name, body)
      return res.body as CRDItem
    }
    const res = await client.replaceClusterCustomObject(group, version, plural, name, body)
    return res.body as CRDItem
  }

  // Delete a CRD item
  const remove = async (name: string) => {
    if (namespace) {
      await client.deleteNamespacedCustomObject(group, version, namespace, plural, name)
    } else {
      await client.deleteClusterCustomObject(group, version, plural, name)
    }
  }

  // Watch for real-time updates using native K8s watch API
  const startWatch = (onChange?: (type: 'ADDED' | 'MODIFIED' | 'DELETED', obj: CRDItem) => void) => {
    stopWatch()
    watchAbortController.value = new AbortController()

    const baseUrl = getBaseURL()
    const watchUrl = namespace
      ? `${baseUrl}/apis/${group}/${version}/namespaces/${namespace}/${plural}?watch=true`
      : `${baseUrl}/apis/${group}/${version}/${plural}?watch=true`

    fetch(watchUrl, {
      signal: watchAbortController.value.signal,
      headers: kc.getDefaultHeaders(),
    })
      .then(async (response) => {
        if (!response.ok) {
          throw new Error(`Watch HTTP error: ${response.status}`)
        }

        const reader = response.body?.getReader()
        if (!reader) {
          throw new Error('No response body reader')
        }

        const decoder = new TextDecoder()
        let buffer = ''

        while (true) {
          const { done, value } = await reader.read()
          if (done) break

          buffer += decoder.decode(value, { stream: true })

          // Process each line (K8s watch sends one JSON per line)
          const lines = buffer.split('\n')
          buffer = lines.pop() || ''

          for (const line of lines) {
            if (!line.trim()) continue

            try {
              const event = JSON.parse(line)
              const eventType = event.type as 'ADDED' | 'MODIFIED' | 'DELETED'
              const item = event.object as CRDItem

              // Update local cache
              if (eventType === 'DELETED') {
                items.value = items.value.filter(i => i.metadata.name !== item.metadata.name)
              } else if (eventType === 'ADDED') {
                if (!items.value.find(i => i.metadata.name === item.metadata.name)) {
                  items.value.push(item)
                }
              } else { // MODIFIED
                const idx = items.value.findIndex(i => i.metadata.name === item.metadata.name)
                if (idx >= 0) {
                  items.value[idx] = item
                } else {
                  items.value.push(item)
                }
              }

              onChange?.(eventType, item)
            } catch (parseError) {
              console.warn('Failed to parse watch event:', parseError)
            }
          }
        }
      })
      .catch((err) => {
        if (err.name !== 'AbortError') {
          console.error('Watch connection error:', err)
          // Auto-reconnect after 3 seconds
          setTimeout(() => startWatch(onChange), 3000)
        }
      })
  }

  const stopWatch = () => {
    if (watchAbortController.value) {
      watchAbortController.value.abort()
      watchAbortController.value = null
    }
  }

  // Get CRD OpenAPI Schema for dynamic form rendering
  const getSchema = async () => {
    const crdName = `${plural}.${group}`
    const baseUrl = getBaseURL()

    try {
      const res = await fetch(`${baseUrl}/apis/apiextensions.k8s.io/v1/customresourcedefinitions/${crdName}`, {
        headers: kc.getDefaultHeaders(),
      })

      if (!res.ok) {
        throw new Error(`HTTP ${res.status}: Failed to fetch CRD schema`)
      }

      const crd = await res.json()
      const versionDef = crd.spec.versions.find((v: any) => v.name === version)
      return versionDef?.schema?.openAPIV3Schema || null
    } catch (e) {
      console.error('Failed to get CRD schema:', e)
      return null
    }
  }

  // Lifecycle management
  onMounted(() => {
    if (autoList) {
      list()
    }
    if (autoWatch) {
      startWatch()
    }
  })

  onUnmounted(() => {
    stopWatch()
  })

  return {
    items,
    loading,
    error,
    list,
    get,
    create,
    update,
    remove,
    startWatch,
    stopWatch,
    getSchema,
  }
}
