## Context

- 前端零 i18n：无库、52/56 文件硬编码中文、ElementPlus 默认英文控件文案与中文界面混排（现状本就不一致）。
- snd i18n res：93 对 zh/en 完全对称（键集实测一致），键=带源模块前缀的 YANG 数据路径 `/huawei-vlan:vlan/vlans/vlan/id`；FieldDef.path 为根容器扁平路径 `/vlan/vlans/vlan/id`——差一个「源模块前缀」段，左树生成物已含 sourceModule↔rootContainers 映射可换算。
- 前端镜像构建上下文 = ./frontend（compose 实测），读不到 ../snd → res 须入库副本（后端 codegen 同理的既定模式）。
- 左树/模块双语字段③期已透出。

## Goals / Non-Goals

**Goals:**
- zh-cn/en-us 全界面切换（chrome + ElementPlus 控件 + 左树 + YANG 字段标签），localStorage 持久化
- YANG 字段标签查表：res 副本懒加载、路径换算、查不到回退节点名
- sync-snd-i18n 同步管线 + 升级步骤契约化

**Non-Goals:**
- 不做后端 FieldDef 多语言（标签映射纯前端，后端零改动）
- 不翻译业务数据（设备名/描述/日志内容等用户输入）
- 不做浏览器语言自动探测（显式切换 + 持久化足够，避免测试不确定性）
- 枚举值/校验消息的深度本地化（res 覆盖 name 为主；后续按需）

## Decisions

**D1：vue-i18n composition 模式（legacy:false），chrome 文案 key 按域组织（nav.*/devices.*/console.*…）。** 52 文件量大但机械；t() 替换与 locale 文件同 PR 同 commit 分域拆分（≤500 行/commit）。

**D2：res 副本入库 + make sync-snd-i18n，而非 docker 上下文扩容。** 改 compose context 影响所有环境构建缓存且把 10MB snd 拖进前端镜像上下文；副本模式与后端生成物同构，升级流程一条：换 snd 目录 → 重跑 gen-yang/go:generate/sync-snd-i18n。副本目录进 pr-size/commit-msg 排除（vendored 口径）。

**D3：字段标签查表放 useFieldLabel composable：输入（module 根容器、FieldDef.path、locale）→ 输出标签。** 路径换算：左树 store 提供 rootContainer→sourceModule；res 键 = `/${sourceModule}:${path 首段}` + 其余段。res 按 `assets/snd-i18n/${locale}/${sourceModule}-res.json` 懒加载（vite glob import，模块级缓存）。任何一步缺失回退 FieldDef.label（R08，界面永不空标签）。

**D4：ElementPlus locale 经 ElConfigProvider 响应式切换**（App.vue 包裹，locale computed 自 i18n.global.locale），一切换全局控件文案即变，无需刷新。

**D5：语言状态放独立 pinia store（locale store）+ localStorage**；不放 URL/后端（个人偏好非共享状态）。默认 zh-cn（现用户群）。

## Risks / Trade-offs

- [52 文件抽取遗漏中文] → 完成后 `grep -rP '[一-龥]'` 清点 src/（排除 locales/assets/注释），残留即未完成（T06 精神）；F2 抽查关键视图双语
- [res 键与 FieldDef.path 换算不匹配（嵌套 choice 剥离段等）] → 查表失败即回退节点名，不崩；F1 用真实 vlan res + 真实 schema path 做换算用例
- [4MB 副本入库] → 懒加载单模块 KB 级；排除清单防触发体积门禁
- [smoke 语言切换用例引入不稳定] → 只断言 2-3 个锚点文案，切回 zh 收尾

## Migration Plan

单 change 多 commit（按域 ≤500 行），预估 1-2 个 PR（基建+核心视图 / 长尾视图）。顺序：sync 管线+基建（i18n/locale store/ConfigProvider）→ F1 红绿 → 左树/Sidebar/Header 双语 → 设备/日志/设置视图 → 模块控制台字段标签查表 → 业务控制台/Dashboard → 中文清点 → smoke。回滚 = revert（默认 zh-cn 行为不变）。

## Open Questions

（无。）
