# yang-api — delta（retire-openconfig-models）

## ADDED Requirements

### Requirement: BR-11 厂商边界（仅 huawei + usmp business）

运行期 schema 树（`yangschema.Load()`）SHALL 仅加载 huawei 厂商生成模型与 usmp 业务意图模型；SHALL NOT 加载 openconfig 或 ietf 模型。`GET /api/v1/yang/modules` 返回的模块 `vendor` 取值 SHALL 仅为 `huawei` 或 `usmp`。ietf 与 openconfig 的 `.yang` 文件仅允许作为华为模型的**构建期编译依赖**（yang-models submodule）存在，SHALL NOT 进入运行期 schema 树；NETCONF 协议命名空间（`urn:ietf:params:xml:ns:netconf:*` 等）不属于模型支持范畴，不受本条约束。

#### Scenario: 模块列表仅含 huawei/usmp
- **WHEN** Manager 加载全部生成模型后请求 `GET /api/v1/yang/modules`
- **THEN** 每个模块项的 `vendor` SHALL 为 `huawei` 或 `usmp`，SHALL NOT 出现 `openconfig`

#### Scenario: openconfig 模块不可获取 schema（负路径）
- **WHEN** 请求 `GET /api/v1/yang/schema/interfaces`（原 openconfig-interfaces 根容器）等未加载模块
- **THEN** SHALL 走 BR-04 未加载模块降级路径，SHALL NOT 返回 openconfig 派生 schema
