import { Page, expect, Locator } from '@playwright/test'

export class InterfacesPage {
  readonly page: Page
  readonly toolbar: Locator
  readonly refreshButton: Locator
  readonly submitButton: Locator

  constructor(page: Page) {
    this.page = page
    this.toolbar = page.locator('.toolbar')
    this.refreshButton = this.toolbar.getByRole('button', { name: '刷新' })
    this.submitButton = this.toolbar.getByRole('button', { name: '下发配置' })
  }

  async goto() {
    await this.page.goto('/')
    // 等待设备数据加载
    await this.page.waitForTimeout(3000)

    // 点击 Interfaces 菜单项
    const interfacesMenuItem = this.page.locator('.el-tree-node__content').filter({ hasText: 'Interfaces' })
    const count = await interfacesMenuItem.count()
    if (count > 0) {
      await interfacesMenuItem.first().click()
      await this.page.waitForTimeout(2000)
    }
  }

  async clickRefresh() {
    await this.refreshButton.click()
    await this.page.waitForTimeout(1000)
  }

  async clickSubmit() {
    await this.submitButton.click()
    await this.page.waitForTimeout(1000)
  }

  async verifyTitleDisplayed() {
    await expect(this.page.getByText('接口配置管理')).toBeVisible()
  }

  async verifyButtonsEnabled() {
    await expect(this.refreshButton).toBeEnabled()
    await expect(this.submitButton).toBeEnabled()
  }
}
