import { describe, it, expect } from 'vitest'
import router from '../src/router'

describe('Router Configuration', () => {
  it('should have dashboard route', () => {
    const route = router.getRoutes().find(r => r.name === 'dashboard')
    expect(route).toBeDefined()
    expect(route?.path).toBe('/')
  })

  // FE-13：业务配置迁移到通用模块控制台，旧路由重定向保留书签可达。
  it('should have module console route and legacy redirects', () => {
    const routes = router.getRoutes()
    const console_ = routes.find(r => r.name === 'module-console')
    expect(console_?.path).toBe('/module/:module')

    expect(routes.find(r => r.path === '/config/interface')?.redirect).toBe('/module/ifm')
    expect(routes.find(r => r.path === '/config/vlan')?.redirect).toBe('/module/vlan')
    expect(routes.map(r => r.name)).toContain('route')
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
