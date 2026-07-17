## ADDED Requirements

### Requirement: DR-05 编码锚点声明

编码描述符 SHALL 声明 `EncodeAnchor`——其 `NewStruct` 容器对应的规范配置路径（如 `/vlan:vlan/vlan:vlans`、`/system:system`）。config-api 写路径 SHALL 据此计算请求路径相对锚点的后缀段并机械包裹请求体（段名剥模块前缀）后根级解码；请求路径与锚点不构成前缀关系时调用方 SHALL 显式拒绝。该字段是 SND 谓词声明式化的第一块数据（后续 MatchEncode 可由锚点派生）。

#### Scenario: 锚点相对包裹解码
- **WHEN** system 描述符锚点为 `/system:system`，请求路径 `/system:system/system:system-info` 携扁平子树
- **THEN** 写路径 SHALL 包裹为 `{"system-info": <子树>}` 并成功解码进 `HuaweiSystem_System`

#### Scenario: path 即锚点零包裹
- **WHEN** 请求路径与描述符锚点相同
- **THEN** 请求体 SHALL 原样作为锚点相对 RFC7951 解码

#### Scenario: 非前缀路径拒绝（负路径）
- **WHEN** 请求路径不以该描述符锚点为前缀
- **THEN** 调用方 SHALL 显式拒绝（400），SHALL NOT 猜测包裹
