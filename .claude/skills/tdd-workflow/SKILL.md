---
name: tdd-workflow
description: 统一的前后端 TDD 工作流，支持 Go 后端单元/集成测试、Vue3 前端组件测试、Playwright E2E 测试、NETCONF 模拟器集成测试，覆盖率 80%+
---

# 统一前后端 TDD 测试驱动开发工作流

针对 USMP 交换机设备管理平台的标准化 TDD 流程，同时覆盖后端 Go 代码和前端 Vue3 代码的全链路测试验证。

## 一、激活时机（自动触发）

### 后端开发场景
1. 新增 YANG 模块 Controller/Reconciler 业务功能
2. 修改或重构缓存、NETCONF 客户端、Manager 等核心组件
3. 新增 API 接口或修改接口行为
4. BUG 修复前先补充测试用例

### 前端开发场景
1. 新增或修改 YangRenderer、YangTable、YangField 等核心组件
2. 新增 YANG 模块动态表单页面（VLAN/Interfaces/System 等）
3. 修改页面交互流程（表单提交、弹窗、导航）
4. 前端状态管理或工具函数修改

### 通用场景
- 任何功能开发前必先编写测试
- PR/MR 提交前必须验证所有测试通过

---

## 二、TDD 工作流总览

```
┌─────────────────────────────────────────────────────────┐
│           需求分析与测试策略制定                          │
│   ├─ 确定测试范围（单元/集成/E2E）                        │
│   └─ 识别必须覆盖的场景（正常/异常/边界/并发）            │
└─────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────┐
│           第一步：编写失败测试（RED）                     │
│   ├─ 后端：编写 *_test.go，使用表格驱动测试               │
│   ├─ 前端：编写 *.spec.ts，使用 Vitest + Vue Test Utils  │
│   └─ 运行测试，确认 FAIL（未实现代码前测试应该失败）      │
└─────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────┐
│           第二步：实现最少代码（GREEN）                   │
│   ├─ 只写刚好让测试通过的代码，不做额外优化               │
│   └─ 保持代码简洁，先求正确再求完美                       │
└─────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────┐
│           第三步：重构代码（REFACTOR）                    │
│   ├─ 在测试保护下优化代码结构                             │
│   ├─ 消除重复、改进命名、提取公共函数                     │
│   └─ 每次重构后立即运行测试，确保仍然 GREEN               │
└─────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────┐
│           第四步：添加集成测试（后端特有）                 │
│   ├─ 启动 NETCONF 模拟器                                  │
│   ├─ 编写 *_integration_test.go                          │
│   └─ 验证端到端配置下发流程                               │
└─────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────┐
│           第五步：添加 E2E 测试（前端特有）                │
│   ├─ 编写 Playwright 测试脚本                             │
│   ├─ 使用 Page Object Model 模式                          │
│   └─ 覆盖真实用户操作流程                                 │
└─────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────┐
│           第六步：覆盖率验证与提交                         │
│   ├─ 后端：go test -cover ./...  目标 80%+               │
│   ├─ 前端：npm run test:coverage  目标 80%+              │
│   └─ 所有测试通过后提交代码                               │
└─────────────────────────────────────────────────────────┘
```

---

## 三、后端 TDD 规范（Go）

### 3.1 测试类型与位置

| 测试类型 | 文件命名 | 位置 | 说明 |
|---------|---------|------|------|
| 单元测试 | `*_test.go` | 与被测代码同目录 | 单个函数/方法，Mock 外部依赖 |
| 集成测试 | `*_integration_test.go` | 与被测代码同目录 | 真实 NETCONF 模拟器，端到端 |
| 基准测试 | `*_bench_test.go` | 与被测代码同目录 | 性能关键代码路径 |

### 3.2 单元测试规范（表格驱动）

