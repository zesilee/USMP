# device-driver-registry — delta（full-yang-onboarding）

## ADDED Requirements

### Requirement: DR-06 表驱动 plain-container 注册与根名路径约定

对「单容器根、走通用 XML 引擎」的模块（plain-container 形态），注册 SHALL 表驱动：
每模块一条数据行（模块名/根容器名/namespace/GoStruct 构造子），由统一注册循环派生
全部谓词与锚点，SHALL NOT 逐模块手写描述符代码块。运行时配置路径 SHALL 以
**根容器名**为前缀段（`/<root>:<root>/...`，与前端 `configPathFor` 派生口径一致），
描述符谓词与 `EncodeAnchor` SHALL 锚定该口径；SHALL NOT 使用 YANG prefix 作为
路由 token（YANG prefix 仅属 XML 编解码的 namespace 层）。表行 namespace SHALL
显式登记（内嵌 schema 的 `Entry.Namespace()` 为空、不可派生）且全表唯一。

#### Scenario: 表行注册可路由可编解码

- **WHEN** 表中登记模块 `ntp`（根 `ntp`，namespace `urn:huawei:yang:huawei-ntp`）
- **THEN** `EncoderFor("/ntp:ntp")`/`DecoderFor("/ntp:ntp")` SHALL 命中，
  `EncodeAnchor` SHALL 为 `/ntp:ntp`，XML 编码 SHALL 携带登记的 namespace

#### Scenario: 根名与 YANG prefix 不同的模块按根名路由（回归）

- **WHEN** 模块根容器名与 YANG prefix 不同（如 `tunnel-management`/`tnlm`、
  `routing-policy`/`rtp`、`network-instance`/`ni`），前端按根名派生路径
  `/tunnel-management:tunnel-management/...`
- **THEN** 描述符 SHALL 命中该路径（此前锚定 YANG prefix 导致控制台写链路不可达）

#### Scenario: 全表 namespace 唯一且 SchemaTree 入口存在

- **WHEN** 遍历注册表全部华为描述符
- **THEN** 每条 XML spec 的 namespace SHALL 非空且互不重复，Schema 闭包 SHALL 能
  解析出对应根容器 Entry；违反即单测失败（防表行漂移）
