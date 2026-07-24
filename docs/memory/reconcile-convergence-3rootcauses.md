---
name: reconcile-convergence-3rootcauses
description: 设备配置对账「一直漂移」的三根因结构 + VLAN 同源潜伏 bug 待修
metadata: 
  node_type: memory
  type: project
  originSessionId: bbca4548-3ed5-478b-9e42-e33edd11430e
---

Stack B 设备配置「新建后一直漂移 + 列表看不到」的根因是**三个叠加**（2026-07-06 修复 IFM，见分支 worktree-fix-ifm-interface-drift 两提交）：

- **A diff 引擎**：`diff/diff.go walk()` 原无 `reflect.Map` 分支。ygot 把 YANG list 生成为 `map[key]*Entry`，整表落 `DeepEqual(desired 稀疏, actual 全量)` 永远不等 → 永不收敛。已修：新增 `walkMap`+`subsetMatches`（合并/子集语义，零值=未设跳过，actual-only key 忽略不删，与 `config_handler.storeConfigMerged` 一致）。**此修复对所有模型通用（含 VLAN）**。
- **B edit-config 序列化**：`client/netconf.go marshalChange` 的 map 分支靠 `change.Path` 含 `"ifm:ifm"` 检测，但 diff 产出的 Path 是 Go 字段名 `"Interface"` → 永不命中 → 落畸形 `<list>` XML → 接口从未真正下发。已修：加 `map[string]*HuaweiIfm_Ifm_Interfaces_Interface` 类型断言（镜像 VLAN 的 `map[uint16]*...Vlan`）。**IFM 专属**。
- **C 设备回读解析**：ygot 结构体无 xml tag，`encoding/xml` 无法把 `<interface>` 解析进 Go map → actual 恒空。已修：新增 `client.ParseHuaweiIfmInterfacesXML`（token 流扫描），reconciler 回读改调它。**IFM 专属**。

**VLAN 同源 bug 已一并修复**：VLAN 的 B 本就有（`marshalChange` 有 VLAN map 断言），C 已补 `client.ParseHuaweiVlanVlansXML`（含嵌套 member-port），`vlan/reconciler.go` 回读改调它。IFM+VLAN 现均端到端收敛。这曾解释 [[frontend-landing-risklog]] 里「drifted=已纠正非持久」的观测，现已消除。三提交在分支 worktree-fix-ifm-interface-drift。

放大项（非根因，仍在）：POST 后 `TriggerReconcile` 异步 + 前端 `loadItems` 不传 `force_refresh` + GET 30s 缓存 → 新接口即便下发成功也可能 30s 内不可见。参见 [[vlan-config-stackb]]。

**未修 follow-up（2026-07-08 P3 choice 交付时 B2 新暴露）**：IFM `bandwidth-type` choice 成员 `bandwidth` 端到端**下发成功、设备可读到**（`sim.RunningHuaweiInterfaces()[…].Bandwidth==1000` 通过），但**二轮对账 Changes==1 不收敛**——回读 diff 对 choice 容器字段（bandwidth/damp 等）仍算漂移（疑似 C 类回读解析对 choice 成员字段未覆盖，或 diff 对该字段零值语义）。与 P3 呈现层无关（reconciler/diff 路径未动），故 `TestInterfaceConfig_Integration_ChoiceMemberToDevice` 只断言「落到设备」不断言收敛。根治另开 reconciler PR（先写复现回归 T07）。见 [[yang-constraint-engine]]。
