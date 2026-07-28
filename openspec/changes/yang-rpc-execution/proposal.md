# yang-rpc-execution

## Why

平台目前只能对 YANG 的**配置树**（container/list/leaf）做增删改查，无法触碰 YANG 的**操作类语句 `rpc`**——如 huawei-ifm 的 `reset-if-counters-by-name`（按接口名清接口计数器）、`restart-if`（重启接口）等 10+ 个运维动作。这类"执行一个操作"的能力（清计数、重启、刷 MIB）在真实运维里高频使用，现在用户只能绕开平台去 CLI 上敲，平台的"统一管理"缺了一大块。

`rpc` 与配置是**不同的交互范式**：配置是"声明期望态→对账收敛"，rpc 是"触发一次带输入参数的操作→拿到执行结果"。前端渲染管线（R05 模型驱动）、i18n（snd res 已含 rpc 及输入叶翻译）、goyang 解析（`Entry.RPC` 已暴露 rpc 输入结构）、NETCONF 执行（scrapligo `Driver.RPC` 支持任意 `<rpc>`）都已就位，缺的只是把这四段接起来。

本能力遵循 R05：**所有模块的所有 rpc 自动渲染**（像配置那样零前端硬编码），新接模块的 rpc 自动入列。

## What Changes

- **构建期 rpc 提取**：新增 codegen 工具从 YANG 源（goyang `Entry.RPC`）提取每模块的 rpc 定义（名称、input 叶及其类型/leafref/mandatory），生成 `.gen.go` 入库（运行镜像零 snd 依赖，与 lefttreegen/tasknamegen 同管线）。ygot 运行期 schema 不含 rpc，故必须构建期提取。
- **后端列 rpc API**：新增按模块列出 rpc 及其 input 字段（复用 `FieldDef`：类型/leafref-options/mandatory/units 等，前端渲染管线零改动即可复用）。rpc 与模块顶层 container 平级呈现（用户指定层级）。
- **NETCONF `<rpc>` 执行通道**：device-protocol 新增 `ExecuteRPC`（scrapligo `Driver.RPC`），把 input 字段编码为 `<rpc>` payload 下发、回读 `<rpc-reply>` 返回结果或 rpc-error。**这是全新协议动作**（区别于 get/edit-config）。
- **模拟网元支持**：netconfsim 新增 custom-rpc 分发——校验 input（mandatory/leafref 存在性）、记录调用、返回 ok/结果，供集成测试端到端验证（T02）。
- **执行 API**：新增 `POST /rpc/:ip/:module/:rpc` 执行端点，返回执行结果/错误。**不进配置缓存**（rpc 非配置态，无 desired/actual 对账语义）。
- **前端 rpc 渲染与执行**：模块控制台内 rpc 与 container 同级呈现（导航层级见用户规格）；点 rpc → 由 input schema 渲染表单（if-name leafref→接口名下拉）→ 校验 mandatory → 执行 → 回显 `<rpc-reply>` 结果或错误。
- **高危 rpc 二次确认**：`restart-if`/`reset-*` 等有真实设备影响的操作，执行前强制二次确认（R08 防误操作）。高危判定策略见 design。
- 首个落地模块 **huawei-ifm**（reset-if-counters-by-name/by-type/all、restart-if 等 10+ rpc），管线泛化后其余模块自动可用。

### 明确不在本次范围

- rpc 的 `output` 复杂结构化渲染——首期只回显 rpc-reply 原文/ok/error，结构化输出表格化留后续。
- rpc 执行审计入 AuditRecord CRD——首期只前端回显，审计留后续。
- 订阅类/长运行 rpc（gNMI 遥测订阅仍为规划能力，R02/§10 不实现）。

## Capabilities

### New Capabilities

- `yang-rpc`: YANG `rpc` 的模型驱动能力——构建期提取 rpc 定义、后端按模块列 rpc（input 复用 FieldDef）、执行 API（触发→结果/错误，不入配置缓存）、高危 rpc 分类。这是本能力的契约核心。

### Modified Capabilities

- `device-protocol`: 新增 Requirement DP-10「NETCONF `<rpc>` 自定义操作执行」——`ExecuteRPC` 把 input 编码为 `<rpc>` 下发、解析 `<rpc-reply>`（结果/`<rpc-error>`），断线重试/超时语义与现有读写一致；get/edit-config 行为不变。
- `netconf-simulator`: 新增 Requirement NS-09「custom rpc 分发与校验」——classifyRPC 识别 custom rpc，校验 input（mandatory/leafref 存在性），记录调用并返回 ok/结果；未识别 rpc 仍返回 ok（NS 既有降级不变）。
- `frontend`: 新增 Requirement FE-19「模型驱动 rpc 渲染与执行」——rpc 与模块 container 同级呈现，input 由 schema 渲染（含 leafref 下拉、mandatory 校验），执行回显 rpc-reply，高危 rpc 二次确认拦截。

## Impact

- **后端新增**：`backend/tools/rpcgen/`（构建期提取，形状对齐 lefttreegen/tasknamegen）；`internal/yangschema/rpc*.gen.go`（生成物）；`internal/api/rpc_handler.go`（列 rpc + 执行）；`pkg/yang-runtime/client/`（ExecuteRPC + netconf.go）；`simulator/netconfsim/`（custom rpc 分发 + 记录）。
- **API 契约**：新增 `GET /yang/rpcs/:module`（或并入 `/yang/schema`）、`POST /rpc/:ip/:module/:rpc`；`make gen-contract` 重生成 api.gen.ts。
- **前端新增**：模块控制台 rpc 区渲染 + 执行表单 + 结果面板 + 高危确认；复用 FieldRenderer/约束引擎。
- **依赖**：零新增第三方依赖（R10）——goyang/scrapligo 均已在用。
- **测试分层**（§5.6）：B1（rpcgen 提取确定性/全 rpc 覆盖；ExecuteRPC 编解码）、**B2 集成**（模拟网元端到端：执行 rpc→sim 校验/记录→结果回读，T02）、B3（handler 列 rpc/执行/mandatory 校验/错误码）、F1/F2（rpc 派生 + 执行表单组件含高危确认）、F3（leafref 下拉真交互）、F4（staging-smoke rpc 渲染+执行冒烟）。
- **分阶段**（DAG，见 design/tasks）：①构建期提取+列 rpc（地基）→②NETCONF 执行通道+模拟网元+执行 API→③前端渲染执行+高危确认。每阶段一 PR（TM04 ≤1000 行）。
- **风险**：高危 rpc 误执行（restart-if 真重启接口）——二次确认 + 高危分类是硬约束；rpc 执行是**有副作用的设备操作**，与只读/对账不同，需谨慎（design 详述爆炸半径与守护）。
