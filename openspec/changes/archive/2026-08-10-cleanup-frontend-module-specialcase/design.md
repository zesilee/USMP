# 设计：前端遗留特判与死代码清理

## Context

`/config/*` 旧路由退役（retire-legacy-config-redirects）后全面审视确认：模块控制台渲染零特判，遗留集中在旧配置页时代的 4 处——logRows 字符串特判、Header 死面包屑映射、两个手写类型死文件、useDeviceConfig 死组合函数。

## Goals / Non-Goals

**Goals:**

- 消除 `src/` 内最后一处按模块名分支的逻辑（logRows），任意新模块无需改前端即可获得正确的日志操作类型标签。
- 死代码清零：零引用文件删除、不可达映射删除、连带 locale 词条删除。

**Non-Goals:**

- 不动合理的默认落点（Dashboard→vlan、Devices→ifm）与 R08 菜单回退项（FE-13 spec 明文）。
- 不动业务层 `business-vlan-service`（首能力拍板，多 Kind 泛化是已记录的独立债）。
- 不改后端审计记录的 path 形状。

## Decisions

- **opLabelOf 派生规则**：`path 首段 → 剥 `:` 前缀 → moduleTitles[段名] ?? 段名；空路径 → t('logs.opGeneric')`。标题映射由 `Logs.vue` 从 menu store `nativeModules`（name→title）计算后作参数传入——logRows 保持纯函数可独测，store 未加载时回退段名（R08 降级，不空白）。旧「VLAN 配置」等四个拼装词条删除：标签直接用模块菜单标题，与左树称谓一致，避免双套命名。
- **extractRows 迁移而非留守**：useDeviceConfig 主体（loadSchema/loadItems/extractItemFields/findItemListPath）零调用方，仅 extractRows 被 RpcExecuteTab 的 leafref 下拉借用。独立成 `utils/extractRows.ts`（含原注释与 R08 降级行为），避免为一个工具函数保留 94 行死组合函数。
- **Header.test 面包屑用例改用 `devices`**：原用例靠桩路由 `name: 'vlan'` 测「name→词条」映射，映射本身是死的；改用现役 `devices` 保留测试意图（映射机制 + fallback 路径末段），不再守护幽灵词条。

## Risks / Trade-offs

- [日志页标签措辞变化] → 旧记录中 vlan 路径从「VLAN 配置」变为模块菜单标题（如「VLAN」）；属展示细节且与左树一致，用户已确认清理方向。
- [menu store 未加载时显示原始段名] → 仅发生在 /yang/modules 失败且直开日志页的窗口，降级可读（段名即模块名），符合 R08。
