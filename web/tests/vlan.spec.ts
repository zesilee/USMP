import { test, expect } from '@playwright/test'

test.describe('VLAN 管理 - 端到端测试', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    await page.waitForTimeout(1000)
    // 点击 VLANs 节点
    await page.getByText('VLANs').first().click()
    await page.waitForTimeout(1000)
  })

  test('页面加载 - 应显示 VLAN 管理标题', async ({ page }) => {
    await expect(page.getByText('VLAN 配置管理')).toBeVisible()
  })

  test('VLAN 表格 - 应显示默认 VLAN 数据', async ({ page }) => {
    // 检查表格中是否有默认 VLAN
    await expect(page.getByText('default').first()).toBeVisible()
  })

  test('VLAN 表格 - 应显示 Management VLAN', async ({ page }) => {
    await expect(page.getByText('Management').first()).toBeVisible()
  })

  test('按钮操作 - 应显示刷新按钮', async ({ page }) => {
    await expect(page.getByRole('button', { name: '刷新' }).first()).toBeEnabled()
  })

  test('按钮操作 - 应显示新建 VLAN 按钮', async ({ page }) => {
    await expect(page.getByRole('button', { name: '新建 VLAN' }).first()).toBeEnabled()
  })

  test('新建 VLAN - 点击新建应打开编辑弹窗', async ({ page }) => {
    const addButton = page.getByRole('button', { name: '新建 VLAN' }).first()
    await addButton.click()
    await page.waitForTimeout(500)

    // 检查弹窗中的按钮
    const hasCancel = page.getByText('取消').isVisible()
    const hasSave = page.getByText('保存').isVisible()
    expect(await hasCancel || await hasSave).toBe(true)
  })

  test('表格列 - 应显示正确的列名', async ({ page }) => {
    await expect(page.getByText('VLAN ID (1-4094)').first()).toBeVisible()
    await expect(page.getByText('VLAN 名称').first()).toBeVisible()
    await expect(page.getByText('管理状态').first()).toBeVisible()
  })
})