**标准模板：**
```go
package yourpackage

import (
    "testing"
    "context"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestYourFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    YourInputType
        want     YourOutputType
        wantErr  bool
        setup    func(*MockDependency) // 可选，设置 Mock 行为
    }{
        {
            name: "正常场景 - 有效输入返回成功",
            input: YourInputType{Field: "valid"},
            want: YourOutputType{Success: true},
            wantErr: false,
            setup: func(m *MockDependency) {
                m.On("SomeMethod", "valid").Return(nil)
            },
        },
        {
            name: "异常场景 - 无效输入返回错误",
            input: YourInputType{Field: ""},
            wantErr: true,
        },
        {
            name: "边界场景 - 输入为零值",
            input: YourInputType{},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange - 准备
            mockDep := NewMockDependency()
            if tt.setup != nil {
                tt.setup(mockDep)
            }
            
            service := NewService(mockDep)

            // Act - 执行
            got, err := service.YourFunction(context.Background(), tt.input)

            // Assert - 断言
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            assert.NoError(t, err)
            assert.Equal(t, tt.want, got)
            mockDep.AssertExpectations(t)
        })
    }
}
```

### 3.3 并发测试规范

```go
func TestCache_ConcurrentAccess(t *testing.T) {
    cache := NewCache(100, 30*time.Second)
    const goroutines = 100
    const operations = 1000

    var wg sync.WaitGroup
    wg.Add(goroutines)

    for i := 0; i < goroutines; i++ {
        go func(id int) {
            defer wg.Done()
            key := fmt.Sprintf("key-%d", id%10)
            
            for j := 0; j < operations; j++ {
                cache.Set(key, fmt.Sprintf("value-%d", j))
                _, _ = cache.Get(key)
                if j%100 == 0 {
                    cache.Invalidate(key)
                }
            }
        }(i)
    }

    // 设置超时防止死锁
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        // 成功完成
    case <-time.After(10 * time.Second):
        t.Fatal("并发测试超时，可能存在死锁")
    }
}
```

### 3.4 NETCONF 集成测试规范

**强制要求：**
1. 必须使用 `if testing.Short() { t.Skip() }` 跳过短模式
2. 必须启动真实的 NETCONF 模拟器
3. 必须验证模拟设备上的最终配置状态
4. 必须覆盖至少一个异常场景

**标准模板：**
```go
package yourcontroller

import (
    "testing"
    "context"
    "github.com/stretchr/testify/assert"
    "github.com/leezesi/usmp/test/netconf-simulator"
)

func TestReconciler_Integration_CreateConfig(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }

    // 1. 启动 NETCONF 模拟服务器
    sim := netsim.NewSimulator()
    err := sim.Start()
    assert.NoError(t, err)
    defer sim.Stop()

    // 2. 初始化依赖（真实客户端，非 Mock）
    pool := client.NewPool()
    err = pool.Add(client.DeviceConnectionInfo{
        IP:       sim.Addr(),
        Port:     sim.Port(),
        Username: sim.Username(),
        Password: sim.Password(),
        Protocol: client.ProtocolNETCONF,
    })
    assert.NoError(t, err)

    // 3. 准备测试数据
    desired := &openconfig.Device{
        Vlans: &openconfig.Vlans{
            Vlan: []*openconfig.Vlans_Vlan{{
                VlanId: 100,
                Config: &openconfig.Vlans_Vlan_Config{
                    VlanId: 100,
                    Name:   "Test_VLAN",
                },
            }},
        },
    }

    // 4. 执行被测逻辑
    r := NewReconciler(pool)
    result, err := r.Reconcile(context.Background(), reconcile.Request{
        DeviceID: "test-device",
        Desired:  desired,
    })
    assert.NoError(t, err)
    assert.False(t, result.NeedRequeue)

    // 5. 验证模拟设备上的最终状态（关键！）
    sim.AssertVlanExists(t, 100)
    sim.AssertVlanName(t, 100, "Test_VLAN")
}

func TestReconciler_Integration_CommitError(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }

    sim := netsim.NewSimulator()
    err := sim.Start()
    assert.NoError(t, err)
    defer sim.Stop()

    // 设置错误场景
    sc := netsim.NewScenarioConfig()
    sc.ErrorOnRPC = map[string]error{
        "commit": fmt.Errorf("commit failed: device busy"),
    }
    sim.SetScenario(sc)

    // ... 执行并验证错误处理逻辑
}
```

### 3.5 后端测试命令

```bash
# 仅运行单元测试（快速）
go test -short ./...

# 运行所有测试（包括集成测试）
go test ./...

# 运行并显示详细输出
go test -v ./...

# 运行特定测试
go test -run TestReconciler ./...

# 运行集成测试
go test -run Integration ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 带竞态检测
go test -race ./...

# 基准测试
go test -bench=. -benchmem ./...
```

