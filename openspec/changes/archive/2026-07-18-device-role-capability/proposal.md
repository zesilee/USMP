## Why

USMP 要用**统一的一套 YANG** 对接多版本、多网络角色（DCGW/EOR 等）的交换机（snd 融合四期计划②，用户 2026-07-18 拍板）。现状两块缺口：能力协商逻辑 `NarrowModulesByCapabilities` 已实现已测但**未接入任何生产链路**——`/yang/modules` 对所有设备返回同一全量模块集，设备实际不支持的模块也会被当作可配；Device CRD **无网络角色概念**，运维无法在系统里表达"这台是 DCGW、那台是 EOR"。

## What Changes

- Device CRD `spec` 新增 `role` 字段（自由标签 + 常用值 DCGW/EOR/TOR/BORDER；人工指定，只做展示与策略标签，**不参与模块裁剪**——拍板决策）；devices API 注册/列表透传 role；前端设备管理页展示与编辑 role
- **能力协商接入生产链路**：设备连接建立后采集 NETCONF hello capabilities 缓存于连接层；`GET /api/v1/yang/modules?device=<id>` 返回**该设备协商后的模块子集**（`NarrowModulesByCapabilities(hello, loaded)`）；无 `device` 参数保持全量（现行为不变，向后兼容）；设备离线/无缓存能力时降级返回全量并附 `negotiated:false` 标记（R08）
- blacklist 注解透出：`snd/ce6866p-yang/blacklist.xml`（模块+revision 屏蔽清单）构建期解析为映射，命中模块在 `/yang/modules` 项上附 `blacklisted:true` 注解——**仅注解不硬裁剪**（黑名单含 system 等在用模块，硬裁剪会回退已交付功能；语义澄清后另行决策）
- 设备详情能力透出：`GET /api/v1/devices/:id` 附 `role` 与协商摘要（能力条数/是否已协商）

## Capabilities

### New Capabilities

- `device-capability-negotiation`: 每设备 YANG 模块能力协商的采集、缓存、查询契约（hello caps → 模块子集、降级语义、blacklist 注解）

### Modified Capabilities

- `devices-api`: 注册/列表/详情增加 `role` 字段透传与校验
- `yang-api`: `GET /api/v1/yang/modules` 新增可选 `device` 参数的按设备协商语义（BR-12）

## Impact

- 后端：`api/core/v1/device_types.go`（+role，`make gen-crd` 重生成 CRD yaml）、`device_handler.go`、`yang_handler.go`、`pkg/yang-runtime/client`（能力缓存暴露）、新增 blacklist 构建期解析（tools 或 yangschema 内）
- 前端：设备管理页 role 列 + 表单字段（F2 组件测试）；`stores/device.ts` 增 role
- 测试：B1（narrow+blacklist 注解表格驱动+race）、B2（netconfsim 定制 hello caps → `?device=` 端到端）、B3（API 契约）、F1/F2；netconfsim 需支持自定义 hello capabilities（现固定全量）
- spec：新 `device-capability-negotiation`；devices-api/yang-api delta
- 不动：Reconciler 链路、左树（③期）、i18n（④期）
