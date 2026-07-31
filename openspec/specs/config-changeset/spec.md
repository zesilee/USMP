# config-changeset — 变更集服务端契约（试运行预览与批量原子提交）

## Purpose

为攒批提交工作流提供服务端契约：变更集的试运行预览（正向/回滚 NETCONF 报文
与结构化 diff，纯计算不下发）与批量原子提交（单设备跨模块 candidate 两阶段、
失败整体回退、删除条目与叶级删除支持）。

## Requirements

### Requirement: CS-01 试运行预览接口（纯计算，不下发）

后端 SHALL 提供 `POST /api/v1/config/changeset/preview`：入参为单设备变更集（`device` + `entries[]`，条目含 `op(create|update|delete)`、锚点路径、RFC7951 payload、list 主键、清除叶列表），出参 SHALL 含正向 NETCONF edit-config 报文、回滚报文与结构化 diff 树（含 增/改/删 计数）。预览 SHALL 为纯计算：SHALL NOT 向设备发起任何写操作、SHALL NOT 改动 desired（ConfigStore）与缓存、SHALL NOT 写审计。diff 基线 SHALL 取控制器目标态：ConfigStore desired 优先，锚点缺失时回退 running cache，再缺失时实时 `<get>` 回读；响应 SHALL 标注基线来源。

#### Scenario: 预览生成正向报文与 diff

- **WHEN** 变更集含一条 update（改 VLAN 描述）与一条 create（新增接口）
- **THEN** 响应 SHALL 含可下发的正向 edit-config XML 与 diff 树（修改 1 / 增加 1），设备与 desired SHALL 无任何变化

#### Scenario: 预览不产生副作用（负路径防线）

- **WHEN** 对同一变更集连续调用预览两次
- **THEN** 两次结果 SHALL 一致（幂等），期间 SHALL 无 edit-config/commit 报文到达设备、无审计记录新增

#### Scenario: 无效变更集拒绝

- **WHEN** 条目路径无法锚定注册表或 payload 解码失败
- **THEN** SHALL 返回 400 与指明条目的错误信息，SHALL NOT 返回部分结果误导

### Requirement: CS-02 回滚报文反算

预览 SHALL 基于 diff 结果反算回滚报文：ADD 变更 SHALL 映射为条目删除、DELETE 变更 SHALL 映射为按基线值重建、MODIFY 变更 SHALL 以旧值重新编码；正向与回滚报文 SHALL 经同一 XML 编码通道（xmlcodec + driver 注册表）生成。

#### Scenario: 修改类变更的回滚

- **WHEN** 变更集将某叶从 `A` 改为 `B`
- **THEN** 正向报文 SHALL 含值 `B`，回滚报文 SHALL 含值 `A`

#### Scenario: 新增条目的回滚

- **WHEN** 变更集新增一条 list 条目
- **THEN** 回滚报文 SHALL 为该条目的删除报文（`nc:operation="delete"`，仅含主键叶）

### Requirement: CS-03 无 XML 通道模块降级

变更集条目所属模块无 XML 编码通道（driver `Descriptor.XML` 为空，如 system）时，预览 SHALL 如实返回「该模块不支持报文预览」的降级标记，diff 树 SHALL 仍然可用；SHALL NOT 以兜底序列化伪造报文（R08）。

#### Scenario: 混合变更集部分降级

- **WHEN** 变更集同时含 vlan（有 XML 通道）与 system（无）条目
- **THEN** vlan 条目 SHALL 有报文预览，system 条目 SHALL 标注不支持报文预览且 diff 仍展示

### Requirement: CS-04 批量原子提交接口（单设备跨模块，整体回退）

后端 SHALL 提供 `POST /api/v1/config/changeset/commit`：同步将单设备变更集经 candidate 两阶段下发——逐条 edit-config 至 candidate，全部成功才 commit；任一失败 SHALL discard candidate 使设备回到提交前状态（整体回退），SHALL 返回指明失败条目的错误。删除条目 SHALL 编码为删除报文经同一事务提交。desired（ConfigStore）写入与缓存失效 SHALL 在设备 commit 成功之后执行（每条目 merge/删除各自落地），随后 SHALL 逐条目写审计（OA-01 口径）并触发对账；提交失败 SHALL NOT 改动 desired、SHALL NOT 写成功审计。既有意图编排（BIO-03）行为 SHALL 不受本扩展影响。

#### Scenario: 跨模块原子提交成功

- **WHEN** 变更集含 vlan 修改与 ifm 接口新增，提交且设备全部接受
- **THEN** SHALL 单次 commit 生效两模块变更，desired 与缓存 SHALL 更新，SHALL 产生对应审计记录并触发对账

#### Scenario: 中途失败整体回退（负路径）

- **WHEN** 第二条 edit-config 被设备拒绝
- **THEN** SHALL discard candidate，设备 running SHALL 无任何本次变更，desired SHALL 保持提交前状态，响应 SHALL 指明失败条目，SHALL NOT 写成功审计

#### Scenario: 含删除条目的提交

- **WHEN** 变更集含一条 delete 条目（既有 VLAN）
- **THEN** 提交成功后该条目 SHALL 从设备与 desired 中移除，回读列表 SHALL 不再含该条目

#### Scenario: 意图链路零回归

- **WHEN** 业务意图（BIO-03）经既有 2PC 下发
- **THEN** 行为 SHALL 与本变更前一致（Fragment 扩展对既有调用方零影响）

#### Scenario: 归属硬锁一致生效（BR-11 口径）

- **WHEN** 变更集含被业务意图认领路径的条目且未携带 `force=true`
- **THEN** SHALL 返回 409 与认领意图清单、SHALL NOT 向设备下发任何条目；携带 `force=true` SHALL 放行并在审计中留痕

### Requirement: CS-05 叶级删除编码

XML 编码通道 SHALL 支持叶级删除：给定 list 条目定位（主键叶）与待删除叶集合，SHALL 生成「条目定位键 + 目标叶携 `nc:operation="delete"`」的 edit-config 片段，用于字段级清除的提交与预览；编码 SHALL 幂等（同输入同输出）。

#### Scenario: 字段级清除生成叶删除报文

- **WHEN** 变更集 update 条目携清除叶 `description`
- **THEN** 正向报文中该条目 SHALL 含主键叶与 `<description nc:operation="delete"/>`，SHALL NOT 误删条目本身

#### Scenario: 模拟网元端到端删除叶

- **WHEN** 对 netconfsim 提交含清除叶的变更集
- **THEN** 回读 SHALL 显示该叶已不存在且条目其余叶保持不变
