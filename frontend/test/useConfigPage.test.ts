import { describe, it, expect, vi, afterEach } from 'vitest'
import { useConfigPage, useNativeModules } from '../src/composables/useConfigPage'

afterEach(() => {
  vi.restoreAllMocks()
})

function mockFetchOnce(body: any, ok = true) {
  global.fetch = vi.fn().mockResolvedValue({
    ok,
    status: ok ? 200 : 500,
    json: async () => body,
  }) as any
}

describe('useConfigPage native dynamic YANG schema', () => {
  it('unwraps the {data} envelope and loads dynamic fields from yang-api', async () => {
    const fields = [
      { path: '/vlan/vlans/vlan/id', type: 'number', label: 'id', group: 'vlan' },
      { path: '/vlan/vlans/vlan/admin-status', type: 'enum', label: 'admin-status' },
    ]
    mockFetchOnce({ code: 0, success: true, message: 'ok', data: { title: 'VLAN', fields } })

    // 'system' is a native module (not in BUSINESS_CRDS) → YANG schema path.
    const page = useConfigPage('system')
    const got = await page.getSchema()

    expect(got).toHaveLength(2)
    expect(got[0].type).toBe('number')
    expect(got[1].type).toBe('enum')
    expect(page.title.value).toBe('VLAN')
    expect(page.configType).toBe('native')
  })

  it('tolerates an already-unwrapped schema body', async () => {
    mockFetchOnce({ title: 'IFM', fields: [{ path: '/ifm/x', type: 'string', label: 'x' }] })
    const page = useConfigPage('system')
    const got = await page.getSchema()
    expect(got).toHaveLength(1)
    expect(page.title.value).toBe('IFM')
  })
})

describe('useNativeModules', () => {
  it('unwraps the envelope data array of modules', async () => {
    const mods = [
      { name: 'ifm', title: 'IFM', vendor: 'huawei' },
      { name: 'interfaces', title: 'Interfaces', vendor: 'openconfig' },
    ]
    mockFetchOnce({ code: 0, success: true, data: mods })

    const nm = useNativeModules()
    await nm.loadModules()

    expect(nm.modules.value).toHaveLength(2)
    expect(nm.modules.value[0].vendor).toBe('huawei')
    // grouped by vendor
    expect(nm.groupedByVendor.value.get('openconfig')?.length).toBe(1)
  })
})
