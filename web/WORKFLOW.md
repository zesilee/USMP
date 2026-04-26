# 前端开发工作流

## 完整开发流程

```
需求分析
    ↓
┌─ 使用 e2e-testing 技能编写 E2E 测试 ─┐
│   • 页面加载测试                        │
│   • 核心流程测试                        │
│   • 表单交互测试                        │
│   • 异常场景测试                        │
└─────────────────────────────────────────┘
    ↓
运行 E2E 测试（应该失败，TDD 模式）
    ↓
┌─ 编写单元测试 (Vitest) ───────────────┐
│   • 组件渲染测试                        │
│   • Props 传递测试                      │
│   • 事件触发测试                        │
└─────────────────────────────────────────┘
    ↓
运行单元测试（应该失败）
    ↓
功能开发
    ↓
单元测试通过 ✅
    ↓
E2E 测试通过 ✅
    ↓
代码审查
    ↓
提交代码
```

---

## 必须添加 E2E 测试的场景

### 1. 新增页面
- ✅ 页面标题、布局、核心组件渲染
- ✅ 页面导航、路由跳转
- ✅ 数据加载、Loading 状态

### 2. 新增表单功能
- ✅ 表单输入验证
- ✅ 提交成功流程
- ✅ 取消/重置功能
- ✅ 错误提示展示

### 3. 新增表格/列表
- ✅ 数据展示正确性
- ✅ 搜索、筛选功能
- ✅ 分页功能
- ✅ 行操作（编辑、删除）

### 4. 弹窗 / 抽屉 / 模态框
- ✅ 打开、关闭行为
- ✅ 内容渲染
- ✅ 确认、取消按钮

### 5. 核心业务流程
- ✅ VLAN 创建、编辑、删除
- ✅ 接口配置下发
- ✅ 设备添加、删除
- ✅ 配置刷新、同步

---

## E2E 测试编写规范

### 第一步：创建 Page Object

每个页面创建对应的 Page Object 文件：

```typescript
// tests/pages/XxxPage.ts
import { Page, expect, Locator } from '@playwright/test'

export class XxxPage {
  readonly page: Page
  readonly addButton: Locator
  readonly submitButton: Locator

  constructor(page: Page) {
    this.page = page
    this.addButton = page.getByRole('button', { name: '新建' })
    this.submitButton = page.getByRole('button', { name: '提交' })
  }

  async goto() {
    await this.page.goto('/')
    // 导航到对应页面...
  }

  async clickAdd() {
    await this.addButton.click()
  }

  async verifyItemExists(name: string) {
    await expect(this.page.getByText(name)).toBeVisible()
  }
}
```

### 第二步：编写测试用例

```typescript
// tests/xxx.spec.ts
import { test, expect } from '@playwright/test'
import { XxxPage } from './pages/XxxPage'

test.describe('功能模块 - E2E 测试', () => {
  let page: XxxPage

  test.beforeEach(async ({ page }) => {
    page = new XxxPage(page)
    await page.goto()
  })

  test('场景描述 - 应该做什么', async () => {
    // Arrange - 准备

    // Act - 执行

    // Assert - 断言
  })
})
```

### 第三步：运行测试验证

```bash
# 开发时使用可视化界面
npm run e2e:ui

# 完成后完整运行
npm run e2e
```

---

## 测试执行要求

### 本地开发

```bash
# 1. 先跑单元测试
npm run test

# 2. 再跑 E2E 测试
npm run e2e

# 3. 构建验证
npm run build
```

### 提交前检查清单

- [ ] 新增功能有对应的单元测试
- [ ] 新增功能有对应的 E2E 测试
- [ ] 所有单元测试通过
- [ ] 所有 E2E 测试通过
- [ ] 测试覆盖了正常 + 异常场景
- [ ] 代码构建无错误
- [ ] 无 TypeScript 类型错误

---

## 测试金字塔

```
        /\
       /  \        E2E 测试 (少量)
      /____\       - 核心业务流程
     /      \
    /  单元  \     单元测试 (大量)
   /   测试   \    - 组件行为
  /____________\   - 交互逻辑
 /              \
/  类型安全     \  TypeScript (全部)
/________________\  - 编译时检查
```

| 测试类型 | 数量 | 执行速度 | 覆盖范围 |
|---------|------|---------|---------|
| 类型检查 | 全部 | < 1s | 类型安全 |
| 单元测试 | 大量 | 几秒 | 组件行为 |
| E2E 测试 | 少量 | 几十秒 | 用户流程 |

---

## 与技能联动

| 技能 | 联动时机 | 输出 |
|------|---------|------|
| **e2e-testing** | 功能开发前 | E2E 测试用例 |
| tdd-test-driven-dev | 组件开发前 | 单元测试用例 |
| frontend-yang-dynamic-form | YANG 页面开发 | 动态表单组件 |
| netconf-sim-integration-test | 后端集成测试 | 全链路验证 |

---

## 验收标准

每次前端功能 PR 必须满足：

- ✅ 新增/修改的功能有对应的 E2E 测试
- ✅ 所有单元测试通过
- ✅ 所有 E2E 测试通过
- ✅ 测试覆盖正常流程 + 至少 1 个异常场景
- ✅ 使用 Page Object 模式封装页面操作
- ✅ 无 flaky 测试（CI 中可稳定执行）
- ✅ `npm run build` 构建成功
- ✅ TypeScript 类型检查无错误
