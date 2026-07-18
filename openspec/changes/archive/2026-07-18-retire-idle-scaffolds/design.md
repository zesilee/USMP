# retire-idle-scaffolds — 设计

## Context

审计（2026-07-18，main@4f84f97）：plugin 包零调用方（Manager 构造持有但 reconcile 全流程不触钩子）；gNMI client 经 pool `ProtocolGNMI`/`AUTO`+9339 可达但 Get/Set 发空请求（spec 注记空壳）；gnmi_sub 订阅源零生产消费。`openconfig/gnmi` 为 go.mod 直依赖。FileSource（97 行）为通用能力有测试，保留。

## Goals / Non-Goals

**Goals:** 空转载体清零；gNMI 假成功路径变显式错误；框架 Manager 表面积收口。

**Non-Goals:** 不放弃 gNMI 规划身份（§1 标注「规划」，Device CRD protocol 枚举保留 `gnmi`）；不动 FileSource/CRD source/周期源。

## Decisions

### D1 gNMI 处置=删空壳留显式错误，而非补真实现或全面剔除

补真实现需 gNMI 模拟网元（独立大工程，无当期需求）；全面剔除（枚举/CRD 值/文档）伤平台规划身份且 CRD 枚举收缩是存量 CR 兼容雷。空壳→显式错误是最小诚实态：`factory` 对 GNMI/AUTO-9339 返回 "gNMI 尚未实现（规划能力）"，设备探活如实离线（R08：明确错误优于伪装成功）。

### D2 plugin 整包删除而非留接口

「预留扩展点」是这笔债的成因——接口留着就还是空转。真需要钩子时按当时需求重新设计（届时有真实调用方驱动契约）。Manager 接口删两方法属框架内部 API（零外部消费者，audit 实证）。

### D3 单实现 PR

删除 ~800 行 + 少量改动，>20 文件走 3000 档或纯删除档；TDD：factory 错误路径与 Manager 收口先写测试。

## Risks / Trade-offs

- **[风险] 存量 Device CR 带 protocol=gnmi/9339 端口** → 行为从假成功变显式错误+探活离线——这是修正不是回归；无现网。
- **[取舍] plugin 重来时无脚手架** → 由真实需求重新设计，代价可接受。

## Migration Plan

零迁移；回滚=revert。

## Open Questions

（无）
