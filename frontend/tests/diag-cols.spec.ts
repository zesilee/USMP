import { test } from '@playwright/test'

// 内网侦察（列丢失定案）：VLAN 列表勾选 9 列仅渲染 3 列——dump eview 表头/
// 首行真实 DOM（单元格数、文本、宽度、类名）+ 滚动几何，判定列是「整列被丢」
// 还是「列在但标题空白」还是「宽度溢出被裁」。
const DEVICE_IP = process.env.E2E_DEVICE_IP || '192.168.1.1'

test('C1 VLAN列表列 DOM 定案', async ({ page }) => {
  await page.goto('/module/vlan', { waitUntil: 'networkidle' })
  await page.locator('.ev_inputSelect').first().click()
  await page.locator('.ev_popup_option', { hasText: DEVICE_IP }).first().click()
  await page.locator('.ev_tab_title', { hasText: /^VLAN列表$/ }).first().click()
  await page.waitForTimeout(3000)
  const info = await page.evaluate(() => {
    const root = document.querySelector('[class*="ev_table"]')?.closest('div[id]') ?? document
    const ths = Array.from(document.querySelectorAll('th'))
    const headCells = ths.map((th) => ({
      t: (th as HTMLElement).innerText.replace(/\s+/g, ' ').slice(0, 16),
      w: Math.round(th.getBoundingClientRect().width),
      cls: th.className.slice(0, 40),
    }))
    const firstRow = document.querySelector('tbody tr')
    const rowCells = Array.from(firstRow?.querySelectorAll('td') ?? []).map((td) => ({
      t: (td as HTMLElement).innerText.slice(0, 12),
      w: Math.round(td.getBoundingClientRect().width),
    }))
    const scrollers = Array.from(document.querySelectorAll('[class*="ev_table"]'))
      .filter((e) => e.scrollWidth > e.clientWidth + 5)
      .slice(0, 3)
      .map((e) => `${e.className.split(' ')[0]} sw=${e.scrollWidth} cw=${e.clientWidth}`)
    const funnels = document.querySelectorAll('.ub-col-filter').length
    const tables = Array.from(document.querySelectorAll('table')).map(
      (t) => `${t.className.split(' ')[0] || 'noclass'} w=${Math.round(t.getBoundingClientRect().width)} cols=${t.querySelectorAll('col').length}`,
    )
    void root
    // C2：操作列表头 sticky 失效定案——最后一个 th 的计算样式 + 祖先链
    // （overflow/transform/margin 同步机制判定）。
    const lastTh = ths[ths.length - 1]
    const cs = lastTh ? getComputedStyle(lastTh) : null
    const anc: string[] = []
    let n: Element | null = lastTh ?? null
    for (let i = 0; n && i < 8; i++) {
      const c = getComputedStyle(n)
      anc.push(
        `${n.tagName}.${n.className.toString().split(' ')[0]}|of=${c.overflow}/${c.overflowX}|tf=${c.transform !== 'none' ? 'Y' : 'n'}|pos=${c.position}|ml=${c.marginLeft}|left=${c.left}`,
      )
      n = n.parentElement
    }
    const rootHasClass = !!document.querySelector('.ub-fixed-right-last')
    const thInMarked = !!lastTh?.closest('.ub-fixed-right-last')
    return {
      thCount: ths.length, headCells, rowCellCount: rowCells.length, rowCells, scrollers, funnels, tables,
      c2: { thPos: cs?.position, thRight: cs?.right, rootHasClass, thInMarked, anc },
    }
  })
  console.log('C1=' + JSON.stringify(info))
})
