# design — config-delete-semantics（删除语义模型化）

## Context

写链路现状（2026-07-09 探索实证）：POST 合并语义（desired 按键 union 累积）→ TriggerReconcile → `walkMap` merge/subset diff（desired ⊆ actual 即收敛，**actual 独有键绝不产 DeleteChange**——刻意防误删）→ per-model XML builder → edit-config merge。删除在此架构下没有任何可达路径；`netconf.go` DeleteChange 分支为死代码 stub。前端删除按钮已按 `operationExclude` 门禁渲染但恒禁用。netconfsim 已实现 RFC 6241 §7.2 operation 语义（delete/data-missing）。

## Goals / Non-Goals

**Goals:**
- 行删除端到端可用：DELETE API → desired 键移除 → operation=delete edit-config 同步下发 → 缓存失效/审计/对账收敛 → 前端启用按钮。
- 删除的模型驱动门禁（operation-exclude / readonly）后端兜底。
- 替换 client 死代码 delete stub 为真实现。

**Non-Goals:**
- 不动 `walkMap` merge/subset 语义与 POST 合并语义（防误删支柱）。
- 不做声明式 tombstone（desired 记删除标记 + diff 识别）——爆炸半径大、tombstone 生命周期复杂，且与 merge 语义冲突。
- 不做 `ext:can-be-deleted` 消费（三服务模块零使用）。
- 不做批量删除/级联删除；不做 gNMI 删除路径（现网链路是 NETCONF，gNMI Delete 已有原生支持留待后续）。

## Decisions

### D1 删除走显式命令通道，同步下发（vs 声明式 tombstone）
`DELETE /config/:ip/*path?key=<k>` 收到即构造删除变更**同步**经 `client.Set`（candidate→commit 两阶段，复用 DP-04 管线）下发，成功才返回 200。理由：声明式通道被 merge/subset 语义刻意封死（防误删）；SetConfig 注释亦预留此意图。删除是低频、明确、用户显式确认的操作，命令语义比异步 ACCEPTED 更诚实（失败当场可见）。下发成功后仍 TriggerReconcile 让对账验证收敛（desired 与 actual 均无该键 → 0 change）。

### D2 key 经 query 参数传递，per-model 解析
`?key=10`（vlan：int vlan-id）/ `?key=GigabitEthernet0/0/1`（ifm：string name，URL 编码）。按 path 关键字路由到模型分支（与 BR-06 convertConfig 同构 switch）：解析 key 类型 → 构造「仅含 key 叶」的单条目模型对象供删除编码与 desired 移除。未知路径/非法 key → 400。

### D3 删除编码复用 Change 管线：DeleteChange + OldValue 携带条目
`client.Change{Type: DeleteChange, OldValue: <仅含 key 的模型对象>, NewValue: nil}`；`marshalChange` 的 DeleteChange 分支按 OldValue 类型路由到 per-model 删除 XML builder：外层包裹容器 + 条目元素带 `operation="delete"` + 仅序列化 key 叶（如 `<vlans><vlan xmlns:nc="urn:ietf:params:xml:ns:netconf:base:1.0" nc:operation="delete"><vlan-id>10</vlan-id></vlan></vlans>`）。理由：复用 Set 的连接/candidate/commit/错误透出，不另辟 RPC 路径；stub 死代码原位替换。命名空间前缀按现有 builder 惯例（与 merge XML 同构，仅多 operation 属性）。

### D4 desired 键移除与合并写同临界区
`storeConfigDeleted(cs, ip, path, key)` 复用 `configMergeMu`：Get→按模型分支从 map 删键→Set（构造新对象，不原地改，与 mergeConfig 同规避并发读竞态 R09）。desired 无该键时为幂等 no-op（设备侧仍尝试删除——用户意图以设备为准）。

### D5 门禁：模型驱动、后端兜底
DELETE 前查 `yangschema`：目标 list 节点 `OperationExcludes()` 含 delete、或节点 `ReadOnly()` → 400 + 明确错误（与前端 canDelete/readonly 门禁互为防御，防绕过 UI 直调 API）。schema 查不到该路径时不拦（降级放行，R08——门禁失效不应使合法删除不可用；设备/converter 是最终权威）。

### D6 审计与缓存
成功下发后：`InvalidatePrefix(ip+"|")`（与 POST 同规）+ 审计 Record（Summary 标注删除条目键）。失败（含 data-missing）不失效缓存、不写审计，错误原样透出（§9）。

## Risks / Trade-offs

- [命令语义与既有 ACCEPTED 异步风格不一致] → 有意为之（D1）：删除低频且不可逆，同步失败可见优于异步吞错；spec 里明确两种语义并存。
- [删除后对账若 desired 仍含该键会把条目加回] → D4 先移 desired 再下发，顺序保证；B2 用「删→二轮对账 0 change」断言收敛。
- [per-model switch 又添一处（D2/D3/D4 各一分支）] → 与既有 convertConfig/mergeConfig/XML builder 模式一致；根治属 P5 SND 声明式化，不在本期。
- [ifm key 含 `/`（接口名）经 URL path 会被吃] → 用 query 参数而非 path 段承载 key（D2），仅需标准 URL 编码。
- [并发删同键/删与下发交错] → configMergeMu 串行化 desired 写；设备侧 candidate/commit 由 client 串行化；B1 加 race 用例。

## Migration Plan

纯新增端点 + 死代码替换，无数据迁移；前端按钮从禁用变可用为渐进增强。回滚 = revert PR（按钮回到禁用态）。

## Open Questions

（无）
