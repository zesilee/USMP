---
name: yang-rpc-execution
description: 做/改 YANG rpc(运维操作)相关前必读——四层链路、执行不重试、rpc不入缓存、高危确认、两个follow-up已全交付(leafref下拉#232+标签本地化#233)
metadata:
  type: project
---

**YANG rpc 执行能力已交付**（三阶段，2026-07-28 全量合入 main `8b9af2e`）：界面上像配置一样渲染并执行 rpc 运维操作（清接口计数器/重启接口等）。主 spec：`yang-rpc`(RPC-01~05) + device-protocol DP-10 + netconf-simulator NS-09 + frontend FE-19/20。change 归档 `2026-07-28-yang-rpc-execution`。

**四层链路**（加/改 rpc 顺着这条查）：
1. **构建期提取** `backend/tools/rpcgen`（`make gen-rpc`）：goyang 从 YANG 提取 rpc → `internal/yangschema/rpc.gen.go`（149 rpc/17 模块）。ygot 运行期 schema **不含 rpc**，故必须构建期提取。**键=根容器名**（同 schema module key，如 ifm）；另出 `ModuleRPCNamespace`（运行期 schema 也不含 per-module namespace，rpc payload 需要）。
2. **列 rpc** `/yang/schema/:module` 响应加 `rpcs`（RPCSchema：name/label/highRisk/input）；input 复用 FieldDef；leafref→Type=string+LeafRef 携目标。FieldDef 加了 `leafRef` 字段。
3. **执行通道** `client.ExecuteRPC(namespace,rpc,inputs)`（DP-10）：scrapligo `Driver.RPC` 发 `<rpc>`、解析 reply。`POST /rpc/:ip/:module/:rpc`（RPC-03）：API 层校验 mandatory（有 schema），模拟网元不校验（无 schema、只记录+可注入错误 via scenario.ErrorOnRPC）。
4. **前端** `RpcExecuteTab.vue`：FieldRenderer 渲 input；ElMessageBox 二次确认。**2026-07-30 起 rpc 入口已从控制台 Tab 栏迁到左树**（[[left-tree-module-expansion]]）：`deriveRpcTabs` 仍在（rpc 直达页 `/module/:m/rpc/:name` 复用其派生），但不再合入 tabs 栏。

**三条必须记住的设计**：
- **ExecuteRPC 不自愈重试**（区别于幂等 Get）：rpc 有副作用，传输中断后重发会重复执行（重启两遍），失败即上抛。
- **rpc 不入配置缓存/不触发对账**（§8/D4）：无 desired/actual，是一次性动作。
- **全部 rpc 执行前确认**（非只高危）：rpc 都改设备状态；highRisk（名称含 restart/reboot/power/delete/rollback/upgrade 等破坏性动词，rpcgen 打标）升级 error 警示。reset/clear-counters 温和不判高危。

**两个 follow-up 均已交付**：① **leafref 下拉 ✅**（#232 `cc8756b`）——`utils/leafref.ts` parseLeafref + RpcExecuteTab 按 leafRef 经 getConfig+extractRows 拉设备实时接口列表注入 options。**2026-08-06 契约收紧（change rpc-leafref-select-only）：leafref 输入禁自由文本**——离线/拉取失败/空列表不再降级文本框，恒渲染空下拉+「无可选项」占位（R08 降级=空下拉），mandatory 靠 submittable 拦截；FieldRenderer select 条件=`options?.length || leafRef`（LeafRef 唯一产地是 rpcInputField，config 表单叶从不携带，不波及配置表单）；F3 真浏览器用例锁死无 allow-create。② **rpc 标签本地化 ✅**（#233 `be47f0b`，change 归档 `2026-07-29-rpc-label-localization`，主 spec UI-03 扩展）——`localizeRpcs`（`useFieldLabels`）按 rpc 键规律查 snd res 重标 Tab 与 input 叶：**rpc 标签键无根容器段** `/<sourceModule>:<rpcName>`（区别配置叶 `/<sourceModule>:<root>/...`），input 叶 `/<sourceModule>:<rpcName>/input/<叶名>`；**只改 label 不动 name/path**（执行 API 路径与 payload 键不受影响）；缺键/缺文件回退原名。ModuleConsolePage `relabelFields` 并行本地化配置字段+rpc，rootName 守卫防竞态。纯前端消费既有 res，零契约改动。真机验证仍留发布门禁。

**踩坑**：docker staging 共享 `usmp-staging` 工程会服**旧前端包**（构建缓存，非代码 bug）——验证前端改动用 `npm run build` + `serve -s dist` 直服，别信 docker staging 的旧包。el-tabs 常驻全部面板，测试选按钮要按 `#pane-<tabname>` 限定（否则 10 个 rpc 面板撞选择器）。

**#233 新踩坑（worktree 前端交付通用）**：① **常驻 rpc 面板的 leafref 下拉污染页面级 e2e 断言**——el-tabs 保活全部 rpc 面板，每个含 if-name 的 rpc 都把接口名渲成 teleport 下拉隐藏项（实测 6 个），页面级 `getByText('200GE0/1/2').toHaveCount(0)` 被误命中；断言要**限定到 `.el-table__row`** 等真正作用域（本次修了高级搜索冒烟的既有脆弱断言）。② **worktree 里 `node_modules` 别用 symlink**——docker `COPY . .` 无 `.dockerignore` 会把符号链接当文件去替容器内 `npm install` 的目录→build 炸；且 `ln -sf 目标 已存在的symlink` 会在真目录里造嵌套自引用，污染出一个坏 `node_modules`（vitest 127 挂、Vue resolveComponent 警告）。正解：commit 前挂干净 symlink（pre-commit vitest 要 node_modules），**push 前删 symlink**（pre-push docker 建镜像；host playwright 靠 `npx` 自取 runner，不需 host node_modules）。③ **别用 `npx vitest`**——会误取 npx 缓存里那个没装 `@vitejs/plugin-vue` 的 vitest，`.vue` 解析直接报 invalid JS；用 `node ./node_modules/vitest/vitest.mjs run`。详见 [[schema-driven-test-harness]]（同为 YANG 模型驱动链路）、[[frontend-contract-gen]]（playwright 双版本坑）、[[backend-ci-flaky-tests]]。
