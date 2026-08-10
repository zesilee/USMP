import { describe, it, expect } from 'vitest'
import router from '../src/router'

describe('Router Configuration', () => {
  it('should have dashboard route', () => {
    const route = router.getRoutes().find(r => r.name === 'dashboard')
    expect(route).toBeDefined()
    expect(route?.path).toBe('/')
  })

  it('should have module console route', () => {
    const routes = router.getRoutes()
    const console_ = routes.find(r => r.name === 'module-console')
    expect(console_?.path).toBe('/module/:module')
  })

  // FE-13：legacy 路由一律不存在——CRD 死路（/native/:module、/config/route）
  // 与旧配置页重定向（/config/interface、/config/vlan，兼容期已结束）均已退役。
  it('legacy 路由不存在（/native/:module、/config/*）', () => {
    const routes = router.getRoutes()
    expect(routes.find(r => r.name === 'native')).toBeUndefined()
    expect(routes.find(r => r.path === '/config/route')).toBeUndefined()
    expect(routes.map(r => r.name)).not.toContain('route')
    expect(routes.find(r => r.path === '/config/interface')).toBeUndefined()
    expect(routes.find(r => r.path === '/config/vlan')).toBeUndefined()
  })

  it('should have logs and settings routes', () => {
    const names = router.getRoutes().map(r => r.name)
    expect(names).toContain('logs')
    expect(names).toContain('settings')
  })
})
