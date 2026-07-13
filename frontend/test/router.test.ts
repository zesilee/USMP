import { describe, it, expect } from 'vitest'
import router from '../src/router'

describe('Router Configuration', () => {
  it('should have dashboard route', () => {
    const route = router.getRoutes().find(r => r.name === 'dashboard')
    expect(route).toBeDefined()
    expect(route?.path).toBe('/')
  })

  // FE-13：原生配置迁移到通用模块控制台，旧可用路由重定向保留书签可达。
  it('should have module console route and legacy redirects', () => {
    const routes = router.getRoutes()
    const console_ = routes.find(r => r.name === 'module-console')
    expect(console_?.path).toBe('/module/:module')

    expect(routes.find(r => r.path === '/config/interface')?.redirect).toBe('/module/ifm')
    expect(routes.find(r => r.path === '/config/vlan')?.redirect).toBe('/module/vlan')
  })

  // FE-13：Stack A CRD 死路已退役——生产中从未可用，直接移除无重定向义务
  it('legacy CRD 死路路由不存在（/native/:module、/config/route）', () => {
    const routes = router.getRoutes()
    expect(routes.find(r => r.name === 'native')).toBeUndefined()
    expect(routes.find(r => r.path === '/config/route')).toBeUndefined()
    expect(routes.map(r => r.name)).not.toContain('route')
  })

  it('should have logs and settings routes', () => {
    const names = router.getRoutes().map(r => r.name)
    expect(names).toContain('logs')
    expect(names).toContain('settings')
  })
})
