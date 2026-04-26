---
name: e2e-testing
description: 使用 Playwright 编写端到端测试，覆盖页面导航、表单提交、表格操作、弹窗交互等真实用户场景
---

# E2E 端到端测试技能

## 一、激活时机
1. **前端新功能开发完成后**自动触发，为新增页面/组件编写 E2E 测试
2. 修改核心交互流程（表单提交、数据删除、批量操作）时补充测试
3. 每次 VLAN、Interfaces、System 等 YANG 模块前端开发完成后
4. 前端重构后验证功能完整性时

## 二、测试范围要求

### 必须覆盖的场景
| 场景类型 | 具体内容 |
|---------|---------|
| **页面加载** | 页面正常渲染、标题显示、骨架屏消失 |
| **表格操作** | 数据展示、排序、分页、搜索筛选 |
| **表单交互** | 输入验证、提交成功、错误提示 |
| **弹窗/抽屉** | 打开、关闭、提交、取消行为 |
| **按钮操作** | 新增、编辑、删除、刷新、批量删除 |
| **状态展示** | Loading 状态、成功/失败提示、设备在线状态 |

### 必须包含的测试类型
1. **冒烟测试** - 核心功能是否正常工作
2. **流程测试** - 完整用户操作路径
3. **异常测试** - 边界情况、错误输入、网络异常
4. **视觉回归** - 关键组件渲染正确性

## 三、最佳实践规范

### 1. Page Object Model (POM) 模式
```typescript
// tests/pages/VlanPage.ts
export class VlanPage {
  constructor(readonly page: Page) {}
  
  async goto() { await this.page.goto('/vlans') }
  async clickAddVlan() { await this.page.getByTestId('btn-add-vlan').click() }
  async fillVlanForm(id: number, name: string) { ... }
}
```

### 2. 元素定位规范
- ✅ **首选**: `data-testid` 属性 - `page.getByTestId('btn-submit')`
- ✅ **其次**: 角色定位 - `page.getByRole('button', { name: '提交' })`
- ✅ **再次**: 标签文本 - `page.getByLabel('VLAN ID')`
- ❌ **禁止**: CSS 类名、DOM 层级、XPath 绝对路径

### 3. 测试用例编写规范
```typescript
test.describe('VLAN 管理页面', () => {
  // 每个测试独立初始化
  let vlanPage: VlanPage
  
  test.beforeEach(async ({ page }) => {
    vlanPage = new VlanPage(page)
    await vlanPage.goto()
  })
  
  // 测试名称清晰描述行为
  test('新建 VLAN - 输入有效信息应创建成功', async () => {
    // Arrange - 准备
    await vlanPage.clickAddVlan()
    
    // Act - 执行
    await vlanPage.fillVlanForm(100, 'Test_VLAN')
    await vlanPage.submit()
    
    // Assert - 断言
    await expect(vlanPage.getSuccessToast()).toBeVisible()
    await expect(vlanPage.getRowById(100)).toBeVisible()
  })
})
```

### 4. 测试目录结构
```
web/
├── tests/
│   ├── pages/              # Page Objects
│   │   ├── VlanPage.ts
│   │   ├── DeviceTreePage.ts
│   │   └── FormPage.ts
│   ├── vlan.spec.ts        # VLAN 功能测试
│   ├── devices.spec.ts     # 设备管理测试
│   └── navigation.spec.ts  # 导航测试
├── playwright.config.ts
└── TESTING.md              # 已包含 E2E 章节
```

## 四、与其他技能联动

1. **tdd-test-driven-dev** - 单元测试覆盖组件逻辑，E2E 覆盖用户交互流程
2. **frontend-yang-dynamic-form** - 每个动态表单页面都要有对应的 E2E 测试
3. **netconf-sim-integration-test** - 后端集成测试 + 前端 E2E = 全链路验证

## 五、验收标准

每次新增前端功能，E2E 测试必须满足：
- ✅ 至少 1 个核心流程测试用例
- ✅ 至少 1 个异常场景测试用例
- ✅ 所有按钮点击、表单提交操作都有对应断言
- ✅ 测试可重复运行，不依赖外部状态
- ✅ CI 环境中可稳定执行，无 flaky 测试
