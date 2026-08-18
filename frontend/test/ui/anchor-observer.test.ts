import { describe, it, expect, afterEach } from 'vitest'
import { installAnchorAttrObserver } from '../../src/ui/bridge'

// FA-05 锚点回填观察器 F1（组 7）：dt- 前缀 id → data-test 属性回填——
// 存量节点立即回填、新增节点经 MutationObserver 回填、断开后停止。
describe('installAnchorAttrObserver（FA-05 锚点回填）', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('存量 dt- 节点立即回填 data-test（含深层）', () => {
    document.body.innerHTML = '<div id="dt-a"><span id="dt-b"></span></div><div id="other"></div>'
    const stop = installAnchorAttrObserver()
    expect(document.getElementById('dt-a')?.getAttribute('data-test')).toBe('a')
    expect(document.getElementById('dt-b')?.getAttribute('data-test')).toBe('b')
    expect(document.getElementById('other')?.hasAttribute('data-test')).toBe(false)
    stop()
  })

  it('新增节点异步回填；断开后不再回填', async () => {
    const stop = installAnchorAttrObserver()
    const el = document.createElement('div')
    el.id = 'dt-late'
    document.body.appendChild(el)
    await new Promise((r) => setTimeout(r, 20))
    expect(el.getAttribute('data-test')).toBe('late')
    stop()
    const el2 = document.createElement('div')
    el2.id = 'dt-after-stop'
    document.body.appendChild(el2)
    await new Promise((r) => setTimeout(r, 20))
    expect(el2.hasAttribute('data-test')).toBe(false)
  })

  it('已有 data-test 的节点不覆盖', () => {
    document.body.innerHTML = '<div id="dt-x" data-test="custom"></div>'
    const stop = installAnchorAttrObserver()
    expect(document.getElementById('dt-x')?.getAttribute('data-test')).toBe('custom')
    stop()
  })
})
