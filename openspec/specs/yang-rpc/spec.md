# yang-rpc Specification

## Purpose

为已接入模块的 YANG rpc（清计数、重启接口等运维操作）提供完整执行链路：构建期 rpcgen 从 YANG 源提取 rpc 定义与 input 叶树 → `/yang/schema` 随模块下发 → 后端 ExecuteRPC 执行（有副作用：不重试、结果不入缓存）→ 前端 rpc 执行入口（左树直达，执行前确认、高危操作升级确认）。leafref input 提供关联下拉，标签本地化只改展示不改键。
## Requirements
### Requirement: RPC-01 构建期 rpc 提取

系统 SHALL 在构建期从 YANG 源（goyang `Entry.RPC`）提取每个已接入模块的 rpc 定义，产出版本化生成物入库（运行镜像零 snd 依赖，与 lefttreegen/tasknamegen 同管线）。提取内容 SHALL 含：rpc 名称、input 叶树（叶名/类型/leafref 目标路径/mandatory/units/range/pattern）。模块集合 SHALL 动态发现，SHALL NOT 硬编码 rpc 名单——新接模块的 rpc 自动纳入。

运行期 ygot schema 不含 rpc，故提取 SHALL 为构建期完成，SHALL NOT 依赖运行期 snd 文件。

#### Scenario: 提取模块全部 rpc

- **WHEN** 对含 rpc 的模块（如 huawei-ifm）执行提取
- **THEN** 系统 SHALL 产出该模块**全部** rpc 的定义，每个含名称与 input 叶树
- **AND** input 叶的 leafref 目标路径、mandatory 标记 SHALL 被保留

#### Scenario: 新增模块 rpc 自动纳入

- **WHEN** 新接入一个含 rpc 的模块，提取工具代码未改
- **THEN** 重新提取 SHALL 自动产出该模块的 rpc 定义

#### Scenario: 提取生成物零漂移门禁

- **WHEN** YANG 源或提取逻辑变更但生成物未重新生成
- **THEN** CI regen-and-diff SHALL 失败并指出差异

### Requirement: RPC-02 按模块列出 rpc

系统 SHALL 通过 API 按模块列出其 rpc 及每个 rpc 的 input 字段。input 字段 SHALL 以与配置字段同构的 FieldDef 形态透出（type/options/required/leafref/units 等），使前端渲染管线可零改动复用。rpc SHALL 与模块顶层配置节点平级呈现。

#### Scenario: 列出模块 rpc 与 input

- **WHEN** 客户端请求某模块（huawei-ifm）的 schema
- **THEN** 响应 SHALL 含该模块 rpc 列表，每个 rpc 含名称与 input 的 FieldDef 树
- **AND** `reset-if-counters-by-name` 的 input `if-name` SHALL 标记 mandatory 且携带 leafref 目标（接口名）

#### Scenario: 无 rpc 模块

- **WHEN** 请求一个不含 rpc 的模块 schema
- **THEN** 响应 SHALL 正常返回，rpc 列表为空（不报错，R08）

### Requirement: RPC-03 rpc 执行

系统 SHALL 提供执行端点，接受目标设备、模块、rpc 名与 input 值，触发设备侧 rpc 执行并返回结果或错误。执行 SHALL NOT 写入 TTL+LRU 配置缓存、SHALL NOT 触发对账（rpc 非配置态，无 desired/actual 语义）。缺失 mandatory input 时 SHALL 拒绝执行并返回明确错误。

#### Scenario: 成功执行

- **WHEN** 以合法 input（if-name = 存在的接口）执行 `reset-if-counters-by-name`
- **THEN** 系统 SHALL 向设备下发对应 `<rpc>`，并返回执行结果（ok 或 rpc-reply 数据）
- **AND** 配置缓存 SHALL 不被写入或失效

#### Scenario: 缺失 mandatory input 被拒

- **WHEN** 执行 rpc 但缺失 mandatory input
- **THEN** 系统 SHALL 拒绝执行并返回校验错误，SHALL NOT 向设备下发

#### Scenario: 设备返回 rpc-error

- **WHEN** 设备对 rpc 返回 `<rpc-error>`（如目标接口不存在）
- **THEN** 系统 SHALL 把错误明确回传给调用方，SHALL NOT 崩溃（R08）

### Requirement: RPC-04 高危 rpc 分类

系统 SHALL 标记有重大设备影响的高危 rpc（如 `restart-if` 重启接口）。高危标记 SHALL 随 rpc 定义在构建期产出，供前端据以升级确认警示。

#### Scenario: 高危 rpc 被标记

- **WHEN** 提取含 `restart-if` 的模块
- **THEN** `restart-if` 的 rpc 定义 SHALL 带 highRisk 标记
- **AND** 无害操作型 rpc 的标记 SHALL 与高危区分

### Requirement: RPC-05 厂商与协议边界

rpc 执行 SHALL 走 NETCONF `<rpc>`（R02），SHALL NOT 引入旧协议。rpc 能力 SHALL 仅对已接入的 huawei（及 usmp business）模块开放，与既有厂商边界（BR-11）一致。

#### Scenario: 仅 NETCONF 通道

- **WHEN** 执行任一 rpc
- **THEN** 系统 SHALL 经 NETCONF `<rpc>` 下发，SHALL NOT 使用 Telnet/SNMP 等旧协议

