# translation-engine — 意图翻译引擎

## Purpose

> **LEGACY / 能力已退役。** `pkg/translator` 已于 2026-07-17 整包物理删除（change retire-businessvlan-bridge，PR #186）：唯一调用方 crdsource 旧桥接退役，厂商门禁查询移交 [[device-driver-registry]] DR-04。本 spec 作历史契约保留；全部 requirements 已按 REMOVED delta 摘除。

把业务意图 Spec（当年为 `bizv1.*` CRD Spec）翻译为厂商 ygot YANG 结构，按厂商注册表派发，供 Stack B crdsource 意图源（与遗留 Actor 路径共用）在下发前调用。仅 Huawei 有注册实现；Cisco/H3C/Juniper 为枚举占位。已知契约缺口：Route 翻译返回裸 map（未完成）、System 翻译不支持、接口以 `interface{}` 传参（R04 张力，输入契约解耦属 P5 声明式化后续）。

> 2026-07-09 由「反向还原」自由格式迁移为 OpenSpec 标准格式（spec-format-standard），行为描述忠实 as-built；TE-05/TE-06 由 change `snd-vendor-registry` 引入（其 delta 中编号为 TE-01/TE-02，因与本文件既有编号冲突在 sync 时重编号）。

## Requirements

> TE-01 厂商派发 / TE-02 VLAN 翻译 / TE-03 Interface 翻译 / TE-04 校验 / TE-05 编译期驱动自注册 / TE-06 按设备厂商解析驱动 已于 change retire-businessvlan-bridge 全量 REMOVED：TE-01~04 随调用方与 bizv1 类型退役无迁移目标，TE-05/TE-06 的等价在用契约见 [[device-driver-registry]] DR-01/DR-04；意图→原生配置的现行链路见 [[business-vlan-service]] 与 [[business-intent-orchestration]]。

### Requirement: TE-00 能力保持退役

`pkg/translator` 意图翻译引擎 SHALL 保持物理删除状态，SHALL NOT 重新引入以 `interface{}` 传参的意图→ygot 翻译层；厂商支持性查询 SHALL 一律走 [[device-driver-registry]] DR-04，业务意图展开 SHALL 一律走 internal/intent 原生配置链路。

#### Scenario: 代码库无翻译引擎残留
- **WHEN** 审计代码库
- **THEN** `backend/pkg/translator` SHALL NOT 存在，任何包 SHALL NOT import 该路径
