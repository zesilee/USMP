import { test, expect } from '@playwright/test'

// 组 7 选择器口径（EviewUI 桥接线后）：结构类选择器集中于此——类名依据
// 内网校准报告实证 DOM（CAL-R16/F3-R10）。data-test 锚（FA-05）不受换库影响。
// 待内网首跑收集：下拉弹层选项、Tab 溢出下拉的真实形态（标注 TODO-E2E）。
// 种子设备地址按环境注入（组 7）：compose staging=192.168.1.1（默认）；
// kind 环境模拟网元注册为 K8s 服务名——E2E_DEVICE_IP=netconf-sim.default；
// 对接真机时传真机 IP。
const DEVICE_IP = process.env.E2E_DEVICE_IP || '192.168.1.1'

const SEL = {
  select: '.ev_inputSelect',
  // 内网 diag4 实证：弹层=div.ev_popup（data-test 随 anchorId 回填为
  // {select 锚}_pop），选项=span.ev_popup_option[role=option]。
  selectOption: '.ev_popup_option',
  formItem: '.form-item-shell',
  formItemLabel: '.fis-label',
  badgeCount: '.ev_badge_content',
  modalClose: '.ev_Dialog_closeIcon',
  tableRow: '.ev_table_content tr',
  tab: '.ev_tab_title',
} as const

// 部署冒烟 —— e2e-staging 工作流的浏览器门禁（v1）。
//
// 目的：用真实浏览器验证「已部署的前端容器」能被访问、Vue 应用能挂载、外壳导航能渲染，
//       且无致命控制台错误。这是对整套部署的端到端浏览器级验证（browser → nginx 容器 → SPA）。
//
// 为何不用现有 navigation/vlan/interfaces/e2e-demo 规格：它们断言的是当前后端/前端未实现的
// 接口契约与设计稿文案（如 <title> 里的“交换机设备管理平台”、data.data.vlans 数组、
// 设备树里的 192.168.1.1 表格数据），与 CRD 驱动的真实应用脱节，需应用级改造，另立 OpenSpec change。
// 这里只断言「真实可稳定通过」的东西，保证门禁诚实为绿。

