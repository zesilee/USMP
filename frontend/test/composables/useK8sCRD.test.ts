import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useK8sCRD } from '../../src/composables/useK8sCRD'

// Mock the kubernetes client
vi.mock('@kubernetes/client-node', () => {
  class MockKubeConfig {
    loadFromDefault = vi.fn()
    makeApiClient = vi.fn().mockReturnValue({
      listClusterCustomObject: vi.fn().mockResolvedValue({ body: { items: [] } }),
      getClusterCustomObject: vi.fn().mockResolvedValue({ body: {} }),
      createClusterCustomObject: vi.fn().mockResolvedValue({ body: {} }),
      replaceClusterCustomObject: vi.fn().mockResolvedValue({ body: {} }),
      deleteClusterCustomObject: vi.fn().mockResolvedValue({}),
    })
    getCurrentCluster = vi.fn().mockReturnValue({ server: 'http://localhost:8001' })
    getDefaultHeaders = vi.fn().mockReturnValue({})
  }

  return {
    KubeConfig: MockKubeConfig,
    CustomObjectsApi: vi.fn(),
  }
})

// Mock fetch for watch and schema
global.fetch = vi.fn().mockResolvedValue({
  ok: true,
  body: {
    getReader: vi.fn().mockReturnValue({
      read: vi.fn().mockResolvedValue({ done: true }),
    }),
  },
  json: vi.fn().mockResolvedValue({
    spec: {
      versions: [
        {
          name: 'v1',
          schema: {
            openAPIV3Schema: { properties: {} },
          },
        },
      ],
    },
  }),
}) as any

describe('useK8sCRD', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should expose CRUD methods', () => {
    const crd = useK8sCRD('biz.usmp.io', 'v1', 'businessvlans')
    expect(crd.list).toBeInstanceOf(Function)
    expect(crd.get).toBeInstanceOf(Function)
    expect(crd.create).toBeInstanceOf(Function)
    expect(crd.update).toBeInstanceOf(Function)
    expect(crd.remove).toBeInstanceOf(Function)
    expect(crd.getSchema).toBeInstanceOf(Function)
    expect(crd.startWatch).toBeInstanceOf(Function)
    expect(crd.stopWatch).toBeInstanceOf(Function)
  })

  it('should have reactive items array', () => {
    const crd = useK8sCRD('biz.usmp.io', 'v1', 'businessvlans')
    expect(crd.items.value).toEqual([])
  })

  it('should have loading and error states', () => {
    const crd = useK8sCRD('biz.usmp.io', 'v1', 'businessvlans')
    expect(typeof crd.loading.value).toBe('boolean')
    expect(crd.error.value).toBeNull()
  })

  it('should accept namespace option', () => {
    const crd = useK8sCRD('biz.usmp.io', 'v1', 'businessvlans', { namespace: 'default' })
    expect(crd).toBeDefined()
  })

  it('should accept autoWatch and autoList options', () => {
    const crd = useK8sCRD('biz.usmp.io', 'v1', 'businessvlans', {
      autoWatch: false,
      autoList: false,
    })
    expect(crd).toBeDefined()
  })

  it('should getSchema fetch CRD schema', async () => {
    const crd = useK8sCRD('biz.usmp.io', 'v1', 'businessvlans', { autoWatch: false, autoList: false })
    const schema = await crd.getSchema()
    expect(schema).toBeDefined()
    expect(fetch).toHaveBeenCalled()
  })
})
