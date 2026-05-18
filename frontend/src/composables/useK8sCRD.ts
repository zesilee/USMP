import { ref, onMounted, onUnmounted } from 'vue'

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
  baseUrl?: string
}

// Simple K8s API client using browser native fetch
class K8sClient {
  private baseUrl: string
  private headers: Record<string, string>

  constructor(baseUrl: string = '') {
    this.baseUrl = baseUrl
    this.headers = {
      'Content-Type': 'application/json',
    }
  }

  // In a browser environment, we rely on:
  // 1. kubectl proxy running for development
  // 2. ServiceAccount + kube-rbac-proxy for in-cluster
  // 3. Backend API proxy endpoint
  private async request<T>(path: string, options: RequestInit = {}): Promise<T> {
    const url = `${this.baseUrl}${path}`
    const response = await fetch(url, {
      ...options,
      headers: {
        ...this.headers,
        ...options.headers,
      },
    })

    if (!response.ok) {
      throw new Error(`K8s API Error: ${response.status} ${response.statusText}`)
    }

    return response.json()
  }

  // List custom objects
  async listCustomObject(group: string, version: string, plural: string, namespace?: string): Promise<{ items: CRDItem[] }> {
    const path = namespace
      ? `/apis/${group}/${version}/namespaces/${namespace}/${plural}`
      : `/apis/${group}/${version}/${plural}`
    return this.request(path)
  }

  // Get single custom object
  async getCustomObject(group: string, version: string, plural: string, name: string, namespace?: string): Promise<CRDItem> {
    const path = namespace
      ? `/apis/${group}/${version}/namespaces/${namespace}/${plural}/${name}`
      : `/apis/${group}/${version}/${plural}/${name}`
    return this.request(path)
  }

  // Create custom object
  async createCustomObject(group: string, version: string, plural: string, body: any, namespace?: string): Promise<CRDItem> {
    const path = namespace
      ? `/apis/${group}/${version}/namespaces/${namespace}/${plural}`
      : `/apis/${group}/${version}/${plural}`
    return this.request(path, {
      method: 'POST',
      body: JSON.stringify(body),
    })
  }

  // Replace custom object
  async replaceCustomObject(group: string, version: string, plural: string, name: string, body: any, namespace?: string): Promise<CRDItem> {
    const path = namespace
      ? `/apis/${group}/${version}/namespaces/${namespace}/${plural}/${name}`
      : `/apis/${group}/${version}/${plural}/${name}`
    return this.request(path, {
      method: 'PUT',
      body: JSON.stringify(body),
    })
  }

  // Delete custom object
  async deleteCustomObject(group: string, version: string, plural: string, name: string, namespace?: string): Promise<void> {
    const path = namespace
      ? `/apis/${group}/${version}/namespaces/${namespace}/${plural}/${name}`
      : `/apis/${group}/${version}/${plural}/${name}`
    await this.request(path, { method: 'DELETE' })
  }

  // Get CRD definition for schema
  async getCRD(group: string, plural: string): Promise<any> {
    const crdName = `${plural}.${group}`
    return this.request(`/apis/apiextensions.k8s.io/v1/customresourcedefinitions/${crdName}`)
  }

  // Get watch URL
  getWatchUrl(group: string, version: string, plural: string, namespace?: string): string {
    const path = namespace
      ? `/apis/${group}/${version}/namespaces/${namespace}/${plural}?watch=true`
      : `/apis/${group}/${version}/${plural}?watch=true`
    return `${this.baseUrl}${path}`
  }
}

export function useK8sCRD(group: string, version: string, plural: string, options: UseK8sCRDOptions = {}) {
  const { autoWatch = true, autoList = true, namespace, baseUrl = '' } = options

  const items = ref<CRDItem[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const watchAbortController = ref<AbortController | null>(null)

  // Initialize lightweight fetch-based K8s client
  // Works with:
  // - kubectl proxy (dev): http://localhost:8001
  // - Backend API proxy: /api/k8s
  // - In-cluster via ServiceAccount (needs kube-rbac-proxy for browser access)
  const client = new K8sClient(baseUrl || getDefaultBaseUrl())

  // Try to determine default base URL based on environment
  function getDefaultBaseUrl(): string {
    // In-cluster: expect backend proxy at /api/k8s
    // Dev: try kubectl proxy default
    if (typeof window !== 'undefined') {
      // For development with kubectl proxy
      if (window.location.hostname === 'localhost') {
        return '' // Assume proxy at same origin or path-based proxy
      }
      // For production in-cluster: use backend proxy endpoint
      return '/api/k8s'
    }
    return ''
  }

  // List CRD items
  const list = async () => {
    loading.value = true
    error.value = null
    try {
      const res = await client.listCustomObject(group, version, plural, namespace)
      items.value = res.items || []
    } catch (e: any) {
      error.value = e.message || 'Failed to list CRD items'
      console.error('List CRD error:', e)
    } finally {
      loading.value = false
    }
  }

  // Get a single CRD item
  const get = async (name: string): Promise<CRDItem> => {
    return client.getCustomObject(group, version, plural, name, namespace)
  }

  // Create a new CRD item
  const create = async (body: Partial<CRDItem>): Promise<CRDItem> => {
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

    return client.createCustomObject(group, version, plural, fullBody, namespace)
  }

  // Update an existing CRD item
  const update = async (name: string, body: CRDItem): Promise<CRDItem> => {
    return client.replaceCustomObject(group, version, plural, name, body, namespace)
  }

  // Delete a CRD item
  const remove = async (name: string): Promise<void> => {
    return client.deleteCustomObject(group, version, plural, name, namespace)
  }

  // Watch for real-time updates using native K8s watch API
  const startWatch = (onChange?: (type: 'ADDED' | 'MODIFIED' | 'DELETED', obj: CRDItem) => void) => {
    stopWatch()
    watchAbortController.value = new AbortController()

    const watchUrl = client.getWatchUrl(group, version, plural, namespace)

    fetch(watchUrl, {
      signal: watchAbortController.value.signal,
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
    try {
      const crd = await client.getCRD(group, plural)
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
