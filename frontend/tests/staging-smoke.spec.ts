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

  // 设备管理页应渲染出后端种子设备（回归门禁）。
  //
  // 此断言此前被排除，原因是 stores/device.ts 对接的是一个虚构后端契约
  // （GET /api/devices + res.data.devices），设备永远拉不到、表格恒空。store 修复后
  // 改用真实契约（GET /api/v1/devices + res.data.data.devices，兼容 online/status），
  // 后端种子设备 192.168.1.1 现在能真实渲染 —— 故此断言现在诚实为真，用作该 BUG 的回归防线。
  test('设备管理页应列出种子设备 192.168.1.1', async ({ page }) => {
    await page.goto('/devices', { waitUntil: 'networkidle' })

    // 设备表格里出现种子设备 IP（证明 store→/api/v1/devices→表格 整条链路打通）
    await expect(page.getByText('192.168.1.1', { exact: false }).first()).toBeVisible({ timeout: 15000 })
  })

  // VLAN 新增表单应由 YANG schema 动态渲染出字段（回归门禁）。
  //
  // 此前两次故障：① VLAN 走 Stack A K8s CRD 死路；② schema 用裸相对 fetch 被 nginx 拦成 index.html
  // → 表单恒空。修复后 schema 走 api 客户端、表单由 /yang/schema/vlan?form=nested 动态渲染。
  // 本断言进 /config/vlan → 选设备 → 新增 → 校验 schema 字段真实渲染，兜住「表单空」回归。
  test('VLAN 新增表单应动态渲染出 YANG 字段', async ({ page }) => {
    await page.goto('/config/vlan', { waitUntil: 'networkidle' })
    await expect(page.getByText('VLAN 配置', { exact: false }).first()).toBeVisible()

    // 选择种子设备（el-select 下拉项）
    await page.locator('.el-select').first().click()
    await page.locator('.el-select-dropdown__item', { hasText: '192.168.1.1' }).first().click()

    // 打开新增抽屉
    await page.getByRole('button', { name: /新增 VLAN/ }).click()

    // 抽屉里出现 schema 驱动的字段（admin-status 为 YANG 叶子名，动态渲染才会有）
    await expect(page.getByText('admin-status', { exact: false }).first()).toBeVisible({ timeout: 15000 })
  })

  // 空表单提交应被前端校验拦截（§9：不提交、行内提示），不静默下发非法配置。
  test('VLAN 表单缺主键(id)时下发应被校验拦截', async ({ page }) => {
    await page.goto('/config/vlan', { waitUntil: 'networkidle' })
    await page.locator('.el-select').first().click()
    await page.locator('.el-select-dropdown__item', { hasText: '192.168.1.1' }).first().click()
    await page.getByRole('button', { name: /新增 VLAN/ }).click()
    await expect(page.getByText('admin-status', { exact: false }).first()).toBeVisible({ timeout: 15000 })

    // 缺主键 id（及其它必填项）为空时，「下发并对账」按钮禁用 —— §9 的拦截在当前设计里以
    // 「不可提交」实现，比行内提示更强：不完整/非法配置根本无法下发。
    // （旧断言点击该按钮并等「必填」文案，但当前按钮 disabled-until-valid，点击恒 30s 超时。）
    await expect(page.getByRole('button', { name: /下发并对账/ })).toBeDisabled()
  })

  // 接口（华为 IFM）新增表单应由 YANG schema 动态渲染（与 VLAN 共用通用配置流 DeviceConfigPage）。
  test('接口新增表单应动态渲染出 YANG 字段', async ({ page }) => {
    await page.goto('/config/interface', { waitUntil: 'networkidle' })
    await expect(page.getByText('接口配置', { exact: false }).first()).toBeVisible()

    await page.locator('.el-select').first().click()
    await page.locator('.el-select-dropdown__item', { hasText: '192.168.1.1' }).first().click()
    await page.getByRole('button', { name: /新增接口/ }).click()

    // mtu 为 IFM 叶子名，schema 动态渲染才会出现
    await expect(page.getByText('mtu', { exact: false }).first()).toBeVisible({ timeout: 15000 })
  })

  // 应用内（SPA）从 VLAN 导航到接口应加载各自模型（回归门禁）。
  //
  // 此前 VLAN/接口共用 DeviceConfigPage，vue-router 复用实例 → 切换后 schema 不重载 →
  // 接口表单显示 VLAN 字段。之前的冒烟用 page.goto 全量重载各自页面，未走应用内导航故漏测。
  // 此断言点侧栏在 SPA 内从 VLAN 切到接口，校验加载的是接口(mtu)而非 VLAN 字段。
  test('SPA 内从 VLAN 切换到接口应加载接口模型（非沿用 VLAN）', async ({ page }) => {
    await page.goto('/config/vlan', { waitUntil: 'networkidle' })
    await expect(page.getByText('VLAN 配置', { exact: false }).first()).toBeVisible()

    // 侧栏点「接口配置」——SPA 内导航（非整页重载）
    await page.getByText('接口配置', { exact: false }).first().click()
    await expect(page).toHaveURL(/config\/interface/)

    await page.locator('.el-select').first().click()
    await page.locator('.el-select-dropdown__item', { hasText: '192.168.1.1' }).first().click()
    await page.getByRole('button', { name: /新增接口/ }).click()

    // 接口独有字段 mtu 应出现（若仍沿用 VLAN schema 则不会有）
    await expect(page.getByText('mtu', { exact: false }).first()).toBeVisible({ timeout: 15000 })
  })
})
