import { describe, it, expect } from 'vitest'
import router from '../src/router'

describe('Router Configuration', () => {
  it('should have dashboard route', () => {
    const route = router.getRoutes().find(r => r.name === 'dashboard')
    expect(route).toBeDefined()
    expect(route?.path).toBe('/')
  })

  it('should have all business config routes', () => {
    const names = router.getRoutes().map(r => r.name)
    expect(names).toContain('interface')
    expect(names).toContain('vlan')
    expect(names).toContain('route')
  })

  it('should have native config dynamic route', () => {
    const route = router.getRoutes().find(r => r.name === 'native')
    expect(route).toBeDefined()
    expect(route?.path).toBe('/native/:module')
  })

  it('should have logs and settings routes', () => {
    const names = router.getRoutes().map(r => r.name)
    expect(names).toContain('logs')
    expect(names).toContain('settings')
  })
})
