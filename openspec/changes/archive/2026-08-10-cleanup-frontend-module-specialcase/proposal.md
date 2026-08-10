# 清理前端对 ifm/vlan 的遗留特判与死代码

## Why

审视确认核心渲染链路已完全模型驱动（R05），但存在 4 处旧配置页时代的遗留：1 处按模块名硬编码的特判逻辑（日志操作类型标签，新模块全部落「配置变更」且 `route` 子串会误匹配）+ 3 处死代码。清理消除特判、缩小维护面。

## What Changes

- `utils/logRows.ts`：`opLabelOf` 从「路径子串 includes 特判 vlan/ifm/system/route」改为模型驱动——取路径首段模块名（剥命名空间前缀），已知模块显示其菜单标题、未知模块回退段名、空路径回退通用标签；`Logs.vue` 传入 menu store 的模块标题映射。删除 `logs.opVlan/opInterface/opSystem/opRoute` 词条。
- `Header.vue`：删除 CRUMB_KEYS 中 `interface/vlan/route/native` 四条死映射（无现役路由 name 可命中）及对应 locale 词条。
- 删除零引用死文件 `types/vlan.ts`、`types/interfaces.ts`（手写 YANG 类型镜像，R04 反模式残留）。
- `composables/useDeviceConfig.ts`：主组合函数已无调用方；唯一活代码 `extractRows` 迁至 `utils/extractRows.ts`（RpcExecuteTab 改引），整文件删除，测试同步迁移。

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `frontend`：ADDED FE-26「日志操作类型模型驱动派生」——把本次清理的实质固化为契约（标签由路径模块名+菜单标题派生，SHALL NOT 按模块名硬编码分支，新模块零前端改动）。面包屑死映射与死文件删除为纯实现细节，无 Requirement 变更。

## Impact

- `frontend/src/utils/logRows.ts`、`views/Logs.vue`、`components/layout/Header.vue`、`locales/{zh-cn,en-us}.json`
- 删除：`types/vlan.ts`、`types/interfaces.ts`、`composables/useDeviceConfig.ts`
- 新增：`utils/extractRows.ts`；`components/config/RpcExecuteTab.vue` 改 import
- 测试：`test/utils/logRows.test.ts` 重写标签用例、`test/components/Header.test.ts` 面包屑用例改用现役路由名、`test/composables/useDeviceConfig.test.ts` 中 extractRows 用例迁至 `test/utils/extractRows.test.ts` 后删除
- 用户可感知变化：日志页「操作类型」列——新接入模块从「配置变更」变为显示模块名；无后端、无 API 变更
