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

  // ===== 通用模块控制台（generic-module-console，FE-10~13）=====
  // 旧 /config/vlan、/config/interface 重定向到 /module/:module；页面 Tab/列/表单
  // 全部由 schema 派生。以下把原「表单动态渲染/when 显隐/校验拦截/SPA 切换」回归
  // 断言迁移到控制台，并新增「种子行/高级搜索」断言。

  // 选设备（页头首个 el-select）。
  async function pickDevice(page: import('@playwright/test').Page) {
    await page.locator('.el-select').first().click()
    await page.locator('.el-select-dropdown__item', { hasText: '192.168.1.1' }).first().click()
  }

  test('VLAN 旧路由重定向到控制台，新增表单动态渲染出 YANG 字段', async ({ page }) => {
    await page.goto('/config/vlan', { waitUntil: 'networkidle' })
    await expect(page).toHaveURL(/module\/vlan/)

    await pickDevice(page)
    await page.getByRole('tab', { name: 'vlans', exact: true }).click()
    await page.getByRole('button', { name: '新增' }).first().click()

    // 抽屉里出现 schema 驱动的字段（admin-status 为 YANG 叶子名，动态渲染才会有）
    await expect(page.getByText('admin-status', { exact: false }).first()).toBeVisible({ timeout: 15000 })
  })

  // 空表单提交应被前端校验拦截（§9）：缺主键 id 时「下发并对账」禁用。
  test('VLAN 表单缺主键(id)时下发应被校验拦截', async ({ page }) => {
    await page.goto('/module/vlan', { waitUntil: 'networkidle' })
    await pickDevice(page)
    await page.getByRole('tab', { name: 'vlans', exact: true }).click()
    await page.getByRole('button', { name: '新增' }).first().click()
    await expect(page.getByText('admin-status', { exact: false }).first()).toBeVisible({ timeout: 15000 })

    await expect(page.getByRole('button', { name: /下发并对账/ })).toBeDisabled()
  })

  // 接口（华为 IFM）：Tab 由模块根派生，interfaces 列表 Tab 内新增表单动态渲染。
  test('接口控制台 Tab 派生 + 新增表单动态渲染出 YANG 字段', async ({ page }) => {
    await page.goto('/config/interface', { waitUntil: 'networkidle' })
    await expect(page).toHaveURL(/module\/ifm/)

    await pickDevice(page)
    await page.getByRole('tab', { name: 'interfaces', exact: true }).click()
    await page.getByRole('button', { name: '新增' }).first().click()

    // mtu 为 IFM 叶子名，schema 动态渲染才会出现
    await expect(page.getByText('mtu', { exact: false }).first()).toBeVisible({ timeout: 15000 })
  })

  // 种子数据（模拟网元 DemoSeedConfig）：5 条接口回读进表格，sub 行显示 parent-name。
  test('接口列表应展示模拟网元种子行（3 main + 2 sub）', async ({ page }) => {
    await page.goto('/module/ifm', { waitUntil: 'networkidle' })
    await pickDevice(page)
    await page.getByRole('tab', { name: 'interfaces', exact: true }).click()

    await expect(page.getByText('200GE0/1/0', { exact: true }).first()).toBeVisible({ timeout: 20000 })
    await expect(page.getByText('200GE0/1/1.1', { exact: false }).first()).toBeVisible()
  })

  // 高级搜索（ext:support-filter 驱动）：class=sub-interface 过滤后主接口行消失。
  test('高级搜索按 class 过滤（support-filter 驱动）', async ({ page }) => {
    await page.goto('/module/ifm', { waitUntil: 'networkidle' })
    await pickDevice(page)
    await page.getByRole('tab', { name: 'interfaces', exact: true }).click()
    await expect(page.getByText('200GE0/1/0', { exact: true }).first()).toBeVisible({ timeout: 20000 })

    await page.getByRole('button', { name: /高级搜索/ }).click()
    const panel = page.locator('.search-panel')
    await panel.locator('.el-select').first().click()
    await page.locator('.el-select-dropdown__item:visible', { hasText: 'sub-interface' }).first().click()
    await panel.getByRole('button', { name: '查询' }).click()

    // 主接口行被过滤掉，仅剩 2 条 sub-interface
    await expect(page.getByText('200GE0/1/2', { exact: false })).toHaveCount(0)
    await expect(page.getByText('200GE0/1/0.1', { exact: false }).first()).toBeVisible()
  })

  // 接口 when 约束（FE-07）：parent-name 由 YANG `when "../class='sub-interface'"` 门控。
  // 断言限定在 .el-drawer 内（页面其他区域可能出现同名文本）。
  test('接口 when 约束：class=sub-interface 才显现 parent-name（数据驱动显隐）', async ({ page }) => {
    await page.goto('/module/ifm', { waitUntil: 'networkidle' })
    await pickDevice(page)
    await page.getByRole('tab', { name: 'interfaces', exact: true }).click()
    await page.getByRole('button', { name: '新增' }).first().click()

    const drawer = page.locator('.el-drawer')
    await expect(drawer.getByText('class', { exact: false }).first()).toBeVisible({ timeout: 15000 })
    await expect(drawer.locator('.el-form-item__label', { hasText: 'parent-name' })).toHaveCount(0)

    // 精确定位 class 字段的下拉（抽屉首个 el-select 是字母序在前的 admin-status），
    // 并只点“可见”的下拉项（teleport 的历史下拉会残留在 DOM 中）。
    const classItem = drawer.locator('.el-form-item', {
      has: page.locator('.el-form-item__label', { hasText: /^class$/ }),
    })
    await classItem.locator('.el-select').click()
    await page.locator('.el-select-dropdown__item:visible', { hasText: 'sub-interface' }).first().click()

    await expect(drawer.getByText('parent-name', { exact: false }).first()).toBeVisible({ timeout: 15000 })
  })

  // SPA 内从 VLAN 模块切到 IFM 模块应重载 schema（回归门禁：路由参数变化 → schema 重载）。
  test('SPA 内从 VLAN 切换到接口模块应加载接口模型（非沿用 VLAN）', async ({ page }) => {
    await page.goto('/module/vlan', { waitUntil: 'networkidle' })
    await expect(page.getByRole('tab', { name: 'vlans', exact: true })).toBeVisible({ timeout: 15000 })

    // 侧栏业务菜单（/yang/modules 驱动）内点 ifm 模块 —— SPA 内导航
    await page.locator('[data-test="module-item-ifm"]').click()
    await expect(page).toHaveURL(/module\/ifm/)

    await pickDevice(page)
    await page.getByRole('tab', { name: 'interfaces', exact: true }).click()
    await page.getByRole('button', { name: '新增' }).first().click()

    // 接口独有字段 mtu 应出现（若仍沿用 VLAN schema 则不会有）
    await expect(page.getByText('mtu', { exact: false }).first()).toBeVisible({ timeout: 15000 })
  })
})
