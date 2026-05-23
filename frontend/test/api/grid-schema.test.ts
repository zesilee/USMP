import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as apiModule from '../../src/api'

describe('grid-schema API', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.spyOn(apiModule.default, 'get').mockResolvedValue({})
    vi.spyOn(apiModule.default, 'post').mockResolvedValue({})
  })

  it('getInterfaceGridSchema calls correct API endpoint', async () => {
    const ip = '192.168.1.1'
    await apiModule.getInterfaceGridSchema(ip)
    expect(apiModule.default.get).toHaveBeenCalledWith(`/ui-schema/devices/${ip}/interfaces`)
  })

  it('applyInterfaceGridConfig calls correct API endpoint with payload', async () => {
    const ip = '192.168.1.1'
    const payload = { schemaVersion: '1', values: {} }
    await apiModule.applyInterfaceGridConfig(ip, payload)
    expect(apiModule.default.post).toHaveBeenCalledWith(`/ui-schema/devices/${ip}/interfaces/apply`, payload)
  })
})
