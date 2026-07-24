---
name: config-delete-semantics
description: 删除语义 change（DELETE 通道/删除编码/门禁/sim RFC 接线/opMu）关键决策与踩坑；改写链路/删除/netconfsim edit-config 前必读
metadata: 
  node_type: memory
  type: project
  originSessionId: 5becb587-f21e-4cb0-83ae-7eed4c3594be
---

Change `config-delete-semantics`（P4 **已闭环**：#128 实现 + #130 sync/归档，2026-07-09）：行删除端到端。叠在 [[ext-ui-annotations]]（readonly/门禁透出）与 [[reconcile-convergence-3rootcauses]]（merge 收敛）之上。

**核心架构决策（改写链路前必读）：**
- **声明式通道删不了东西是刻意的**：`diff.walkMap`（ygot list=Go map 的真实分支）merge/subset 语义——desired ⊆ actual 即收敛、actual 独有键**绝不**产 DeleteChange（防误删设备存量）。删除走**显式命令通道** `DELETE /config/:ip/*path?key=`（handler 同步下发，非 ACCEPTED 异步），SetConfig 注释早有预留。别试图用 tombstone/全量覆盖改造声明式通道。
- 顺序军规：**先移 desired 再下发**（对账不复活刚删条目）；成功才缓存失效+审计；失败零副作用。
- per-model 分支又添三处（parseDeleteTarget/storeConfigDeleted/marshalDeleteChange），与 convertConfig/mergeConfig 同模式——根治属 P5 SND 声明式化。

**netconfsim 关键变更（写集成测试前必读）：**
- `handleEditConfig` 已从**整树替换**（SetCandidate）接线到 **RFC 6241 §7.2 per-operation**（treeDatastore.EditConfig：merge/create/delete/remove/replace + data-missing 报错）。此前整树替换掩盖了两类 bug：跨模块子树被抹、findMatch list 判定缺陷。
- findMatch 的 keyed-list 判定信号 = 显式 op ∪ 存量多兄弟 ∪ **edit 内多兄弟**（新补）。残留已知限制：单条目 edit 对单不同键存量条目仍会误并（schema-less 歧义，现实推送是 desired 全量 union 不触发）。
- 删除 XML 形态（**PR#145 已修正为嵌套**）：`<ifm xmlns=NS><interfaces><interface nc:operation="delete" xmlns:nc=base><name>…</name></interface></interfaces></ifm>` / `<vlan xmlns=NS><vlans><vlan …><id>10</id></vlan></vlans></vlan>`——sim 按属性本名匹配任意前缀、真机要 nc: 前缀；key 叶须为首子元素；华为 vlan key 是 **id 不是 vlan-id**。

**「内置接口删不掉」根因（PR#145，2026-07-13）——改 xmlcodec 编码前必读**：写链路（Encode/EncodeDelete）此前只发**扁平** `<interfaces>`/`<vlans>` 根，漏了 YANG 模块顶层容器 `<ifm>`/`<vlan>`。真机与 sim 种子 `DemoSeedConfig` 都把 list 容器**嵌套**在模块容器下，扁平删除报文在设备嵌套树里匹配不到既有条目 → 删除落空、还不报错；对账 subset 语义（desired 空 ⊆ actual）又误报「已收敛」掩盖之。**现象：新增接口能删（曾以扁平写入空 candidate 故扁平删能命中）、内置/种子接口删不掉**。整个写路径一直是扁平的、只因非种子 sim 是 mirror-write 才没暴露；golden 冻结的也是扁平（legacy builder 同样错）。修复：`resolved.wrappers()` 按 schema `Entry.Parent` 链派生祖先容器（止于 fake root）包裹，namespace 移到最外层、内层继承；无祖先退回扁平（R08）。**Decode 连带修**：华为模块容器 `<vlan>` 与条目 `<vlan>` **同名**，裸扫条目名会把外层容器误当条目 → 改为 depth 计数在 list 容器（root=vlans/interfaces）内锚定条目。IFM 无同名（ifm≠interface）故未暴露。回归锚点 `TestDeleteConfig_Integration_SeededInterfaceEndToEnd`。

**R09 加固**：NETCONFClient `opMu` 串行化（#128 起）→ **#131（并行会话重投递 #129）增强为全 RPC 串行化 + 死连接自愈**（markDisconnected/ensureConnected；scrapligo v1.4.0 死连接上 Close 必死锁故直接关 Channel；messageID++ 无锁是并发根因）。改客户端并发/重连语义前读 #131。

**踩坑**：**merge 未验证成功前禁止连锁删分支**——`gh pr merge` 可能被分支保护挡下（BEHIND 需先并 main；本仓未开 auto-merge），删了 head 分支 GitHub 会直接关 PR（#130 曾误关，重推+reopen 恢复）；Set 聚合错误（"one or more changes failed"）吞 per-change 的 data-missing——消费方要 per-change 优先；commit-msg 钩子有**单提交 ≤500 行**限制（区别于 pr-size），大改动按原子功能拆提交。

相关：[[ext-ui-annotations]]、[[reconcile-convergence-3rootcauses]]、[[vlan-config-stackb]]
