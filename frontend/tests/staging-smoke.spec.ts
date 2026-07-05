import { test, expect } from '@playwright/test'

// 部署冒烟 —— e2e-staging 工作流的浏览器门禁（v1）。
//
// 目的：用真实浏览器验证「已部署的前端容器」能被访问、Vue 应用能挂载、外壳导航能渲染，
//       且无致命控制台错误。这是对整套部署的端到端浏览器级验证（browser → nginx 容器 → SPA）。
//
// 为何不用现有 navigation/vlan/interfaces/e2e-demo 规格：它们断言的是当前后端/前端未实现的
// 接口契约与设计稿文案（如 <title> 里的“交换机设备管理平台”、data.data.vlans 数组、
// 设备树里的 192.168.1.1 表格数据），与 CRD 驱动的真实应用脱节，需应用级改造，另立 OpenSpec change。
// 这里只断言「真实可稳定通过」的东西，保证门禁诚实为绿。

test.describe('部署冒烟 - 前端 SPA', () => {
  test('SPA 应被服务且成功挂载', async ({ page }) => {
    const consoleErrors: string[] = []
    page.on('console', (m) => {
      if (m.type() === 'error') consoleErrors.push(m.text())
    })

    await page.goto('/', { waitUntil: 'networkidle' })

    // 页面标题存在（静态 HTML 已服务）
    expect(await page.title()).toBeTruthy()

    // #app 已渲染出内容（Vue 应用挂载成功，而非空壳）
    const appHtml = await page.locator('#app').innerHTML()
    expect(appHtml.length).toBeGreaterThan(50)

    // 无致命控制台错误
    expect(consoleErrors, `console errors:\n${consoleErrors.join('\n')}`).toHaveLength(0)
  })

  test('应用外壳导航应渲染', async ({ page }) => {
    await page.goto('/', { waitUntil: 'networkidle' })

    // 侧边栏真实导航项可见（证明应用外壳完整渲染）
    await expect(page.getByText('设备管理', { exact: false }).first()).toBeVisible()
    await expect(page.getByText('概览', { exact: false }).first()).toBeVisible()
    await expect(page.getByText('系统设置', { exact: false }).first()).toBeVisible()
  })
})
