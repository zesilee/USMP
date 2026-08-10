# 前端 delta：日志操作类型模型驱动派生

## ADDED Requirements

### Requirement: FE-26 日志操作类型模型驱动派生

操作日志的「操作类型」标签 SHALL 由审计路径首段模块名派生（剥命名空间前缀）：
已知模块 SHALL 显示其菜单标题（与左侧导航称谓一致），未知模块 SHALL 回退显示段名，
空路径 SHALL 回退通用标签。派生 SHALL NOT 按具体模块名硬编码分支（R05 同口径）——
新模块接入 SHALL 零前端改动获得正确标签。模块标题映射不可用时 SHALL 降级为段名展示（R08）。

#### Scenario: 已知/未知模块与降级

- **WHEN** 审计路径为 `/vlan:vlan/vlan:vlans` 且模块标题映射含 `vlan`
- **THEN** 操作类型 SHALL 显示 vlan 模块的菜单标题
- **WHEN** 审计路径首段不在模块标题映射中（含映射整体不可用）
- **THEN** 操作类型 SHALL 显示该段名而非通用占位
- **WHEN** 审计路径为空
- **THEN** 操作类型 SHALL 显示通用标签且渲染不失败
