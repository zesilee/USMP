# 前端 delta：退役旧路由重定向

## MODIFIED Requirements

### Requirement: FE-13 模型驱动原生配置导航与路由迁移

左侧**原生配置**菜单 SHALL 由 `/yang/modules` 返回的模块列表驱动生成（指向 `/module/:name`），
加载失败 SHALL 回退既有硬编码项（R08）。模块项携带 `category` 时菜单 SHALL 按 category
分组展示；无 `category` 的模块 SHALL 归入默认分组，分组渲染 SHALL NOT 因缺失 category 失败（R08）。
legacy 路由 `/native/:module`、`/config/route`、`/config/interface`、`/config/vlan`
SHALL 不存在（Stack A CRD 死路与旧配置页入口均已退役，旧地址兼容期结束，无重定向义务）；
站内跳转 SHALL 直接使用 `/module/:module`，SHALL NOT 依赖任何旧路由。

#### Scenario: 菜单生成与回退

- **WHEN** `/yang/modules` 返回含 `ifm`（模块根名）的列表
- **THEN** 菜单 SHALL 含指向 `/module/ifm` 的项
- **WHEN** 该 API 失败
- **THEN** 菜单 SHALL 显示回退项且不崩

#### Scenario: 任务域分组

- **WHEN** 模块列表含带 `category` 与不带 `category` 的模块
- **THEN** 菜单 SHALL 按 category 分组展示带值模块，无值模块 SHALL 归入默认分组且渲染不失败

#### Scenario: 菜单命名与概念对齐

- **WHEN** 渲染左侧导航
- **THEN** 模块控制台菜单组标题 SHALL 为「原生配置」，SHALL NOT 存在指向 `/native/*` 的菜单项

#### Scenario: 旧配置页路由退役

- **WHEN** 检视路由表
- **THEN** SHALL NOT 存在 `/config/interface`、`/config/vlan` 路径（含重定向）
- **WHEN** 用户在 Dashboard 点击「下发配置」
- **THEN** SHALL 直接导航到 `/module/vlan`
