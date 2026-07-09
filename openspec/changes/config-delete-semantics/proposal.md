# config-delete-semantics — 删除语义模型化（P4）

## Why

通用模块控制台的行「删除」按钮按模型门禁渲染但**恒禁用**（generic-module-console follow-up 债 #1）：后端 `/config` 只有合并语义的 POST，无行删除契约。探索实证（2026-07-09，代码级核对）：

1. 声明式通道**天然删不了东西**——`walkMap` diff 是刻意的 merge/subset 语义（desired ⊆ actual 即收敛，actual 独有键绝不产 DeleteChange，防误删支柱），从 desired 移除条目对设备零效果；
2. `netconf.go:346` 的 delete 分支是死代码残根（裸 `<delete operation="delete"/>`，无层级无键，永远不会被走到）；
3. `SetConfig` 注释早已预留架构意图：「删除走独立 DELETE 端点，不经此路径」；
4. `netconfsim/editconfig.go` 已实现 RFC 6241 §7.2 的 operation 语义（含 delete 与 data-missing）——模拟网元侧零工作量。

## What Changes

- **后端删除通道**：新增 `DELETE /api/v1/config/:ip/*path?key=<条目主键>`——从 desired 移除该键（per-model，加锁与合并写同临界区）→ 构造 per-model 键式删除 edit-config（如 `<vlan operation="delete"><vlan-id>10</vlan-id></vlan>`）经既有 `client.Set`（candidate→commit）**同步下发** → 失效运行缓存 → 审计记录 → 触发对账验证收敛。删除是命令语义（非声明式），设备报 data-missing 等错误如实透出。
- **client 编码**：`marshalChange` 的 DeleteChange 死代码 stub 替换为真实现——按模型构造带 `operation="delete"` 属性、仅含 key 叶的 edit-config XML（vlan/ifm 两模型）。
- **模型驱动门禁**：DELETE 对 list 级 `operation-exclude` 含 delete、或 readonly（config false）路径 SHALL 拒绝（400），与前端门禁互为防御。
- **前端启用行删除**：`canDelete` 门禁已在——启用按钮 + 二次确认（ElMessageBox）→ 调 DELETE API → 成功刷新列表/新鲜度，失败如实展示错误（R08/§9）。
- **不动** `walkMap` merge/subset 收敛语义与 POST 合并语义。
- FieldDef/契约无变化；新增 DELETE 端点进 openapi → `make gen-contract`。

## Capabilities

### New Capabilities

（无 — 全部落在既有能力的需求扩展上）

### Modified Capabilities

- `config-api`：新增 BR-09「行删除通道（DELETE，命令语义）」、BR-10「删除的模型驱动门禁」。
- `device-protocol`：新增 DP-07「NETCONF 删除变更编码（operation=delete 键式条目）」。
- `frontend`：新增 FE-16「列表行删除（confirm→DELETE→刷新）」；FE-11 操作列门禁行为不变（已覆盖）。

## Impact

- **后端**：`internal/api/config_handler.go`（DeleteConfig handler + desired 键移除 + 门禁）、`main.go`（路由）、`pkg/yang-runtime/client/netconf.go`（DeleteChange 编码）+ `changes.go` 备注；不改 diff/reconcile。
- **前端**：`src/api`（deleteConfig）、`components/config/ModuleListTab.vue`（启用按钮/confirm/调用/刷新）、api.gen.ts（再生成）。
- **测试层**（§5.6，涉协议编解码与下发 → B2 强制）：B1（删除 XML 编码 per-model、desired 键移除、门禁判定，表格驱动+race）、**B2 集成**（netconfsim：建→删→回读消失→对账收敛；删不存在→data-missing 透出；并发删）、B3（DELETE 契约：成功/非法 key/门禁拒绝/未知路径）、F2（按钮启用态/confirm 流/错误态）。
- **门禁**：`make gen-contract`；覆盖率棘轮 后端 57.8 / 前端 74/71/67/74。
- 不涉及：数据库（R03）、YANG 模型变更（无需 yang-config-test-design 全矩阵——非新模型接入，但 B2 集成为军规必补）。