// 选设备（页头设备下拉 data-test）：模块控制台/业务切换用例共用。
async function pickDevice(page: import('@playwright/test').Page) {
  await page.locator(SEL.select).first().click()
  await page.locator(SEL.selectOption, { hasText: DEVICE_IP }).first().click()
}

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
    await expect(page.getByText(DEVICE_IP, { exact: false }).first()).toBeVisible({ timeout: 15000 })
  })

  // ===== 通用模块控制台（generic-module-console，FE-10~13）=====
  // 页面 Tab/列/表单全部由 schema 派生（旧 /config/* 重定向已退役，直接访问现役路由）。
  // 以下把原「表单动态渲染/when 显隐/校验拦截/SPA 切换」回归断言迁移到控制台，
  // 并新增「种子行/高级搜索」断言。

  test('VLAN 控制台创建表单动态渲染出 YANG 字段', async ({ page }) => {
    await page.goto('/module/vlan', { waitUntil: 'networkidle' })
    await expect(page).toHaveURL(/module\/vlan/)

    await pickDevice(page)
    await page.locator(SEL.tab, { hasText: /^VLAN列表$/ }).first().click()
    await page.getByRole('button', { name: '创建' }).first().click()

    // 详情编辑区（FE-21 master-detail，抽屉退役）出现 schema 驱动的字段
    //（UI-03 后标签经 snd res 本地化）；限定 pane——列表列头同名且固定列
    // 副本隐藏，裸 first() 会命中隐藏 cell。
    const pane = page.locator('[data-test="item-detail-pane"]')
    await expect(pane.getByText('VLAN管理状态', { exact: false }).first()).toBeVisible({ timeout: 15000 })
  })

  // 空表单提交应被前端校验拦截（§9）：缺主键 id 时「下发并对账」禁用。
  test('VLAN 表单缺主键(id)时下发应被校验拦截', async ({ page }) => {
    await page.goto('/module/vlan', { waitUntil: 'networkidle' })
    await pickDevice(page)
    await page.locator(SEL.tab, { hasText: /^VLAN列表$/ }).first().click()
    await page.getByRole('button', { name: '创建' }).first().click()
    const pane = page.locator('[data-test="item-detail-pane"]')
    await expect(pane.getByText('VLAN管理状态', { exact: false }).first()).toBeVisible({ timeout: 15000 })

    await expect(pane.locator('[data-test="detail-submit"]')).toBeDisabled()
  })

  // 攒批闭环（FE-03/FE-21/FE-23 二期）：创建入集 → 徽标/变更内容 → 提交配置 →
  // 原子下发收敛 → 列表出现新条目。唯一一条 write-path 冒烟（对 staging 模拟网元）。
  test('攒批闭环：创建 VLAN 入集 → 变更内容核对 → 提交配置 → 列表可见', async ({ page }) => {
    const vlanId = String(3000 + (Date.now() % 900)) // 避开种子与历史运行残留
    await page.goto('/module/vlan', { waitUntil: 'networkidle' })
    await pickDevice(page)
    await page.locator(SEL.tab, { hasText: /^VLAN列表$/ }).first().click()
    await page.getByRole('button', { name: '创建' }).first().click()

    const pane = page.locator('[data-test="item-detail-pane"]')
    await expect(pane.getByText('VLAN标识', { exact: false }).first()).toBeVisible({ timeout: 15000 })
    // key 叶（VLAN标识，number 控件）：定位该表单项内的输入框
    await pane
      .locator(SEL.formItem)
      .filter({ hasText: 'VLAN标识' })
      .first()
      .locator('input')
      .first()
      .fill(vlanId)
    await expect(pane.locator('[data-test="detail-submit"]')).toBeEnabled()
    await pane.locator('[data-test="detail-submit"]').click()

    // 入集：待创建标记行 + 工具栏徽标
    await expect(page.locator('[data-test="mark-create"]').first()).toBeVisible()
    await expect(page.locator(SEL.badgeCount).first()).toHaveText(/[1-9]/)

    // 变更内容弹窗核对后关闭
    await page.locator('[data-test="batch-changes"]').click()
    const changesDialog = page.getByRole('dialog').filter({ hasText: '变更内容' })
    await expect(changesDialog.getByText(vlanId, { exact: false }).first()).toBeVisible()
    await changesDialog.locator(SEL.modalClose).click()

    // 提交配置：确认 → 进度弹窗 → 完成关闭
    await page.locator('[data-test="batch-commit"]').click()
    await page.getByRole('button', { name: /确\s*定/ }).last().click()
    const commitDialog = page.getByRole('dialog').filter({ hasText: '提交配置' })
    await expect(commitDialog.locator('[data-test="commit-close"]')).toBeEnabled({ timeout: 30000 })
    await expect(commitDialog.locator('[data-test="commit-error"]')).toHaveCount(0)
    await commitDialog.locator('[data-test="commit-close"]').click()

    // 提交后：徽标清零、列表出现新条目（限定数据行作用域防隐藏面板误命中）
    await expect(page.locator(SEL.tableRow).filter({ hasText: vlanId }).first()).toBeVisible({ timeout: 15000 })
  })

  // 接口（华为 IFM）：Tab 由模块根派生，interfaces 列表 Tab 内创建表单动态渲染。
  test('接口控制台 Tab 派生 + 创建表单动态渲染出 YANG 字段', async ({ page }) => {
    await page.goto('/module/ifm', { waitUntil: 'networkidle' })
    await expect(page).toHaveURL(/module\/ifm/)

    await pickDevice(page)
    await page.locator(SEL.tab, { hasText: /^接口列表$/ }).first().click()
    await page.getByRole('button', { name: '创建' }).first().click()

    // mtu 为 IFM 叶子名，schema 动态渲染才会出现
    await expect(page.getByText('mtu', { exact: false }).first()).toBeVisible({ timeout: 15000 })
  })

  // rpc 入口收敛到左树（FE-19/LT-03）：模块叶展开出 container 与 rpc 平级节点，
  // 点 rpc 节点直达执行页（/module/ifm/rpc/<name>），控制台 Tab 栏不再有 rpc Tab。
  // 标签经烘焙双语（LT-01）：树节点显示中文「按接口名清除统计」。
  test('左树展开模块叶出 rpc 节点，直达执行页渲染输入与执行按钮', async ({ page }) => {
    await page.goto('/module/ifm', { waitUntil: 'networkidle' })
    await pickDevice(page)

    // Tab 栏无 rpc Tab（导航落点已迁移到左树）。
    await expect(page.locator(SEL.tab, { hasText: /^接口列表$/ }).first()).toBeVisible({ timeout: 15000 })
    await expect(page.locator(SEL.tab, { hasText: /^按接口名清除统计$/ }).first()).toHaveCount(0)

    // 左树：接口管理 → 接口基础 → huawei-ifm（可展开叶）→ rpc 节点。
    await page.locator('[data-test="lefttree-group-接口管理"]').first().click()
    await page.locator('[data-test="lefttree-group-接口基础"]').first().click()
    await page.locator('[data-test="lefttree-leaf-huawei-ifm"]').first().click()
    // container 节点（通用接口）与 rpc 节点平级可见。
    await expect(page.locator('[data-test="lefttree-node-ifm"]')).toBeVisible({ timeout: 15000 })
    const rpcNode = page.locator('[data-test="lefttree-rpc-ifm-reset-if-counters-by-name"]')
    await expect(rpcNode).toContainText('按接口名清除统计')
    await rpcNode.click()
    await expect(page).toHaveURL(/module\/ifm\/rpc\/reset-if-counters-by-name/)

    // 执行页：仅该 rpc 面板（无 Tab 栏），input + 执行按钮渲染（schema 驱动）。
    await expect(page.locator('[data-test="rpc-execute"]')).toBeVisible({ timeout: 15000 })
    await expect(page.locator('[data-test="rpc-execute-tab"] .sub-field').first()).toBeVisible()
    // 缺 mandatory input → 执行按钮禁用（§9 校验拦截）。
    await expect(page.locator('[data-test="rpc-execute"]')).toBeDisabled()
  })

  // 种子数据（模拟网元 DemoSeedConfig）：5 条接口回读进表格，sub 行显示 parent-name。
  test('接口列表应展示模拟网元种子行（3 main + 2 sub）', async ({ page }) => {
    await page.goto('/module/ifm', { waitUntil: 'networkidle' })
    await pickDevice(page)
    await page.locator(SEL.tab, { hasText: /^接口列表$/ }).first().click()

    await expect(page.getByText('200GE0/1/0', { exact: true }).first()).toBeVisible({ timeout: 20000 })
    await expect(page.getByText('200GE0/1/1.1', { exact: false }).first()).toBeVisible()
  })

  // config=false 只读字段回显（NS-08/BR-01）：读路径 <get> 带回状态种子，
  // 详情编辑区（FE-21）内 dynamic 嵌套组成为二级 Tab，mac-address 以禁用态展示状态值。
  test('接口详情区回显 config=false 状态字段（dynamic/mac-address）', async ({ page }) => {
    await page.goto('/module/ifm', { waitUntil: 'networkidle' })
    await pickDevice(page)
    await page.locator(SEL.tab, { hasText: /^接口列表$/ }).first().click()
    await expect(page.getByText('200GE0/1/0', { exact: true }).first()).toBeVisible({ timeout: 20000 })

    // 打开种子行 200GE0/1/0 的详情编辑区（master-detail，FE-21）
    const row = page.locator(SEL.tableRow, { hasText: '200GE0/1/0' }).first()
    await row.getByRole('button', { name: '编辑' }).click()

    // dynamic 嵌套容器（SND i18n 汉化为「接口动态信息」）现为详情区二级 Tab；
    // 切过去后其子叶渲染为 .sub-field（标签「生效MAC地址」）。
    // 接口详情 Tab 有 47 个，antd Tabs 溢出折叠：目标 Tab 收在「更多」下拉里
    //（直点 nav 里的 tab 节点落在视口外、不触发切换），走下拉切换。
    const pane = page.locator('[data-test="item-detail-pane"]')
    // TODO-E2E(内网首跑): antd 的「更多」溢出下拉在 eview 为 observerWidthChange
    // 折叠，形态未实证——先按标签直点（若溢出隐藏不可点，内网首跑按真实
    // DOM 校准）。
    await pane.locator(`.detail-tabs ${SEL.tab}`, { hasText: '接口动态信息' }).first().click()
    const macRow = pane.locator('.sub-field').filter({ hasText: '生效MAC地址' }).first()
    await expect(macRow.locator('input').first()).toHaveValue('00:e0:fc:12:34:01', { timeout: 15000 })
    await expect(macRow.locator('input').first()).toBeDisabled()
  })

  // 高级搜索（ext:support-filter 驱动）：class=sub-interface 过滤后主接口行消失。
  test('高级搜索按 class 过滤（support-filter 驱动）', async ({ page }) => {
    await page.goto('/module/ifm', { waitUntil: 'networkidle' })
    await pickDevice(page)
    await page.locator(SEL.tab, { hasText: /^接口列表$/ }).first().click()
    await expect(page.getByText('200GE0/1/0', { exact: true }).first()).toBeVisible({ timeout: 20000 })

    await page.getByRole('button', { name: /高级搜索/ }).click()
    const panel = page.locator('.search-panel')
    await panel.locator(SEL.select).first().click()
    await page.locator(`${SEL.selectOption}:visible`, { hasText: 'sub-interface' }).first().click()
    // antd 两字按钮自动插空格（「查 询」），按正则匹配。
    await panel.getByRole('button', { name: /查\s*询/ }).click()

    // 主接口行被过滤掉，仅剩 2 条 sub-interface。断言限定在表格行内：rpc 的
    // if-name leafref 下拉（多个 rpc 面板常驻）也把接口名作为 teleport 选项渲染进
    // 页面，页面级 getByText 会误命中这些隐藏下拉项（实测 6 个），故只查表格行。
    await expect(page.locator(SEL.tableRow, { hasText: '200GE0/1/2' })).toHaveCount(0)
    await expect(page.locator(SEL.tableRow, { hasText: '200GE0/1/0.1' }).first()).toBeVisible()
  })

  // 接口 when 约束（FE-07）：parent-name 由 YANG `when "../class='sub-interface'"` 门控。
  // 断言限定在详情编辑区内（页面其他区域可能出现同名文本）。
  test('接口 when 约束：class=sub-interface 才显现 parent-name（数据驱动显隐）', async ({ page }) => {
    await page.goto('/module/ifm', { waitUntil: 'networkidle' })
    await pickDevice(page)
    await page.locator(SEL.tab, { hasText: /^接口列表$/ }).first().click()
    await page.getByRole('button', { name: '创建' }).first().click()

    const pane = page.locator('[data-test="item-detail-pane"]')
    await expect(pane.getByText('接口类别', { exact: false }).first()).toBeVisible({ timeout: 15000 })
    await expect(pane.locator(SEL.formItemLabel, { hasText: '主接口名' })).toHaveCount(0)

    // 精确定位 class 字段的下拉（UI-03 后标签为「接口类别」），
    // 并只点“可见”的下拉项（teleport 的历史下拉会残留在 DOM 中）。
    const classItem = pane.locator(SEL.formItem, {
      has: page.locator(SEL.formItemLabel, { hasText: /^接口类别$/ }),
    })
    await classItem.locator(SEL.select).click()
    await page.locator(`${SEL.selectOption}:visible`, { hasText: 'sub-interface' }).first().click()

    await expect(pane.getByText('主接口名', { exact: false }).first()).toBeVisible({ timeout: 15000 })
  })

  // SPA 内从 VLAN 模块切到 IFM 模块应重载 schema（回归门禁：路由参数变化 → schema 重载）。
  test('SPA 内从 VLAN 切换到接口模块应加载接口模型（非沿用 VLAN）', async ({ page }) => {
    await page.goto('/module/vlan', { waitUntil: 'networkidle' })
    // 全局设备上下文（FE-10）：Tab 内容区以已选设备为前提
    await pickDevice(page)
    await expect(page.locator(SEL.tab, { hasText: /^VLAN列表$/ }).first()).toBeVisible({ timeout: 15000 })

    // 侧栏 SND 左树（LT-03）：展开 接口管理→接口基础→huawei-ifm 叶，点 container 节点。
    await page.locator('[data-test="lefttree-group-接口管理"]').first().click()
    await page.locator('[data-test="lefttree-group-接口基础"]').first().click()
    await page.locator('[data-test="lefttree-leaf-huawei-ifm"]').first().click()
    await page.locator('[data-test="lefttree-node-ifm"]').click()
    await expect(page).toHaveURL(/module\/ifm/)

    // 设备上下文跨模块保持：无需重新选设备
    await page.locator(SEL.tab, { hasText: /^接口列表$/ }).first().click()
    await page.getByRole('button', { name: '创建' }).first().click()

    // 接口独有字段 mtu 应出现（若仍沿用 VLAN schema 则不会有）
    await expect(page.getByText('mtu', { exact: false }).first()).toBeVisible({ timeout: 15000 })
  })

  // ===== 全量 YANG 模型接入（full-yang-onboarding，LT-04）=====

  // 左树全量可用基线（部署面）：可用叶 = 全部叶 − 5 例外（pic 延期 + 4 个
  // augment-only）。与后端 LT-04 单测同口径，这里验证的是「已部署容器」。
  test('左树全量可用：60/65 叶 available 且带 module', async ({ request }) => {
    // 后端直连（staging nginx 不代理 /api，相对路径会命中 SPA fallback 返回 HTML）；
    // 地址口径与前端 api 客户端一致（VITE_API_URL 缺省 localhost:8080）。
    const apiBase = process.env.USMP_API_URL || 'http://localhost:8080/api/v1'
    const res = await request.get(`${apiBase}/yang/left-tree`)
    expect(res.ok()).toBeTruthy()
    const body = await res.json()
    const leaves: any[] = []
    const walk = (nodes: any[]) => {
      for (const n of nodes || []) {
        // 模块叶现在也带 children（LT-02 模块级 container/rpc 子节点），
        // 叶判定以 sourceModule 为准。
        if (n.sourceModule) leaves.push(n)
        else walk(n.children || [])
      }
    }
    walk(body.data)
    const available = leaves.filter((l) => l.available)
    // 相对不变式（与后端 LT-04 同口径）：可用 = 全部 − 5 例外
    //（pic 延期 + 4 augment-only），加模块不需要改这里的字面量
    expect(leaves.length).toBeGreaterThanOrEqual(65)
    expect(available.length).toBe(leaves.length - 5)
    for (const l of available) expect(l.module, l.sourceModule).toBeTruthy()
  })

  // 新接入模块控制台冒烟：ntp（全量接入前不可用）schema 驱动渲染出 Tab。
  test('新模块（ntp）控制台 schema 驱动渲染', async ({ page }) => {
    await page.goto('/module/ntp', { waitUntil: 'networkidle' })
    await pickDevice(page)
    await expect(page.locator(SEL.tab).first()).toBeVisible({ timeout: 15000 })
    await expect(page.locator('[data-test="select-device-empty"]')).toHaveCount(0)
  })

  // ===== 全局设备上下文（device-first-config-context，FE-10）=====

  // 先选设备、后做配置管理：设备管理「查看配置」写入全局上下文，跨模块切换保持。
  test('查看配置进入控制台后切换模块，设备选中保持不丢', async ({ page }) => {
    await page.goto('/devices', { waitUntil: 'networkidle' })
    const row = page.locator(SEL.tableRow, { hasText: DEVICE_IP }).first()
    await row.getByRole('button', { name: '查看配置' }).click()
    await expect(page).toHaveURL(/module\/ifm/)

    // 入口已写上下文：无需选设备，Tab 直接可用
    await expect(page.locator(SEL.tab, { hasText: /^接口列表$/ }).first()).toBeVisible({ timeout: 15000 })
    await expect(page.locator('[data-test="select-device-empty"]')).toHaveCount(0)

    // 左树切到 VLAN 模块：设备上下文沿用，Tab 直接渲染（未选设备时只会显示引导空态）
    await page.locator('[data-test="lefttree-group-以太网交换"]').first().click()
    await page.locator('[data-test="lefttree-group-VLAN"]').first().click()
    await page.locator('[data-test="lefttree-leaf-huawei-vlan"]').first().click()
    await page.locator('[data-test="lefttree-node-vlan"]').click()
    await expect(page).toHaveURL(/module\/vlan/)
    await expect(page.locator(SEL.tab, { hasText: /^VLAN列表$/ }).first()).toBeVisible({ timeout: 15000 })
    await expect(page.locator('[data-test="select-device-empty"]')).toHaveCount(0)
  })

  // 未选设备：引导空态而非静默空数据。
  test('直开控制台且无设备上下文时展示引导空态', async ({ page }) => {
    await page.goto('/module/vlan', { waitUntil: 'networkidle' })
    await expect(page.locator('[data-test="select-device-empty"]')).toBeVisible({ timeout: 15000 })
    await expect(page.locator(SEL.tab, { hasText: /^VLAN列表$/ }).first()).toHaveCount(0)
  })
})

