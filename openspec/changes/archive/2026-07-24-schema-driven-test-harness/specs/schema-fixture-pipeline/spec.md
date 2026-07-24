## ADDED Requirements

### Requirement: SF-01 全模块 fixture 导出

系统 SHALL 提供一条生成管线，把后端**实际已加载**的全部 YANG 模块的嵌套呈现 schema 导出为版本化 JSON fixture 并入库。模块集合 SHALL 由运行期加载结果动态发现，SHALL NOT 硬编码模块名单或模块数量——新接入的模块必须在接入当次自动获得 fixture，无需修改导出器代码。

导出 SHALL NOT 依赖 HTTP 服务、docker 或任何外部进程，SHALL 可在 `go test` 与 CI 中直接执行。

#### Scenario: 导出覆盖全部已加载模块

- **WHEN** 对已加载 N 个模块的 schema 执行导出管线
- **THEN** 系统 SHALL 为每个已加载模块产出恰好一份 fixture，且 fixture 数量等于 N
- **AND** 每份 fixture SHALL 含该模块的 module/title/vendor 标识与嵌套 fields 树

#### Scenario: 新增模块自动纳入（动态发现）

- **WHEN** 后端新接入一个 YANG 模块并成功加载，导出管线代码未作任何修改
- **THEN** 重新执行导出 SHALL 自动产出该新模块的 fixture

#### Scenario: 脱离 HTTP 与容器执行

- **WHEN** 在无运行中后端服务、无 docker 的环境执行导出管线
- **THEN** 导出 SHALL 成功完成并产出完整 fixture

### Requirement: SF-02 导出确定性

导出结果 SHALL 是确定的：对同一份 YANG 模型输入，任意次数、任意进程的导出 SHALL 产出逐字节相同的 fixture。字段与子节点顺序 SHALL 稳定，SHALL NOT 依赖 map 遍历顺序。

本要求把「确定性」从当前实现细节升格为受保护契约：上游 goyang 升级或 schema 包重构若引入非确定顺序，门禁 SHALL 失败。

#### Scenario: 重复导出结果一致

- **WHEN** 对同一 YANG 模型连续执行两次导出
- **THEN** 两次产出的 fixture SHALL 逐字节相同

#### Scenario: 非确定顺序被拦截

- **WHEN** schema 遍历链路被改为依赖 map 遍历顺序而未重新定序
- **THEN** 确定性测试 SHALL 失败，阻断提交

### Requirement: SF-03 fixture 忠实于线上契约

fixture SHALL 与用户经 HTTP 实际取得的 schema 完全一致。对每个已加载模块，导出管线产出的 schema SHALL 与 `GET /yang/schema/:module?form=nested` 响应的 `data` 字段逐字节相等。

本要求是整个测试体系的信任锚点：fixture 若与线上契约脱钩，基于 fixture 的下游验证会全绿而实际渲染是错的——比没有测试更危险。等值校验 SHALL 覆盖全部已加载模块，SHALL NOT 抽样。

#### Scenario: 导出结果与 HTTP 响应等值

- **WHEN** 对任一已加载模块分别取得导出 fixture 与 `GET /yang/schema/:module?form=nested` 的 `data`
- **THEN** 两者序列化结果 SHALL 逐字节相等

#### Scenario: 契约脱钩被拦截

- **WHEN** HTTP schema 构建路径被修改而导出路径未同步（或反之）
- **THEN** 等值测试 SHALL 失败，阻断提交

### Requirement: SF-04 fixture 漂移门禁

CI SHALL 以 regen-and-diff 方式校验 fixture 未漂移：重新执行导出后若与入库 fixture 存在任何差异，CI SHALL 失败。fixture SHALL 被视为生成物——手工编辑 fixture 而不改 schema 源，SHALL 被同一门禁拦截。

fixture 目录 SHALL 加入 PR 体积门禁排除清单，口径对齐既有生成物排除项。

#### Scenario: 后端 schema 变更未刷 fixture

- **WHEN** 后端 schema 构建逻辑或 YANG 模型变更导致输出改变，但 fixture 未重新生成
- **THEN** CI 漂移校验 SHALL 失败并指出差异模块

#### Scenario: 手工篡改 fixture

- **WHEN** 直接编辑入库 fixture 内容而未变更任何 schema 源
- **THEN** CI 漂移校验 SHALL 失败

#### Scenario: fixture 不计入 PR 体积

- **WHEN** 一个 PR 仅因重新生成 fixture 而产生大量行变更
- **THEN** PR 体积门禁 SHALL NOT 因这些生成物行数而拦截
