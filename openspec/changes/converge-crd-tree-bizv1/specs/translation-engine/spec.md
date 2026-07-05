## MODIFIED Requirements

### Requirement: translator 消费部署权威 schema api/biz/v1

翻译器 SHALL 以 `api/biz/v1`（有生成 CRD YAML=部署 schema）为输入类型，字段映射对齐 biz/v1↔厂商 ygot，替换此前绑 `api/v1` 导致的解码错位。翻译输出 SHALL 为 ygot 生成结构（R04）。

#### Scenario: VLAN biz/v1 字段映射到 huawei ygot
- **WHEN** 翻译 `api/biz/v1.BusinessVlanSpec`（含 MacLearning/BroadcastDiscard/UnknownMulticastDiscard）
- **THEN** SHALL 映射到 huawei ygot 同名字段（此前被忽略的字段现生效）

#### Scenario: Interface 收敛为 L2 模型
- **WHEN** 翻译 `api/biz/v1.BusinessInterfaceSpec`（access/trunk/hybrid）
- **THEN** SHALL 产出 L2 接口 ygot（`IfName→Name`、MTU、ServiceType=L2）；不再有 L3/IpAddress 分支（biz/v1 无此字段）

#### Scenario: 无 api/v1 引用
- **WHEN** 审计 translator + crdsource 的导入
- **THEN** SHALL 无 `api/v1` 引用（统一 `api/biz/v1`）