// ===== 业务网络配置（FE-17，矩阵 A8/A11/A12 冒烟）=====
// staging 无 K8s 集群：菜单与 schema 渲染必须可用（/yang/modules、/yang/schema
// 不依赖集群），实例数据面诚实呈现 503 降级告警（R08）。
test.describe('部署冒烟 - 业务网络配置', () => {
  test('业务菜单组出现且业务控制台渲染（含无集群降级告警）', async ({ page }) => {
    await page.goto('/', { waitUntil: 'networkidle' })

    // 菜单组由 task-name=business-network 自动分桶（零硬编码）。
    const group = page.locator('[data-test="business-group"]')
    await expect(group).toBeVisible({ timeout: 15000 })
    await group.click()
    await page.locator('[data-test="business-item-business-vlan-service"]').click()
    await expect(page).toHaveURL(/business\/business-vlan-service/)

    // 页面骨架 + 业务台状态（compose staging 无 apiserver→降级告警；kind
    // 环境有真集群→业务表格——环境自适应断任一，组 7）。
    await expect(page.getByText('业务网络配置').first()).toBeVisible()
    await expect(
      page.locator('[data-test="business-table"], [data-test="business-unavailable"]').first(),
    ).toBeVisible({ timeout: 15000 })
  })

  test('新建抽屉由意图 YANG schema 驱动渲染（devices 嵌套 list）', async ({ page }) => {
    await page.goto('/business/business-vlan-service', { waitUntil: 'networkidle' })
    await page.locator('[data-test="business-create"]').click()

    // 页面含新建/详情两个抽屉容器，按 aria-label 精确定位（strict mode）。
    const drawer = page.getByRole('dialog', { name: '新建业务实例' })
    await expect(drawer).toBeVisible({ timeout: 15000 })
    await expect(drawer.getByText('实例名').first()).toBeVisible()
    // schema 派生字段（意图 YANG 叶子）与嵌套 devices list 的添加入口。
    await expect(drawer.getByText('vlan-id', { exact: false }).first()).toBeVisible({ timeout: 15000 })
    await expect(drawer.getByRole('button', { name: /添加/ }).first()).toBeVisible()

    // 校验拦截：缺实例名/必填字段时提交按钮禁用（§9 不提交）。
    await expect(drawer.locator('[data-test="business-submit"]')).toBeDisabled()
  })

  test('SPA 内业务控制台 ⇄ 原生模块控制台切换不串 schema', async ({ page }) => {
    await page.goto('/business/business-vlan-service', { waitUntil: 'networkidle' })
    await expect(page.locator('[data-test="business-table"], [data-test="business-unavailable"]').first()).toBeVisible({ timeout: 15000 })

    // 原生配置子菜单默认展开（openKeys 初始含 native-config），直接逐层展开 SND 左树再点 container 节点（LT-03）。
    await page.locator('[data-test="lefttree-group-以太网交换"]').first().click()
    await page.locator('[data-test="lefttree-group-VLAN"]').first().click()
    await page.locator('[data-test="lefttree-leaf-huawei-vlan"]').first().click()
    await page.locator('[data-test="lefttree-node-vlan"]').click()
    await expect(page).toHaveURL(/module\/vlan/)
    // 全局设备上下文（FE-10）：业务侧进入无设备选中，先选设备再断言 Tab
    await pickDevice(page)
    await expect(page.locator(SEL.tab, { hasText: /^VLAN列表$/ }).first()).toBeVisible({ timeout: 15000 })
  })
})

