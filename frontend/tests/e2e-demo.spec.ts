import { test, expect } from '@playwright/test'

// E2E 测试演示 - 验证测试环境正常工作
test.describe('E2E 环境验证', () => {
  test('前端页面应该可以正常加载', async ({ page }) => {
    await page.goto('/')
    await page.waitForTimeout(2000)

    // 验证页面标题存在
    const title = await page.title()
    expect(title).toBeTruthy()

    // 检查页面没有 404 或错误
    const pageContent = await page.content()
    expect(pageContent).not.toContain('404')
    expect(pageContent).not.toContain('Error')
  })

  test('后端 API 应该可以正常访问', async ({ request }) => {
    const response = await request.get('http://localhost:8080/api/v1/devices')
    expect(response.ok()).toBe(true)

    const data = await response.json()
    expect(data.success).toBe(true)
    expect(data.data).toBeDefined()
    expect(Array.isArray(data.data)).toBe(true)
  })

  test('VLAN 配置 API 应该返回数据', async ({ request }) => {
    const response = await request.get('http://localhost:8080/api/v1/config/192.168.1.1/vlans')
    expect(response.ok()).toBe(true)

    const data = await response.json()
    expect(data.success).toBe(true)
    expect(data.data.vlans).toBeDefined()
    expect(Array.isArray(data.data.vlans)).toBe(true)
    expect(data.data.vlans.length).toBeGreaterThanOrEqual(4)
  })
})
