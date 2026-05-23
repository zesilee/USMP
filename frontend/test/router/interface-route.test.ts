import { describe, expect, it } from 'vitest'
import router from '../../src/router'

describe('interface route', () => {
  it('passes device query as InterfaceGridPage deviceIp prop', () => {
    const route = router.resolve('/config/interface?device=192.168.1.2')
    const props = route.matched[0].props.default

    expect(typeof props).toBe('function')
    expect((props as (route: typeof route) => unknown)(route)).toEqual({
      deviceIp: '192.168.1.2'
    })
  })
})
