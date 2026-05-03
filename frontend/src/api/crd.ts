import axios from 'axios'

export type ConfigPhase = 'Pending' | 'Updating' | 'Ready' | 'Failed'

export interface Field {
  path: string
  type: 'string' | 'number' | 'boolean' | 'enum' | 'group'
  label: string
  placeholder?: string
  required?: boolean
  pattern?: string
  readonly?: boolean
  options?: { label: string; value: string | number }[]
  fields?: Field[]
}

export interface Schema {
  module: string
  title: string
  fields: Field[]
  listFields: Field[]
}

export interface DeviceConfigCR {
  apiVersion: string
  kind: string
  metadata: {
    name: string
    device: string
    creationTimestamp?: string
    annotations?: Record<string, string>
  }
  spec: Record<string, any>
  status?: {
    phase: ConfigPhase
    message?: string
    lastSyncTime?: string
  }
}

const api = axios.create({
  baseURL: '/api/crd',
  timeout: 15000
})

export async function getSchema(module: string): Promise<Schema> {
  const res = await api.get(`/schema/${module}`)
  return res.data
}

export async function createConfig(cr: DeviceConfigCR): Promise<DeviceConfigCR> {
  const res = await api.post('/configs', cr)
  return res.data
}

export async function updateConfig(name: string, cr: DeviceConfigCR): Promise<DeviceConfigCR> {
  const res = await api.put(`/configs/${name}`, cr)
  return res.data
}

export async function deleteConfig(name: string): Promise<void> {
  await api.delete(`/configs/${name}`)
}

export function watchConfigs(device: string, module: string): EventSource {
  const url = `/api/crd/watch/${device}/${module}`
  return new EventSource(url)
}
