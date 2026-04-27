import { test, expect } from '@playwright/test'
import { InterfacesPage } from './pages/InterfacesPage'

test.describe('Interfaces 接口配置管理 - 端到端测试', () => {
  let interfacesPage: InterfacesPage

  test.beforeEach(async ({ page }) => {
    interfacesPage = new InterfacesPage(page)
    await interfacesPage.goto()
  })

  test('页面加载 - 应显示接口配置管理标题', async () => {
    await interfacesPage.verifyTitleDisplayed()
  })

  test('按钮操作 - 刷新按钮和下发配置按钮应可用', async () => {
    await interfacesPage.verifyButtonsEnabled()
  })

  test('刷新操作 - 点击刷新按钮不应报错', async () => {
    await interfacesPage.clickRefresh()
    await interfacesPage.verifyTitleDisplayed()
  })

  test('页面元素 - 工具栏应包含两个按钮', async ({ page }) => {
    const buttons = page.locator('.toolbar button')
    expect(await buttons.count()).toBeGreaterThanOrEqual(2)
  })
})
