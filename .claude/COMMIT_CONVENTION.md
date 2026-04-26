# 提交信息规范 - 约定式提交 (Conventional Commits)

## 格式规范

```
<type>: <subject>

What: <修改内容概要，一句话说清楚改了什么>
Why: <为什么要改，解决什么问题，背景和动机>
How: <具体实现方式，技术方案说明>
```

---

## Type 类型说明

| 类型 | 说明 | 示例 |
|------|------|------|
| `feat:` | 新增功能 | `feat: 实现 VLAN 动态表单渲染` |
| `fix:` | 修复 bug | `fix: 修复 CORS 端口不匹配导致设备树为空` |
| `docs:` | 文档变更 | `docs: 更新 README 架构图` |
| `test:` | 测试相关 | `test: 添加 VLAN 创建流程 E2E 测试` |
| `refactor:` | 代码重构（不影响功能） | `refactor: 提取 YangField 通用组件` |
| `style:` | 代码样式（不影响代码逻辑） | `style: 统一 TypeScript 格式化` |
| `chore:` | 构建/工具/依赖调整 | `chore: 更新 Playwright 到 v1.44` |
| `perf:` | 性能优化 | `perf: 优化大表格渲染性能` |

---

## Subject 规范

- 使用中文，清晰准确
- 不超过 50 字符
- 结尾不加句号
- 使用祈使句（"修复"、"新增"、"优化"，而非"修复了"、"新增了"）

---

## What / Why / How 三段式说明

### What（改了什么）
- 1-2 句话概括本次提交的所有修改
- 具体到模块、文件、功能点
- **错误：** `修复前端问题`
- **正确：** `修复前端 VLAN 页面设备树加载失败问题，同步更新测试用例的前置条件断言`

### Why（为什么改）
- 说明修改的动机、解决的问题
- 描述问题现象、影响范围、严重程度
- 可以引用 bug ID、需求文档、技术债说明
- **错误：** `修复 CORS 问题`
- **正确：** `手动验收时发现设备树为空，但 E2E 测试全部通过。根因是 CORS 只配置了 5173 端口，而 Vite 实际运行在 3000，导致真实浏览器下 API 请求被拦截，阻塞验收流程。`

### How（具体怎么改的）
- 技术实现细节、方案选择
- 列出关键的文件修改、配置变更
- 说明是否有副作用、需要注意的点
- **错误：** `更新 CORS 配置`
- **正确：** `在 cmd/test-server/main.go 的 CORS AllowOrigins 数组中添加 3000 端口；同步更新 playwright.config.ts baseURL 与实际端口一致；在 vlan.spec.ts beforeEach 中添加设备树加载显式断言。`

---

## 完整示例

### ✅ 正确示例

```
fix: 修复 CORS 端口白名单不包含 3000 导致设备树为空

What: 在后端测试服务器的 CORS 配置中添加 3000 端口支持，同步修正测试框架端口配置，增强测试用例的前置条件验证。
Why: 手动验收时发现前端设备树为空，但 E2E 测试全部通过。根因是 CORS 只允许了 5173 端口，而 Vite 实际运行在 3000，导致真实浏览器环境下 API 请求被拦截，阻塞验收流程。
How: 1. 在 cmd/test-server/main.go 的 CORS AllowOrigins 数组中添加 "http://localhost:3000" 和 "http://127.0.0.1:3000"；2. 修正 playwright.config.ts baseURL 从 5173 改为 3000；3. 在 vlan.spec.ts beforeEach 中添加设备树加载显式断言，确保 CORS 正常工作。
```

### ❌ 错误示例

```
What: 修复 CORS 问题

Why: 前端访问不了 API

How: 改了下配置
```

---

## 特殊场景

### 提交包含多种类型改动
以主要改动的 type 为准，在 What 中说明全部修改：

```
feat: 新增 Interfaces 配置页面

What: 实现 Interfaces YANG 模型驱动配置页面，同时修复了 YangTable 编辑弹窗关闭按钮失效问题，补充 3 个 E2E 测试用例。
...
```

### 破坏性变更 (Breaking Change)
在 type 后加 `!` 标记，并在开头说明：

```
feat!: 重构 Controller Runtime API 接口

BREAKING CHANGE: DeviceClient 接口签名变更，所有调用方需要更新。

What: ...
Why: ...
How: ...
```

---

## 快速检查清单

提交前自问：
- [ ] Type 选对了吗？(feat/fix/docs/test/refactor...)
- [ ] Subject 清晰吗？（看完知道本次提交做什么）
- [ ] What: 具体说明了所有改动吗？
- [ ] Why: 说明了为什么要改吗？
- [ ] How: 说明了具体实现方式吗？

---

## 与项目技能联动

本规范与 `git-what-why-how-commit` 技能配合使用：
- **Type + Subject** 是快速索引
- **What/Why/How** 提供完整上下文，便于追溯和 Code Review
