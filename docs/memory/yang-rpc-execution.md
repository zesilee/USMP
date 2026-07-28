---
name: yang-rpc-execution
description: 做/改 YANG rpc(运维操作)相关前必读——四层链路、执行不重试、rpc不入缓存、高危确认、两个follow-up
metadata:
  type: project
---

**YANG rpc 执行能力已交付**（三阶段，2026-07-28 全量合入 main `8b9af2e`）：界面上像配置一样渲染并执行 rpc 运维操作（清接口计数器/重启接口等）。主 spec：`yang-rpc`(RPC-01~05) + device-protocol DP-10 + netconf-simulator NS-09 + frontend FE-19/20。change 归档 `2026-07-28-yang-rpc-execution`。

**四层链路**（加/改 rpc 顺着这条查）：
1. **构建期提取** `backend/tools/rpcgen`（`make gen-rpc`）：goyang 从 YANG 提取 rpc → `internal/yangschema/rpc.gen.go`（149 rpc/17 模块）。ygot 运行期 schema **不含 rpc**，故必须构建期提取。**键=根容器名**（同 schema module key，如 ifm）；另出 `ModuleRPCNamespace`（运行期 schema 也不含 per-module namespace，rpc payload 需要）。
2. **列 rpc** `/yang/schema/:module` 响应加 `rpcs`（RPCSchema：name/label/highRisk/input）；input 复用 FieldDef；leafref→Type=string+LeafRef 携目标。FieldDef 加了 `leafRef` 字段。
3. **执行通道** `client.ExecuteRPC(namespace,rpc,inputs)`（DP-10）：scrapligo `Driver.RPC` 发 `<rpc>`、解析 reply。`POST /rpc/:ip/:module/:rpc`（RPC-03）：API 层校验 mandatory（有 schema），模拟网元不校验（无 schema、只记录+可注入错误 via scenario.ErrorOnRPC）。
4. **前端** `RpcExecuteTab.vue`：`deriveRpcTabs` 把 rpc 派生成 kind='rpc' 的 Tab 与容器平级（name 加 `__rpc__` 前缀防撞）；FieldRenderer 渲 input；ElMessageBox 二次确认。

**三条必须记住的设计**：
- **ExecuteRPC 不自愈重试**（区别于幂等 Get）：rpc 有副作用，传输中断后重发会重复执行（重启两遍），失败即上抛。
- **rpc 不入配置缓存/不触发对账**（§8/D4）：无 desired/actual，是一次性动作。
- **全部 rpc 执行前确认**（非只高危）：rpc 都改设备状态；highRisk（名称含 restart/reboot/power/delete/rollback/upgrade 等破坏性动词，rpcgen 打标）升级 error 警示。reset/clear-counters 温和不判高危。

**两个 follow-up（未做）**：① **leafref 下拉**——if-name 暂用文本框，下拉需接设备实时接口列表数据源（tasks 8.4/9.3）；② **rpc 标签本地化**——Tab 用后端原始名，snd res 已含中文名（如「按接口名清除统计」），待 localizeFields 扩到 rpc 路径。

**踩坑**：docker staging 共享 `usmp-staging` 工程会服**旧前端包**（构建缓存，非代码 bug）——验证前端改动用 `npm run build` + `serve -s dist` 直服，别信 docker staging 的旧包。el-tabs 常驻全部面板，测试选按钮要按 `#pane-<tabname>` 限定（否则 10 个 rpc 面板撞选择器）。详见 [[schema-driven-test-harness]]（同为 YANG 模型驱动链路）、[[backend-ci-flaky-tests]]。
