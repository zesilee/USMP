# yang-native-runtime delta: config-write-validation

## MODIFIED Requirements

### Requirement: YN-04 服务端校验子集（现状等价）

系统 SHALL 提供基于 Schema IR 的服务端校验器替代生成的 `Validate()`（意图层与
原生配置写路径共同消费）：覆盖 pattern/range/length/enum 值域校验，与前端表单校验
同源（同一 IR 元数据）。范围 SHALL 为现状等价：must/when SHALL NOT 做运行时求值
（与 ygot `Validate()` 现状一致，设备侧兜底）。校验失败 SHALL 返回命名路径与约束的
明确错误。

校验实现 SHALL 位于 `pkg/yang-runtime/schema` 包内并经 `schema.Schema` 接口的
`Validate(path, config)` 对外提供，SHALL NOT 独立成包——独立成包会与 `schema` 构成
循环依赖，使接口方法无法实现。语义冻结项（mandatory 不校验、min-elements 仅对存在
的空 list 生效、must/when 不求值）SHALL NOT 因实现位置迁移而改变；收紧任一项 SHALL
作为独立契约变更另行拍板。

#### Scenario: 意图校验行为等价
- **WHEN** 以既有意图校验（`intent/cr.go`）的行为快照用例分别跑旧 `Validate()` 与新校验器
- **THEN** 接受/拒绝结论 SHALL 一致，拒绝时错误 SHALL 指明违反的叶路径与约束

#### Scenario: 越界值拒绝
- **WHEN** 提交 range 越界或 pattern 不匹配的叶值
- **THEN** 校验 SHALL 拒绝并命名该叶，SHALL NOT 下发到设备

#### Scenario: 实现迁移后语义不漂
- **WHEN** 校验实现由独立包迁入 `schema` 包后重跑意图层快照与双跑用例
- **THEN** 接受/拒绝结论 SHALL 逐条不变

#### Scenario: 接口方法不再是空壳
- **WHEN** 经 `schema.Schema.Validate(path, config)` 校验一个违反约束的配置
- **THEN** SHALL 返回指明叶路径与约束的错误，SHALL NOT 恒返回 nil
