# frontend (delta)

## ADDED Requirements

### Requirement: FE-17 业务网络配置菜单组与平台作用域控制台

侧边栏 SHALL 出现「业务网络配置」菜单组：由意图 YANG 模块的 task-name category 经既有分桶机制自动生成（零菜单硬编码，R05）。业务能力 SHALL 渲染为**平台作用域**控制台（一个意图实例管理多台设备，不绑定单设备上下文）：意图表单 SHALL 由意图 YANG schema 自动渲染（devices 嵌套 list 含增删改）、实例列表 SHALL 展示每实例收敛状态汇总（deviceStates 聚合：全 synced/部分 failed/pending）、实例详情 SHALL 展示每设备状态与失败原因。

#### Scenario: 菜单组自动出现
- **WHEN** 意图 YANG 模块带业务 category 注册且被 `GET /yang/modules` 返回
- **THEN** 侧边栏 SHALL 自动出现「业务网络配置」组及该能力入口，无前端菜单代码改动

#### Scenario: 意图表单模型驱动
- **WHEN** 打开「跨设备 VLAN 打通」控制台新建意图
- **THEN** 表单 SHALL 按意图 YANG 渲染（vlan-id 数字输入带 range、devices 嵌套 list 可增删改行），校验失败 SHALL 行内提示且不提交

#### Scenario: 收敛状态呈现
- **WHEN** 某意图 2 台设备中 1 台 failed
- **THEN** 实例列表 SHALL 呈现部分失败态，详情 SHALL 列出失败设备与原因

### Requirement: FE-18 原生控制台归属徽标

原生模块控制台渲染被业务意图认领的对象/路径时 SHALL 显示「由业务配置 <意图名> 管理」徽标；用户对认领路径提交手改且响应含归属警告时 SHALL 以非阻断方式提示（下发照常）。

#### Scenario: 认领对象带徽标
- **WHEN** 原生 vlan 控制台列表中某 VLAN 被意图 X 认领
- **THEN** 该行 SHALL 显示归属徽标（含意图名）

#### Scenario: 手改警告非阻断
- **WHEN** 用户对认领路径提交配置且响应含 ownershipWarning
- **THEN** SHALL 弹出非阻断提示说明意图收敛会覆盖手改，提交结果照常呈现