test.describe('部署冒烟 - 语言切换（UI-01）', () => {
  test('切换 en-us 导航变英文并持久化，切回 zh-cn 收尾', async ({ page }) => {
    await page.goto('/', { waitUntil: 'networkidle' })
    await expect(page.locator('.ub-menu').getByText('设备管理')).toBeVisible({ timeout: 15000 })

    await page.locator('[data-test="locale-switch"]').click()
    // Dropdown 桥 label 文本化丢 data-test（锚点债，tasks 7.1b）——按弹层
    // 选项文本选（语言名不随界面语言翻译，恒定安全）。
    await page.locator(SEL.selectOption, { hasText: 'English' }).first().click()
    await expect(page.locator('.ub-menu').getByText('Devices', { exact: true })).toBeVisible({ timeout: 5000 })
    await expect(page.locator('.ub-menu').getByText('Native Configuration')).toBeVisible()

    // 刷新持久化（localStorage）
    await page.reload({ waitUntil: 'networkidle' })
    await expect(page.locator('.ub-menu').getByText('Devices', { exact: true })).toBeVisible({ timeout: 15000 })

    // 收尾切回 zh，避免影响后续用例顺序无关性
    await page.locator('[data-test="locale-switch"]').click()
    await page.locator(SEL.selectOption, { hasText: '中文' }).first().click()
    await expect(page.locator('.ub-menu').getByText('设备管理')).toBeVisible({ timeout: 5000 })
  })
})
