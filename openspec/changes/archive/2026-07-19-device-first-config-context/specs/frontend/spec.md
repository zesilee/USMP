# frontend — delta（device-first-config-context）

## MODIFIED Requirements

### Requirement: FE-10 通用模块控制台（Tab 由模块根派生）

前端 SHALL 提供通用模块控制台页（路由 `/module/:module`，零 per-module props）：
右侧内容区 SHALL 渲染面包屑（配置/厂商/模块/激活 Tab）与一级 Tab；Tab 集合 SHALL 由
nested schema 模块根的顶层子节点自动派生——list→列表 Tab、group/choice→表单 Tab、
散落根叶子聚合为「基本属性」表单 Tab。SHALL NOT 针对任一具体 YANG 模块硬编码
Tab/列/字段。Tab 切换 SHALL 保留各 Tab 的表单与搜索状态。

设备选择 SHALL 为**全局设备上下文**（device store 单一事实源，IP 口径）：控制台设备
下拉 SHALL 双向绑定全局上下文，模块间切换 SHALL 保持选中设备不变（先选设备、后做
配置管理）。设备管理「查看配置」入口与 `?device=<ip>` 深链 SHALL 写入同一全局上下文。
未选设备时 SHALL 展示引导空态（提示先选择设备），SHALL NOT 静默渲染空列表/空表单。
平台作用域业务控制台（`/business/:module`）SHALL NOT 绑定设备上下文。

#### Scenario: huawei-ifm 派生

- **WHEN** 打开 `/module/ifm`
- **THEN** Tab 集合 SHALL 含 `global`（表单）、`damp`（表单）、`auto-recovery-times`（列表或表单）、
  `interfaces`（列表）等根子节点，无任何硬编码模块名

#### Scenario: schema 加载失败降级

- **WHEN** schema API 失败
- **THEN** 页面 SHALL 展示错误提示且不崩（R08），设备选择仍可用

#### Scenario: 跨模块切换保持选中设备

- **WHEN** 在 `/module/ifm` 选中设备 192.168.1.2 后经左树切换到 `/module/vlan`
- **THEN** VLAN 控制台 SHALL 已选中 192.168.1.2，无需重新选择，配置数据按该设备加载

#### Scenario: 深链与「查看配置」写入全局上下文

- **WHEN** 从设备管理点击某设备「查看配置」（或直接打开 `/module/ifm?device=<ip>`）
- **THEN** 全局设备上下文 SHALL 更新为该设备，后续切换到其它模块页 SHALL 沿用该选中

#### Scenario: 未选设备引导空态

- **WHEN** 全局上下文无选中设备时打开任一 `/module/:module`
- **THEN** 内容区 SHALL 展示「请先选择设备」引导空态而非空数据表单/列表，选中设备后 SHALL 恢复正常渲染
