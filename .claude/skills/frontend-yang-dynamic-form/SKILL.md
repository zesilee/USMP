---
name: frontend-yang-dynamic-form
description: YANG 模型驱动的动态表单渲染（R05 载体）：React 19 + src/form 表单核心 + src/ui 适配层，schema 经 /yang/schema 下发、控制台派生自动出 Tab/表格/表单，新增 YANG 模块零手写表单。激活条件：前端开发、YANG 配置页面、设备管理功能、硬编码表单重构。
---

# Frontend YANG Dynamic Form Skill

## 一、激活时机
1. 开发/修改 YANG 配置页面（VLAN、IFM、BGP 等模块控制台）
2. 硬编码表单改为模型驱动（R05：禁止手写固定表单）
3. 后端新增 YANG 模块需要配套前端呈现
4. 关键词：动态表单、YANG 页面、配置管理、模块控制台

不激活：与 YANG 模型无关的业务表单、纯静态页面。

## 二、真实架构（React 19，链路唯一 = Stack B 直连）

```
后端 /yang/schema（模块 schema + rpc 清单）
        ↓
控制台派生层（frontend/src/utils）：deriveTabs / deriveColumns /
  deriveKeyField / filterableFields / deriveSchemaTree
        ↓
src/form 表单核心（schema→控件树、校验、增删改）
        ↓
src/ui 适配层（@ui-backend 别名单点切换：生产=EviewUI 桥，
  外网测试/e2e-local=antd 镜像 src/ui/antd-backend）
        ↓
views/ModuleConsolePage.tsx（通用模块控制台，含变更集提交链路）
```

数据面：读 `GET /config`（TTL 缓存 30s）、写走变更集 preview→commit 2PC（hooks/useChangesetSubmit.ts、useConfigForm.ts）。

## 三、YANG 类型 → 控件映射（经适配层，禁止直引组件库）

| YANG 类型 | 控件 | 校验 |
|-----------|------|------|
| boolean | 开关 | - |
| enumeration | 下拉 | 枚举范围 |
| string | 输入框 | pattern/length（数据驱动，来自 schema 约束） |
| int*/uint* | 数字输入 | range 校验 |
| list | 表格 + 增删改 | 子节点递归渲染、主键必填 |
| container | 分组面板/Tab | 递归 |
| leafref | 关联下拉 | 引用完整性 |
| when/must | 联动显隐/约束 | 数据驱动（YANG 约束引擎） |

## 四、军规（违反即评审不通过）
1. **FA-01~04 适配层军规**：业务代码禁止直接 import 组件库（antd/eview），一律经 `src/ui`（`@ui-backend`），守护测试拦截。
2. **R05**：禁止为单个 YANG 模块手写固定表单；新增模块=后端注册+派生自动渲染，前端零模块专属代码。
3. **删键必解构**（FE-27）：删除 list 行/键时不得原地 delete。
4. **改派生逻辑必刷派生黄金**（GD-01/SF-04）：动 deriveTabs/deriveColumns/deriveKeyField/filterableFields/deriveSchemaTree 后重跑全模块黄金并人工核对，缺黄金/漂移即拦。

## 五、测试要求（按 frontend/TESTING.md 选层）
- 表单核心/派生纯逻辑 → F1（happy-dom，test/{utils,ui}）
- 组件/页面（含 add/**edit/remove**/校验错误态）→ F2（@testing-library/react）
- Select 弹层/teleport/嵌套 list 真实交互 → F3 真浏览器（vitest.browser.config.ts）
- 新页面/路由/端到端流 → F4（frontend/tests/staging-smoke.spec.ts，`make e2e-local`）

## 六、必读背景
碰前端运行时/桥/测试口径前必读 `docs/memory/eviewui-switch-implementation.md` 与 `docs/memory/react-antd-rebuild.md`（三大自绘 Tree/Tabs/Popover、双后端双运行时开关体系、E2E 三根因）。
