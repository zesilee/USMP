# config-api (delta)

## ADDED Requirements

### Requirement: BR-11 归属警告（软归属）

`POST /api/v1/config/:ip/*path` 与 `DELETE` 行删除命中业务意图认领路径时，SHALL 正常执行（不拦截）并在响应体附加归属警告字段（认领的意图 CR 名称、提示意图收敛会覆盖手改）；归属查询 SHALL 经 API 暴露供前端徽标渲染。硬锁（409 拒绝）不在本变更范围。

#### Scenario: 手改命中认领路径返回警告
- **WHEN** 对被意图 X 认领的 vlan 路径 POST 配置
- **THEN** 下发 SHALL 照常接受，响应 SHALL 含 `ownershipWarning`（意图 X 名称与提示文案）

#### Scenario: 未认领路径无警告
- **WHEN** 对无任何意图认领的路径 POST 配置
- **THEN** 响应 SHALL 不含归属警告字段
