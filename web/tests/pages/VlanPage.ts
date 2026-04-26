import { Page, expect, Locator } from '@playwright/test'

export class VlanPage {
  readonly page: Page
  readonly addVlanButton: Locator
  readonly refreshButton: Locator
  readonly vlanTable: Locator

  constructor(page: Page) {
    this.page = page
    this.addVlanButton = page.getByRole('button', { name: '新建 VLAN' })
    this.refreshButton = page.getByRole('button', { name: '刷新' })
    this.vlanTable = page.locator('.el-table')
  }

  async goto() {
    await this.page.goto('/')
    // 等待设备数据加载
    await this.page.waitForTimeout(3000)

    // 点击设备节点展开
    const deviceNode = this.page.locator('.el-tree-node__content').filter({ hasText: '192.168.1.1' })
    if (await deviceNode.count() > 0) {
      await deviceNode.first().click()
      await this.page.waitForTimeout(500)
    }

    // 点击 VLANs 菜单项
    const vlanMenuItem = this.page.locator('.el-tree-node__content').filter({ hasText: 'VLANs' })
    if (await vlanMenuItem.count() > 0) {
      await vlanMenuItem.first().click()
      await this.page.waitForTimeout(1000)
    }
  }

  async clickAddVlan() {
    await this.addVlanButton.click()
    await this.page.waitForTimeout(500)
  }

  async clickRefresh() {
    await this.refreshButton.click()
    await this.page.waitForTimeout(1000)
  }

  async verifyVlanExists(vlanId: number) {
    const row = this.page.locator('.el-table__row').filter({ hasText: String(vlanId) })
    await expect(row).toBeVisible()
  }

  async verifyStatusDisplayed() {
    // 检查状态标签
    const hasUP = this.page.getByText('启用')
    const hasActive = this.page.getByText('运行中')
    await expect(hasUP.or(hasActive).first()).toBeVisible()
  }

  async waitForTableLoaded() {
    await this.page.waitForSelector('.el-table__row', { state: 'attached', timeout: 10000 })
  }
}