---

## 四、前端 TDD 规范（Vue3 + TypeScript）

### 4.1 测试类型与位置

| 测试类型 | 文件命名 | 位置 | 说明 |
|---------|---------|------|------|
| 组件单元测试 | `*.spec.ts` | `src/components/__tests__/` | Vue 组件渲染、交互 |
| 工具函数测试 | `*.spec.ts` | 与被测文件同目录 | composables、utils |
| E2E 测试 | `*.spec.ts` | `tests/` | Playwright 端到端 |
| Page Object | `*.ts` | `tests/pages/` | 页面对象抽象 |

### 4.2 组件单元测试规范（Vitest + Vue Test Utils）

**标准模板：**
```typescript
import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import YangTable from '../YangTable.vue'
import type { YangNode } from '../../types/yang-schema'

describe('YangTable 组件', () => {
  // 准备测试数据
  const testSchema: YangNode = {
    path: '/vlans/vlan',
    name: 'vlan',
    type: 'list',
    config: true,
    children: [
      { path: 'vlan-id', name: 'vlanId', type: 'uint16', config: true },
      { path: 'name', name: 'name', type: 'string', config: true },
    ],
  }

  const testData = [
    { vlanId: 100, name: 'VLAN_100' },
    { vlanId: 200, name: 'VLAN_200' },
  ]

  it('正常场景 - 应该正确渲染表格数据', () => {
    const wrapper = mount(YangTable, {
      props: { node: testSchema, modelValue: testData }
    })

    // 断言表格行数
    const rows = wrapper.findAll('tbody tr')
    expect(rows.length).toBe(2)
    
    // 断言单元格内容
    expect(rows[0].text()).toContain('100')
    expect(rows[0].text()).toContain('VLAN_100')
  })

  it('边界场景 - 空数据应该显示空状态提示', () => {
    const wrapper = mount(YangTable, {
      props: { node: testSchema, modelValue: [] }
    })

    expect(wrapper.find('.el-table__empty-text').exists()).toBe(true)
  })

  it('交互场景 - 点击新增按钮应该触发新增事件', async () => {
    const wrapper = mount(YangTable, {
      props: { node: testSchema, modelValue: testData, editable: true }
    })

    await wrapper.find('[data-testid="btn-add-row"]').trigger('click')
    
    // 断言弹窗打开
    expect(wrapper.find('.el-dialog').exists()).toBe(true)
  })

  it('异常场景 - Schema 缺失 children 应该优雅降级', () => {
    const badSchema: YangNode = {
      path: '/test',
      name: 'test',
      type: 'list',
      config: true,
      // 故意缺少 children
    }

    const wrapper = mount(YangTable, {
      props: { node: badSchema, modelValue: [] }
    })

    // 不应该崩溃，应该有错误提示
    expect(wrapper.find('[data-testid="schema-error"]').exists()).toBe(true)
  })
})
```

### 4.3 Composable 测试规范

```typescript
import { describe, it, expect, vi } from 'vitest'
import { useYangForm } from '../composables/useYangForm'
import type { YangNode } from '../types/yang-schema'

describe('useYangForm Composable', () => {
  const testSchema: YangNode = {
    path: '/vlans',
    name: 'vlans',
    type: 'container',
    config: true,
    children: [],
  }

  it('应该初始化正确的表单状态', () => {
    const { formData, errors, isSubmitting } = useYangForm(testSchema)
    
    expect(isSubmitting.value).toBe(false)
    expect(Object.keys(errors.value).length).toBe(0)
  })

  it('提交失败应该设置错误状态', async () => {
    vi.mock('../api/config', () => ({
      submitConfig: vi.fn().mockRejectedValue(new Error('Network Error'))
    }))

    const { submit, errors, isSubmitting } = useYangForm(testSchema)
    
    await submit()
    
    expect(isSubmitting.value).toBe(false)
    expect(errors.value.submit).toBeDefined()
  })
})
```

### 4.4 E2E 测试规范（Playwright）

**强制要求：**
1. 使用 Page Object Model 模式
2. 使用 `data-testid` 定位元素
3. 每个测试独立，不依赖其他测试
4. 覆盖正常流程和至少一个异常场景

