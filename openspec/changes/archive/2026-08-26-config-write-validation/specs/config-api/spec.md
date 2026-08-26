# config-api delta: config-write-validation

## MODIFIED Requirements

### Requirement: BR-08 无效请求拒绝

非法 JSON 或类型转换失败 SHALL 返回 `code=400`，SHALL NOT 存储或触发对账。

写路径（`POST /api/v1/config/:ip/*path`）SHALL 在类型转换成功之后、写入 desired
之前，以请求 path 解析 schema 节点并对**本次提交的配置**跑 Schema IR 约束校验
（pattern 锚定全匹配 / range / length / min-elements，YN-04 语义子集）。
校验不通过 SHALL 返回 `code=400` 并透出被违反的叶路径与约束；
SHALL NOT 写 ConfigStore、SHALL NOT 触发对账、SHALL NOT 触达设备。

校验对象 SHALL 为本次提交的增量，SHALL NOT 为与存量 desired 合并后的结果
（存量历史非法值 SHALL NOT 导致此后所有提交被拒）。
YANG 模型未编码的域约束（如华为 VLAN ID 范围 1–4094）SHALL 继续由显式校验承担，
与 IR 约束校验并列生效。

#### Scenario: 非法 JSON
- **WHEN** 请求 body 非合法 JSON
- **THEN** SHALL 返回 `code=400` "Invalid request"，不写 ConfigStore

#### Scenario: 违反 YANG 约束的配置被拒
- **WHEN** 提交的配置含违反 schema 约束的叶值（pattern 不匹配 / range 越界 / length 超限）
- **THEN** SHALL 返回 `code=400`，错误 SHALL 指明违反的叶路径与约束
- **AND** ConfigStore SHALL NOT 被写入，对账 SHALL NOT 被触发，设备 SHALL NOT 被触达

#### Scenario: 合法配置照常下发
- **WHEN** 提交的配置满足全部 schema 约束
- **THEN** SHALL 照常写入 desired 并触发对账，行为与接入校验前一致

#### Scenario: 存量非法值不阻断新提交
- **WHEN** 已存 desired 含历史非法叶值，本次提交的增量本身合法
- **THEN** SHALL 接受本次提交（校验只针对提交增量，不针对合并结果）

#### Scenario: 模型未编码的域约束仍然生效
- **WHEN** 提交 VLAN ID 为 0 或 4095（YANG 模型未编码此范围）
- **THEN** SHALL 返回 `code=400`，不写 ConfigStore
