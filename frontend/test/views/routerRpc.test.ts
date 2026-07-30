import { describe, it, expect } from 'vitest'
import router from '../../src/router'

// FE-19 rpc 直达路由：/module/:module/rpc/:rpcName 复用模块控制台页。
describe('router · rpc 直达路由（FE-19）', () => {
  it('/module/ifm/rpc/restart-if 解析到 module-rpc 且参数齐全', () => {
    const r = router.resolve('/module/ifm/rpc/restart-if')
    expect(r.name).toBe('module-rpc')
    expect(r.params).toEqual({ module: 'ifm', rpcName: 'restart-if' })
  })

  it('/module/ifm 仍解析到 module-console（既有路由不受影响）', () => {
    const r = router.resolve('/module/ifm')
    expect(r.name).toBe('module-console')
    expect(r.params).toEqual({ module: 'ifm' })
  })
})
