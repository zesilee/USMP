import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useConfigPage, useNativeModules } from '../../src/composables/useConfigPage'

// Mock fetch globally
global.fetch = vi.fn()

describe('useConfigPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('business config mode', () => {
    it('should detect business config for known modules', () => {
      const { configType } = useConfigPage('vlan')
      expect(configType).toBe('business')
    })

    it('should return correct title for business configs', () => {
      const { title } = useConfigPage('vlan')
      expect(title.value).toBe('VLAN 配置')
    })

    it('should expose crd methods for business configs', () => {
      const { list, get, create, update, remove } = useConfigPage('vlan')
      expect(typeof list).toBe('function')
      expect(typeof get).toBe('function')
      expect(typeof create).toBe('function')
      expect(typeof update).toBe('function')
      expect(typeof remove).toBe('function')
    })

    it('should provide listByDevice filter method', async () => {
      const { listByDevice } = useConfigPage('vlan')
      expect(typeof listByDevice).toBe('function')
    })

    it('should provide getSchema method', () => {
      const { getSchema } = useConfigPage('vlan')
      expect(typeof getSchema).toBe('function')
    })

    it('should have reactive schema and loading states', () => {
      const { schema, schemaLoading, schemaError } = useConfigPage('vlan')
      expect(schema.value).toEqual([])
      expect(schemaLoading.value).toBe(false)
      expect(schemaError.value).toBe(null)
    })
  })

  describe('native config mode', () => {
    it('should detect native config for unknown modules', () => {
      const { configType } = useConfigPage('openconfig-interfaces')
      expect(configType).toBe('native')
    })

    it('should expose crd methods for native configs', () => {
      const { list, get, create, update, remove } = useConfigPage('openconfig-interfaces')
      expect(typeof list).toBe('function')
      expect(typeof get).toBe('function')
      expect(typeof create).toBe('function')
      expect(typeof update).toBe('function')
      expect(typeof remove).toBe('function')
    })

    it('should provide listByDevice filter method for native configs', async () => {
      const { listByDevice } = useConfigPage('openconfig-interfaces')
      expect(typeof listByDevice).toBe('function')
    })

    it('should provide getSchema method that fetches from yang API', () => {
      const { getSchema } = useConfigPage('openconfig-interfaces')
      expect(typeof getSchema).toBe('function')
    })

    it('should have reactive schema and loading states', () => {
      const { schema, schemaLoading, schemaError } = useConfigPage('openconfig-interfaces')
      expect(schema.value).toEqual([])
      expect(schemaLoading.value).toBe(false)
      expect(schemaError.value).toBe(null)
    })
  })

  describe('getSchema for native configs', () => {
    it('should fetch schema from yang API and update states', async () => {
      const mockSchema = {
        title: 'Interface Configuration',
        fields: [
          { path: 'name', type: 'string', label: 'Name' },
          { path: 'enabled', type: 'boolean', label: 'Enabled' },
        ],
      }

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockSchema),
      } as Response)

      const { getSchema, schema, schemaLoading, title, schemaError } = useConfigPage('openconfig-interfaces')

      const promise = getSchema()
      expect(schemaLoading.value).toBe(true)

      const result = await promise

      expect(fetch).toHaveBeenCalledWith('/api/v1/yang/schema/openconfig-interfaces')
      expect(schemaLoading.value).toBe(false)
      expect(title.value).toBe('Interface Configuration')
      expect(schema.value).toEqual(mockSchema.fields)
      expect(result).toEqual(mockSchema.fields)
      expect(schemaError.value).toBe(null)
    })

    it('should handle fetch errors gracefully', async () => {
      vi.mocked(fetch).mockRejectedValueOnce(new Error('Network error'))

      const { getSchema, schemaLoading, schemaError } = useConfigPage('openconfig-interfaces')

      await expect(getSchema()).rejects.toThrow('Network error')

      expect(schemaLoading.value).toBe(false)
      expect(schemaError.value).toBe('Network error')
    })

    it('should handle non-ok HTTP responses', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: false,
        status: 404,
      } as Response)

      const { getSchema, schemaError } = useConfigPage('openconfig-interfaces')

      await expect(getSchema()).rejects.toThrow('HTTP 404: Failed to fetch schema')
      expect(schemaError.value).toBe('HTTP 404: Failed to fetch schema')
    })
  })
})

describe('useNativeModules', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should initialize with empty state', () => {
    const { modules, loading, error } = useNativeModules()
    expect(modules.value).toEqual([])
    expect(loading.value).toBe(false)
    expect(error.value).toBe(null)
  })

  it('should load modules from yang API', async () => {
    const mockModules = {
      models: [
        { name: 'openconfig-interfaces', title: 'Interfaces', vendor: 'OpenConfig' },
        { name: 'openconfig-vlan', title: 'VLANs', vendor: 'OpenConfig' },
        { name: 'huawei-if', title: 'Huawei Interface', vendor: 'Huawei' },
      ],
    }

    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve(mockModules),
    } as Response)

    const { loadModules, modules, loading, error } = useNativeModules()

    const promise = loadModules()
    expect(loading.value).toBe(true)

    await promise

    expect(fetch).toHaveBeenCalledWith('/api/v1/yang/modules')
    expect(loading.value).toBe(false)
    expect(modules.value).toEqual(mockModules.models)
    expect(error.value).toBe(null)
  })

  it('should group modules by vendor', async () => {
    const mockModules = {
      models: [
        { name: 'oc-if', title: 'Interfaces', vendor: 'OpenConfig' },
        { name: 'oc-vlan', title: 'VLANs', vendor: 'OpenConfig' },
        { name: 'hw-if', title: 'Huawei IF', vendor: 'Huawei' },
      ],
    }

    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve(mockModules),
    } as Response)

    const { loadModules, groupedByVendor } = useNativeModules()
    await loadModules()

    const groups = groupedByVendor.value
    expect(groups.has('OpenConfig')).toBe(true)
    expect(groups.has('Huawei')).toBe(true)
    expect(groups.get('OpenConfig')?.length).toBe(2)
    expect(groups.get('Huawei')?.length).toBe(1)
  })

  it('should put modules without vendor into "其他" group', async () => {
    const mockModules = {
      models: [
        { name: 'unknown-module', title: 'Unknown Module', vendor: '' },
      ],
    }

    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve(mockModules),
    } as Response)

    const { loadModules, groupedByVendor } = useNativeModules()
    await loadModules()

    expect(groupedByVendor.value.has('其他')).toBe(true)
  })

  it('should handle load errors gracefully', async () => {
    vi.mocked(fetch).mockRejectedValueOnce(new Error('Failed to load'))

    const { loadModules, loading, error } = useNativeModules()
    await loadModules()

    expect(loading.value).toBe(false)
    expect(error.value).toBe('Failed to load')
  })
})
