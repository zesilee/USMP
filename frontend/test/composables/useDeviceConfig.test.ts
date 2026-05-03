import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useDeviceConfig } from '../../src/composables/useDeviceConfig'
import * as crdApi from '../../src/api/crd'
import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'

vi.mock('../../src/api/crd', () => ({
  getSchema: vi.fn(),
  createConfig: vi.fn(),
  updateConfig: vi.fn(),
  deleteConfig: vi.fn(),
  watchConfigs: vi.fn(() => ({
    onmessage: null,
    onerror: null,
    close: vi.fn()
  }))
}))

function setupComposable() {
  let composableResult: ReturnType<typeof useDeviceConfig> | null = null
  const TestComponent = defineComponent({
    template: '<div></div>',
    setup() {
      composableResult = useDeviceConfig('device-1', 'interface')
      return {}
    }
  })
  const wrapper = mount(TestComponent)
  return { wrapper, composable: composableResult! }
}

describe('useDeviceConfig Composable', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(crdApi.getSchema).mockResolvedValue({
      module: 'interface',
      title: '接口配置',
      fields: [],
      listFields: []
    })
  })

  it('should expose all required reactive states', () => {
    const { composable } = setupComposable()
    expect(composable.configCR).toBeDefined()
    expect(composable.schema).toBeDefined()
    expect(composable.isLoading).toBeDefined()
    expect(composable.isSyncing).toBeDefined()
    expect(composable.error).toBeDefined()
  })

  it('should expose all methods', () => {
    const { composable } = setupComposable()
    expect(typeof composable.save).toBe('function')
    expect(typeof composable.remove).toBe('function')
    expect(typeof composable.refresh).toBe('function')
  })

  it('should have correct isSyncing computed value', () => {
    const { composable } = setupComposable()
    expect(composable.isSyncing.value).toBe(false)
  })

  it('should call getSchema on initialization', async () => {
    setupComposable()
    await new Promise(resolve => setTimeout(resolve, 10))
    expect(crdApi.getSchema).toHaveBeenCalledWith('interface')
  })
})
