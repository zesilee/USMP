# 前端测试指南

## 测试分层架构

```
┌─────────────────────────────────────────┐
│          E2E 端到端测试                  │
│      真实浏览器 + 模拟网元后端           │
│     Playwright - 验证完整用户流程        │
├─────────────────────────────────────────┤
│          单元/组件测试                    │
│     Vitest + Vue Test Utils             │
│     验证组件行为、交互、数据处理         │
├─────────────────────────────────────────┤
│          静态类型检查                    │
│     TypeScript - 编译时类型安全         │
└─────────────────────────────────────────┘
```

---

## 一、单元测试 (Vitest)

### 运行命令

```bash
# 运行所有测试
npm run test

# 监听模式开发
npm run test:watch

# 可视化界面
npm run test:ui

# 生成覆盖率报告
npm run test:coverage
```

### 测试文件位置

```
src/
├── __tests__/
│   ├── utils/
│   │   └── index.ts          # 测试工具函数
│   ├── StatusBadge.spec.ts   # 状态徽章测试
│   ├── VlanTable.spec.ts     # VLAN 表格测试
│   ├── VlanFormDrawer.spec.ts # 表单抽屉测试
│   └── VlanManager.spec.ts   # VLAN 管理器测试
```

### 当前测试覆盖

| 组件 | 测试数 | 覆盖率 |
|------|--------|--------|
| StatusBadge | 6 | 100% |
| VlanTable | 6 | 80% |
| VlanFormDrawer | 6 | 70% |
| VlanManager | 7 | 60% |

---

## 二、E2E 端到端测试 (Playwright)

### 核心架构

```
Playwright 浏览器
      │
      ▼
   前端 Vue 应用 (Vite)
      │
      ▼
 Vite 代理 :3000 -> :8080
      │
      ▼
测试后端服务器 (test-server)
      │
      ▼
 NETCONF 模拟网元 (netsim)
```

### 运行命令

```bash
# 完整 E2E 测试（自动启动后端+前端）
npm run e2e

# 可视化界面调试
npm run e2e:ui

# 有头模式运行（可见浏览器操作）
npm run e2e:headed

# 查看测试报告
npm run e2e:report

# 只运行特定测试文件
npm run e2e -- tests/e2e-demo.spec.ts

# 指定浏览器
npm run e2e -- --project=chromium
npm run e2e -- --project=firefox
npm run e2e -- --project=webkit
```

### 测试文件位置

```
tests/
├── pages/
│   └── VlanPage.ts           # Page Object 封装
├── e2e-demo.spec.ts          # 环境验证测试
├── vlan.spec.ts              # VLAN 功能测试
└── navigation.spec.ts        # 页面导航测试
```

### Page Object 模式

每个页面封装成独立的 Page Object，提高测试可维护性：

```typescript
// tests/pages/VlanPage.ts
export class VlanPage {
  readonly page: Page
  readonly addVlanButton: Locator

  constructor(page: Page) {
    this.page = page
    this.addVlanButton = page.getByRole('button', { name: '新建 VLAN' })
  }

  async goto() { await this.page.goto('/') }
  async clickAddVlan() { await this.addVlanButton.click() }
  async verifyVlanExists(id: number) { ... }
}
```

### 测试用例编写规范

```typescript
test.describe('功能模块', () => {
  let page: VlanPage

  test.beforeEach(async ({ page }) => {
    page = new VlanPage(page)
    await page.goto()
  })

  test('用例描述 - 应该做什么', async () => {
    // Arrange - 准备

    // Act - 执行

    // Assert - 断言
  })
})
```

### 元素定位最佳实践

| 优先级 | 方法 | 示例 |
|--------|------|------|
| 1 | Role 定位 | `page.getByRole('button', { name: '提交' })` |
| 2 | Label 定位 | `page.getByLabel('VLAN ID')` |
| 3 | Text 定位 | `page.getByText('新建 VLAN')` |
| 4 | data-testid | `page.getByTestId('btn-submit')` |
| ❌ 避免 | CSS 类名 | `page.locator('.el-button--primary')` |

---

## ⚠️ E2E 测试避坑指南 (血泪经验)

### 问题案例：CORS 端口不匹配导致验收失败

**现象：**
- E2E 测试全部通过 ✅
- 开发人员手动验收时，设备树为空 ❌

**根因：**
```go
// 后端 CORS 只配置了 5173 端口
AllowOrigins: []string{"http://localhost:5173"}

// 但 Vite 实际运行在 3000 端口
// Playwright 配置不一致
baseURL: 'http://localhost:5173'
webServer: { url: 'http://localhost:3000' }
```

**教训：**

