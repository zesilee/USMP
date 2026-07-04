## ADDED Requirements

### Requirement: YANG 模型加载进 schema 树（混合源）

Manager SHALL 在启动时构建可用的 YANG schema 树，供动态 schema 生成与 path 校验使用，消除 schema 树运行时为空的空转（迁移债 D4）。schema 来源 SHALL 采用混合策略：优先依设备 NETCONF hello capabilities 收敛该设备支持的 YANG 模块集合，模块的属性 schema 取自 `internal/generated` 已生成的 ygot YANG 模型树作为回退基线。

#### Scenario: 从 ygot 模型树构建 schema
- **WHEN** Manager 启动且未从设备获得能力信息
- **THEN** schema 树 SHALL 从已生成的 ygot YANG 模型（huawei/openconfig）构建，`GetSchema().Modules()` 返回非空模块列表

#### Scenario: 设备能力收敛模块集合
- **WHEN** 某设备的 NETCONF hello capabilities 已知
- **THEN** 该设备可用的 YANG 模块集合 SHALL 收敛为其 capabilities 广告的模块，属性 schema 仍取自对应 ygot 模型树

#### Scenario: schema 缺失不崩溃
- **WHEN** 请求一个 schema 树中不存在的模块
- **THEN** 系统 SHALL 返回明确的「模块未知」结果而非 panic（R08）

### Requirement: ConfigStore 可枚举设备与路径

内存 ConfigStore SHALL 提供 `List` 与 `ListDevices` 的真实实现（消除返回 nil 的空转 D7），使 `PeriodicSource` 能枚举待轮询设备，且不引入任何数据库（R03，仍为纯内存）。

#### Scenario: 列出已存期望配置的设备
- **WHEN** 已通过 ConfigStore.Set 写入若干设备的期望配置后调用 ListDevices
- **THEN** SHALL 返回这些设备的去重列表

#### Scenario: 空存储返回空列表
- **WHEN** ConfigStore 为空时调用 List/ListDevices
- **THEN** SHALL 返回空列表而非 nil 或错误
