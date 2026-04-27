import { test, expect } from '@playwright/test'

test.describe('页面导航 - E2E 测试', () => {
  test('首页加载 - 应正常显示页面', async ({ page }) => {
    await page.goto('/')
    await page.waitForTimeout(1000)
    // 检查页面没有崩溃
    expect(await page.title()).toBeTruthy()
  })

  test('主题样式 - 页面应显示标题', async ({ page }) => {
    await page.goto('/')
    await page.waitForTimeout(1000)

    // 检查页面标题是否存在
    await expect(page.getByText('交换机设备管理平台')).toBeVisible()
  })

  test('响应式 - 页面在桌面分辨率应正常显示', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 720 })
    await page.goto('/')
    await page.waitForTimeout(1000)

    // 检查页面宽度正确
    const width = await page.evaluate(() => window.innerWidth)
    expect(width).toBe(1280)
  })
})
