---
name: git-what-why-how-commit
description: 约定式提交 + 三段式描述，格式：<type>: <subject> + What/Why/How 正文。支持 feat/fix/docs/test/refactor/chore/perf/style 等标准类型
---

## 一、激活时机（何时自动触发）
1.  当用户需求包含「Commit」「提交日志」「版本更新」等关键词时，自动激活。
2.  开发流程中，代码评审通过后，自动触发本技能，生成标准提交信息。
3.  每次提交仅对应一个原子功能/BUG修复，自动校验代码行数，不符合则拒绝生成。

## 二、核心原则（底层设计逻辑）
1.  **约定式提交前缀**：所有提交**必须**以 `<type>: ` 开头，清晰标识变更类型。
2.  **中文统一原则**：所有提交信息**必须全程使用中文**，保持项目风格统一。
3.  清晰可追溯原则：提交信息需明确说明"做了什么、为什么做、怎么做"。
4.  小步迭代原则：单次提交仅对应一个原子变更，避免多功能合并。
5.  合规性原则：自动校验代码行数，单次代码 < 500 行。

## 三、Type 前缀规范

| 前缀 | 适用场景 |
|------|---------|
| `feat:` | 新增功能、新特性 |
| `fix:` | Bug 修复 |
| `docs:` | 文档变更（README、设计文档、技能定义、记忆等） |
| `test:` | 测试新增、测试修复、测试重构 |
| `refactor:` | 代码重构（不影响功能、非 Bug 修复） |
| `style:` | 代码格式化、样式调整（不影响代码逻辑） |
| `chore:` | 构建工具、依赖升级、CI 配置等 |
| `perf:` | 性能优化 |

## 四、完整格式规范

```
<type>: <subject>

What: <修改内容概要，一句话说清楚改了什么>
Why: <为什么要改，解决什么问题，背景和动机>
How: <具体实现方式，技术方案说明>
```

**强制规则：**
- Subject 不超过 50 字符，中文，结尾不加句号，用祈使句（"修复""新增"，而非"修复了"）
- What/Why/How 每段 1-3 句话；**关键文件、关键改动在 How 中明确列出**
- What 具体到模块/功能点（❌ `修复前端问题`）；Why 讲问题现象与影响（❌ `修复 CORS 问题`）；How 讲方案与副作用（❌ `改了下配置`）

## 五、使用样例

### 样例1：Bug 修复（What/Why/How 的合格粒度）
```
fix: 修复 CORS 端口白名单缺 3000 导致设备树为空

What: 在后端测试服务的 CORS 配置中添加 3000 端口支持，同步修正 Playwright 端口配置，并在冒烟用例中补设备树加载显式断言。
Why: 手动验收发现设备树为空但 E2E 全绿，根因是 CORS 只配了 5173 而 Vite 实际跑在 3000，真实浏览器下 API 被拦截。
How: 后端 CORS AllowOrigins 增加 3000；playwright.config.ts baseURL 对齐实际端口；frontend/tests/staging-smoke.spec.ts 增加设备树加载断言。
```

### 样例2：测试相关
```
test: 新增 VLAN 配置流程模拟网元集成测试

What: 新增 VLAN 创建/修改/删除全流程集成测试与 commit 失败异常场景。
Why: VLAN 是核心配置模块，缺集成防线容易回归；对齐 T02 强制要求。
How: 在 backend/internal/api/ 增加 *_integration_test.go，基于 netconfsim 模拟网元，断言用 testsupport.AssertHuaweiVlan* 系列。
```

## 六、特殊场景

### 提交包含多种类型改动
以主要改动的 type 为准，在 What 中说明全部修改。

### 破坏性变更（Breaking Change）
type 后加 `!` 标记，并在正文开头说明：
```
feat!: 重构 Controller Runtime API 接口

BREAKING CHANGE: DeviceClient 接口签名变更，所有调用方需要更新。

What/Why/How: ...
```

### 记忆与功能同 PR
记忆文件（docs/memory/）可随功能 PR 提交，但**必须单独一个 `docs:` commit**，不得与功能改动混在同一 commit（MEM04）。

## 七、提交前快速检查清单
- [ ] Type 选对了吗？Subject 一眼能看懂做什么？
- [ ] What 覆盖了全部改动？Why 讲清了动机？How 列出了关键文件？
- [ ] 是否只含一个原子变更？行数 ≤500？
- [ ] 结尾带 `Co-Authored-By: Claude Code <noreply@anthropic.com>`
