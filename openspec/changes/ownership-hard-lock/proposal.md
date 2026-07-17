# ownership-hard-lock — 归属硬锁二期：认领路径手改 409 拒绝 + force 逃生

## Why

一期软归属（BIO-07/BR-11，2026-07-16 交付）只做警告不拦截：手改命中业务意图认领路径时照常下发，意图周期收敛再静默覆盖——用户看到的是「改了又被改回」的拉锯，警告文案并不能阻止配置事实上的来回抖动。二期按既定拍板把拉锯升级为**写入时硬拒绝**，并配 force 逃生通道防运维死锁。

## What Changes

- **写入硬锁**：`POST /api/v1/config/:ip/*path` 与 `DELETE`（行删除）命中 `intent.DefaultOwnership` 认领路径时，缺省 SHALL 拒绝——信封码 409（HTTP 恒 200 惯例不变），message 指引「先删/改对应意图或携带 force」，data 附认领意图列表。**BREAKING（行为收紧）**：一期「警告但放行」的默认行为变为拒绝。
- **force 逃生通道**：query `force=true` 时放行下发，响应仍附 `ownershipWarning`（意图收敛仍会覆盖，诚实提示），且审计记录 SHALL 携带 force 标记与被覆盖的认领意图（可追责）。
- **审计扩展**：`audit.Record` 增加可选 `Forced bool` + `ForcedOwners []string` 字段，AuditRecord CRD manifest 同步（可选字段，向后兼容）；`GET /logs` DTO 透出。
- **前端阻断确认流**：下发命中 409 归属拒绝时，弹阻断确认框（列出认领意图 + 「意图收敛会覆盖」提示）；用户确认后携 `force=true` 重发。软警告路径（force 放行后）保留现有 ElMessage.warning。

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `config-api`: BR-11 由「归属警告（软归属，不拦截）」升级为「归属硬锁（缺省 409 拒绝 + force 放行附警告）」。
- `operation-audit`: OA-01 审计记录字段扩展（Forced/ForcedOwners），force 覆盖必须留痕可辨识。
- `frontend`: 下发流程新增归属 409 阻断确认交互（确认后 force 重发）。

## Impact

- **后端**：`internal/api/config_handler.go`（SetConfig/DeleteConfig 入口加门禁 + force 解析 + 审计字段）、`pkg/yang-runtime/audit`（Record 字段 + CRDStore 映射 + memory store 透传）、`deploy/crds/auditrecords.core.usmp.io.yaml`（+forced/forcedOwners 可选属性）。
- **前端**：`useConfigSubmit.ts` 与 `ModuleFormTab.vue` 的下发错误分支识别信封码 409 + data.intents → ElMessageBox 确认 → force 重发；`api` 契约生成物随 swagger 注解再生。
- **测试**：B3（handler 409/force/未认领放行/删除通道）、audit 单测（字段透传）、F1/F2（确认流分支）、契约生成物漂移门禁。
- **不受影响**：意图控制器写路径（intent 自身收敛不走 config-api）、GET 读路径、软归属索引数据面（复用 `Owners()`）。