| 序号 | 规则 | 强制执行方式 |
|------|------|-------------|
| **1** | **测试环境必须与开发环境完全一致** | `playwright.config.ts` 中 `baseURL` 必须与 `webServer.url` 端口一致 |
| **2** | **必须验证前置条件** | beforeEach 中不能只做操作，必须显式断言关键前置条件 |
| **3** | **所有涉及的端口必须在 CORS 白名单** | 后端 CORS 配置必须包含所有可能的前端端口 |

---

### E2E 测试编写黄金法则

#### ✅ 法则 1：显式验证前置条件

**错误写法 ❌**
```typescript
test.beforeEach(async ({ page }) => {
  await page.goto('/')
  await page.click('text=VLANs')  // 假设元素存在，CORS 失败时这里会静默超时
})
```

**正确写法 ✅**
```typescript
test.beforeEach(async ({ page }) => {
  await page.goto('/')
  // 显式断言设备树加载成功 - 验证 CORS + API 正常工作
  await expect(page.getByText('192.168.1.1')).toBeVisible()
  // 再执行后续操作
  await page.getByText('VLANs').first().click()
})
```

#### ✅ 法则 2：端口配置单一数据源

**项目约定：**
```
前端开发端口：3000 (Vite 默认)
后端测试端口：8080
```

所有配置文件必须统一：
- `playwright.config.ts`: `baseURL: 'http://localhost:3000'`
- `vite.config.ts`: `server.port: 3000`
- 后端 CORS: 必须包含 `http://localhost:3000`

#### ✅ 法则 3：环境一致性检查

在 `e2e-demo.spec.ts` 中必须包含：
```typescript
// 验证 API 连通性
test('后端 API 应该可以正常访问', async ({ request }) => {
  const res = await request.get('/api/v1/devices')
  expect(res.ok()).toBeTruthy()
})
```

---

## 三、测试后端服务器

### 启动方式

测试脚本会自动启动后端服务器，也可以手动启动：

```bash
# 从项目根目录运行
cd backend && go run ./cmd/test-server/main.go
```

### 模拟网元功能

`backend/simulator/netsim/simulator.go` 提供：

| 功能 | 说明 |
|------|------|
| VLAN 配置存储 | 内存存储，支持 CRUD |
| VLAN 状态模拟 | adminStatus, operStatus |
| 端口关联 | tagged/untagged 端口列表 |
| API 接口 | 完整的 REST API 与真实后端一致 |

### 测试数据

默认预置 4 个 VLAN：

| ID | 名称 | 状态 |
|----|------|------|
| 1 | default | UP / ACTIVE |
| 10 | Management | UP / ACTIVE |
| 20 | User_Network | UP / ACTIVE |
| 30 | Guest | DOWN / INACTIVE |

---

## 四、CI/CD 集成

### GitHub Actions 示例

```yaml
name: Frontend Tests
on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
      - run: cd frontend && npm ci
      - run: cd frontend && npm run test
      - run: cd frontend && npm run build

  e2e-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
      - uses: actions/setup-go@v5
      - run: cd frontend && npm ci
      - run: cd frontend && npx playwright install chromium
      - run: cd frontend && npm run e2e -- --project=chromium
```

---

## 五、测试验收标准

每次前端功能 PR 必须满足：

| 检查项 | 要求 |
|--------|------|
| 单元测试 | 全部通过 |
| E2E 测试 | 全部通过 |
| 新功能测试覆盖 | 正常 + 异常场景 |
| 构建 | `npm run build` 无错误 |
| 代码质量 | ESLint 无错误 |

---

## 六、常见问题

### Q: Playwright 浏览器未安装？
```bash
npx playwright install chromium
```

### Q: 端口被占用？
```bash
# 清理 8080 端口
lsof -ti :8080 | xargs kill -9
```

### Q: 测试在 CI 中不稳定？
- 增加超时时间
- 使用 `waitFor` 替代固定等待
- 减少测试间的依赖

### Q: 如何调试失败的测试？
```bash
# 有头模式运行
npm run e2e:headed

# 查看失败截图
ls test-results/

# 查看完整报告
npm run e2e:report
```

### Q: 如何只运行特定测试？
```bash
# 按文件名过滤
npm run e2e -- vlan

# 按测试名过滤
npm run e2e -- -g "新建 VLAN"
```

---

## 七、技能联动

| 技能 | 联动场景 |
|------|----------|
| **e2e-testing** | 新增前端功能、修改核心交互流程 |
| frontend-yang-dynamic-form | 动态表单开发完成后添加 E2E 测试 |
| netconf-sim-integration-test | 后端集成测试 + 前端 E2E = 全链路验证 |