**标准模板：**
```typescript
// tests/pages/VlanPage.ts
import { type Page, expect } from '@playwright/test'

export class VlanPage {
  constructor(readonly page: Page) {}

  async goto() {
    await this.page.goto('/vlans')
  }

  async clickAddVlan() {
    await this.page.getByTestId('btn-add-vlan').click()
  }

  async fillVlanForm(vlanId: number, name: string) {
    await this.page.getByLabel('VLAN ID').fill(String(vlanId))
    await this.page.getByLabel('名称').fill(name)
  }

  async submitForm() {
    await this.page.getByRole('button', { name: '确定' }).click()
  }

  async getRowByVlanId(vlanId: number) {
    return this.page.getByRole('row', { name: String(vlanId) })
  }

  async getSuccessToast() {
    return this.page.getByText('配置下发成功')
  }

  async getErrorToast() {
    return this.page.getByText('配置下发失败')
  }
}

// tests/vlan.spec.ts
import { test, expect } from '@playwright/test'
import { VlanPage } from './pages/VlanPage'

test.describe('VLAN 管理页面', () => {
  let vlanPage: VlanPage

  test.beforeEach(async ({ page }) => {
    vlanPage = new VlanPage(page)
    await vlanPage.goto()
  })

  test('正常流程 - 新建 VLAN 成功', async () => {
    // Arrange
    await vlanPage.clickAddVlan()
    
    // Act
    await vlanPage.fillVlanForm(100, 'Test_VLAN')
    await vlanPage.submitForm()
    
    // Assert
    await expect(vlanPage.getSuccessToast()).toBeVisible()
    await expect(vlanPage.getRowByVlanId(100)).toBeVisible()
  })

  test('异常场景 - VLAN ID 超出范围应该提示错误', async () => {
    await vlanPage.clickAddVlan()
    await vlanPage.fillVlanForm(9999, 'Invalid_VLAN')  // VLAN ID 最大 4094
    await vlanPage.submitForm()
    
    // 应该显示表单验证错误
    await expect(vlanPage.page.getByText('VLAN ID 必须在 1-4094 之间')).toBeVisible()
  })

  test('边界场景 - 删除 VLAN 后表格应该更新', async () => {
    // 先创建一个 VLAN
    await vlanPage.clickAddVlan()
    await vlanPage.fillVlanForm(200, 'To_Delete')
    await vlanPage.submitForm()
    await expect(vlanPage.getSuccessToast()).toBeVisible()

    // 删除
    const row = await vlanPage.getRowByVlanId(200)
    await row.getByTestId('btn-delete').click()
    await vlanPage.page.getByRole('button', { name: '确认删除' }).click()

    // 验证已删除
    await expect(vlanPage.getRowByVlanId(200)).not.toBeVisible()
  })
})
```

### 4.5 前端测试命令

```bash
# 运行单元测试（监听模式，开发时使用）
npm run test:unit

# 运行单元测试（单次，CI 使用）
npm run test:unit:run

# 生成覆盖率报告
npm run test:unit:coverage

# 运行 E2E 测试（需要后端和模拟器运行）
npm run test:e2e

# E2E 测试（UI 模式）
npm run test:e2e:ui

# 运行所有测试
npm run test:all
```

---

## 五、全链路测试集成（前后端联动）

### 5.1 开发环境测试流程

```bash
# 终端 1：启动 NETCONF 模拟器
cd backend
go run test/netconf-simulator/cmd/main.go

# 终端 2：启动后端服务
go run cmd/server/main.go

# 终端 3：启动前端开发服务器
cd ../frontend
npm run dev

# 终端 4：运行 E2E 测试
npm run test:e2e
```

### 5.2 CI 流水线集成

