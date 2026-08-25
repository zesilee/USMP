# frontend — 前端架构设计（as-built）

> **还原基准**：2026-08-25 配置自审计随笔刷新（原 `main@b1cfbae` Vue3 版全文已过期重写）。行为契约以同目录 `spec.md` 为权威，本文只做架构导航。
> 历史沿革：Vue3+Element-Plus 初代 → React19+antd 重建（PR#316-#337）→ EviewUI 切换 + openinula 运行时（PR#342-#404）。详见 `docs/memory/{react-antd-rebuild,eviewui-switch-implementation}.md`。

## 1. 职责

由 YANG 模型**自动渲染**设备管理界面（R05：禁止手写固定表单）：`/yang/schema` → 派生 → 表单/表格/Tab；编辑 → 变更集 preview/commit 两阶段提交联动后端下发；展示设备/缓存/下发/异常状态。

## 2. 技术栈

React 19 语法 + **openinula 运行时**（`USMP_RUNTIME` 缺省 inula，react 可切换；路由经 `@app-router` 双实现）；组件库 **EviewUI**（`@nce/eview-react`）经 `src/ui` 适配层 + `@ui-backend` 别名单点切换——生产=eview 桥、外网测试/e2e-local=antd 镜像 `src/ui/antd-backend`（EviewUI 实现包不出内网）；Tree/Tabs/Popover 为桥内自绘。状态=自研 store 薄层（`src/stores/createStore.ts`，非 Pinia/Zustand）；HTTP=axios；i18n=自研薄层（见 `ui-i18n` spec）。构建 Vite；测试 Vitest（happy-dom + Browser Mode）+ Playwright。

## 3. 核心渲染管线（模块控制台）

```
GET /yang/schema（模块嵌套呈现 schema + rpc 清单）
  → 派生纯函数（src/utils）：deriveTabs / deriveColumns（默认显示集+可用列全集）/
      deriveKeyField / filterableFields / deriveSchemaTree / deriveDetailTabs
  → src/form 表单核心（schema→控件树、when/must/pattern/range 数据驱动校验、list 增删改）
  → src/ui 适配层控件（FA-01~04：业务代码禁直接 import 组件库，守护测试拦截）
  → views/ModuleConsolePage.tsx（通用模块控制台：列表/详情/编辑/列设置/获取数据源）
```

- 派生结果由**全模块黄金快照**钉住（GD-01，`console-derivation-golden` spec；fixture 来自 `schema-fixture-pipeline`）。
- 左树深到模块级（container+rpc 平铺入树，构建期烘焙双语 children），rpc 入口唯一在左树（`left-tree-navigation` spec）。
- rpc 执行走 RpcExecuteTab（执行前确认+高危升级，不入缓存，`yang-rpc` spec）。

## 4. 数据面

- **读**：`GET /config`（后端 TTL 缓存 30s）；大 list 走服务端分页（阈值 200 双模式）；状态（config=false）只读 Tab 整树走 `<get>` 通道。
- **写**：变更集链路 `hooks/useChangesetSubmit.ts`（preview → commit 2PC，即时下发已退役）；表单态 `hooks/useConfigForm.ts`。
- API 客户端 `src/api`（契约由 `make gen-contract` 从 swag 生成，漂移有门禁）。
- 历史注记：K8s CRD 前端消费链（useK8sCRD/ConfigPage/BUSINESS_CRDS）已于 2026-07（#143 及 native-config-reposition）整链退役删除，链路唯一 = Stack B 直连。

## 5. 测试

分层权威 `frontend/TESTING.md`：F1 纯逻辑（happy-dom）/ F2 组件 / F3 真浏览器（Select 弹层、嵌套 list、自绘 Tree/Tabs 交互；EVIEW_REAL=1 内网打真桥）/ F4 Playwright staging-smoke（`make e2e-local`，pre-push 拦截）。覆盖率棘轮 vitest thresholds。

## 6. 关联

`frontend-runtime`（运行时/双后端开关）、`frontend-ui-adapter`（FA 军规）、`ui-i18n`、`left-tree-navigation`、`console-derivation-golden`、`schema-fixture-pipeline`、`yang-rpc`；后端接口 `yang-api`/`devices-api`/`config-api`/`config-changeset`。技能：`frontend-yang-dynamic-form`。
