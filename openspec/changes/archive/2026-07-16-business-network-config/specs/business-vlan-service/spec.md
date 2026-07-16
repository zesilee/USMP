# business-vlan-service (delta)

## ADDED Requirements

### Requirement: BVS-01 跨设备 VLAN 打通意图模型

首个业务能力 SHALL 为「跨设备 VLAN 打通」，意图 YANG 模型 SHALL 至少包含：`vlan-id`（uint16，range 1..4094，list key）、`name`（string，可选）、`devices` list（key=ip）：每设备 `access-ports` leaf-list 与 `trunk-ports` leaf-list（接口名）。模型 SHALL 携带 task-name 扩展归入「业务网络配置」category。

#### Scenario: 模型字段与约束
- **WHEN** 校验意图实例 {vlan-id:100, devices:[{ip, access-ports, trunk-ports}]}
- **THEN** vlan-id 越界(0/4095) SHALL 被拒绝，devices 缺 ip SHALL 被拒绝，合法实例 SHALL 通过

### Requirement: BVS-02 每设备展开规则

一个 VLAN 打通意图 SHALL 对 devices 中每台设备展开为：huawei-vlan 片段（vlans 下建 id=vlan-id 的条目，含 name）+ huawei-ifm 片段（access 口配 PVID/access 型、trunk 口放行该 VLAN，具体叶子映射见 design）。展开 SHALL 复用既有原生模块路径（与模块控制台同路径），保证声明式对账与删除通道可直接工作。

#### Scenario: 双设备展开
- **WHEN** 意图 {vlan-id:100, devices:[A{access:[GE0/0/1]}, B{trunk:[GE0/0/2]}]}
- **THEN** SHALL 产出 A 的 vlan100+GE0/0/1 access 片段与 B 的 vlan100+GE0/0/2 trunk 片段，路径与原生 vlan/ifm 模块一致

### Requirement: BVS-03 端到端收敛验收

对接两台模拟网元的集成测试 SHALL 覆盖：创建意图→两台设备 running 收敛出 VLAN 与端口配置→status 全 synced；修改意图（换端口/移除设备）→增量收敛与孤儿清理；删除意图→两台设备配置清除。

#### Scenario: 创建到收敛
- **WHEN** 创建覆盖 2 台模拟网元的 VLAN 打通意图
- **THEN** 两台 sim 的 running SHALL 可断言到 vlan 与端口配置，CR status SHALL Converged=True

### Requirement: BVS-04 负路径与并发防线

SHALL 覆盖（T02b 完备矩阵）：单台设备 prepare 失败→全体不落 running；设备离线→deviceStates failed+重试；并发提交同 vlan-id 不同意图→后者 Validated=False（认领冲突）或按 design 定义的冲突语义呈现；-race 下无数据竞态（R09）。

#### Scenario: prepare 失败不留残配置
- **WHEN** 设备 B 注入 edit-config 错误
- **THEN** 设备 A 的 candidate SHALL 被 discard，两台 running 均无该 VLAN