```yaml
# .github/workflows/test.yml
name: Test Pipeline

on: [push, pull_request]

jobs:
  backend-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      
      - name: Run unit tests
        run: |
          cd backend
          go test -short -race -coverprofile=coverage.out ./...
      
      - name: Check coverage
        run: |
          go tool cover -func=coverage.out | grep total | \
          awk -F'%' '{if ($1 < 80) exit 1}'

      - name: Run integration tests
        run: |
          cd backend
          go test -run Integration ./...

  frontend-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
      
      - name: Install dependencies
        run: |
          cd frontend
          npm ci
      
      - name: Run unit tests
        run: |
          cd frontend
          npm run test:unit:run -- --coverage

  e2e-test:
    runs-on: ubuntu-latest
    needs: [backend-test, frontend-test]
    steps:
      - uses: actions/checkout@v4
      
      - name: Start backend + simulator
        run: |
          cd backend
          go run test/netconf-simulator/cmd/main.go &
          go run cmd/server/main.go &
          sleep 5  # 等待服务启动
      
      - name: Start frontend
        run: |
          cd frontend
          npm run build
          npm run preview &
          sleep 3
      
      - name: Run Playwright tests
        run: |
          cd frontend
          npm run test:e2e
```

---

## 六、覆盖率要求与验收标准

### 6.1 强制覆盖率目标

| 代码类型 | 后端覆盖率 | 前端覆盖率 | 说明 |
|---------|-----------|-----------|------|
| 核心业务逻辑（Reconciler） | 90%+ | - | 配置对齐、下发、重试 |
| 核心组件（Cache/Client） | 90%+ | - | 缓存、NETCONF 客户端 |
| API Handler | 85%+ | - | 所有 HTTP 接口 |
| Yang 核心组件 | - | 85%+ | YangRenderer/Table/Field |
| 工具函数 / Composables | 80%+ | 80%+ | 可复用逻辑 |
| 生成代码（ygot） | 排除 | 排除 | 自动生成的代码 |

### 6.2 测试覆盖率计算

**后端：**
```bash
cd backend
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# 查看特定包覆盖率
go test -coverprofile=coverage.out ./internal/controller/...
```

**前端：**
```bash
cd frontend
npm run test:unit:coverage -- --reporter=text
```

### 6.3 提交代码前必须完成的检查清单

✅ 后端单元测试全部通过：`go test -short ./...`
✅ 后端集成测试全部通过：`go test -run Integration ./...`
✅ 后端覆盖率达标：`go test -cover ./...`
✅ 前端单元测试全部通过：`npm run test:unit:run`
✅ 前端 E2E 测试（核心流程）通过
✅ 无 flaky 测试（重复运行 3 次都通过）
✅ 代码符合项目规范（lint 通过）

---

## 七、最佳实践与反模式

### ✅ 推荐做法

1. **测试行为，而非实现**
   - 好：测试 `Get(key)` 返回正确的值
   - 坏：测试内部 `cache.map` 包含某个 key

2. **每个测试独立**
   - 每个测试都应该能单独运行
   - 不依赖其他测试的副作用

3. **有意义的测试名称**
   - 好：`TestReconciler_WhenDeviceOffline_ShouldRetry`
   - 坏：`TestReconciler2`

4. **清晰的 Arrange-Act-Assert 结构**
   - 每个测试都应该有明确的三段式结构

5. **避免过度 Mock**
   - 单元测试可以 Mock 外部依赖
   - 集成测试应该使用真实组件（如模拟器）

### ❌ 反模式（必须避免）

1. **"测试神"** - 一个测试覆盖太多场景
2. **脆弱测试** - 依赖 CSS 类名、DOM 层级、特定实现
3. **Magic Sleep** - 使用 `time.Sleep()` 等待异步操作
4. **只测试快乐路径** - 忽视异常和边界场景
5. **测试私有方法** - 应该通过公共 API 测试
6. **"为了覆盖率而写测试"** - 质量比数量重要

---

## 八、测试失败排查指南

### 后端测试失败

1. **竞态问题**：运行 `go test -race` 检测数据竞争
2. **Mock 不匹配**：检查 mock 调用参数与实际是否一致
3. **模拟器端口冲突**：确保模拟器使用的端口未被占用
4. **超时**：集成测试可能需要更长超时时间，使用 `-timeout 60s`

### 前端测试失败

1. **组件未挂载完成**：使用 `await wrapper.vm.$nextTick()`
2. **异步操作未完成**：使用 `findBy*` 而不是 `getBy*`
3. **E2E 元素不可见**：添加 `{ timeout: 5000 }` 等待元素出现
4. **状态不同步**：确保测试间正确清理状态

---

**记住**：测试不是负担，而是安全网。它让你有信心重构、快速迭代、并确保生产环境的可靠性。
