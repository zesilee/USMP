## Why

snd 融合四期计划④（收官期）：界面需支持中英文。现状零 i18n 基础设施——无 vue-i18n、52/56 前端文件硬编码中文、Element Plus locale 未配置、YANG 表单字段标签为英文节点名无翻译。snd 包已提供 93 对完全对称的中英 res.json（键=YANG 数据路径→本地化名）与左树双语字段（③期已随载荷透出），是字段级翻译的现成数据源。

## What Changes

- 前端引入 vue-i18n（legacy=false composition 模式）+ `ElConfigProvider` 联动 Element Plus locale（zh-cn/en-us）；语言偏好 localStorage 持久化，Header/设置页提供切换入口
- 界面 chrome 文案全量抽取到 `locales/{zh-cn,en-us}.json`（导航/布局/6 个视图/通用提示），硬编码中文清零（业务数据除外）
- **YANG 字段标签本地化**：`make sync-snd-i18n` 把 `snd/resources/i18n` 同步为 `frontend/src/assets/snd-i18n/`（入库副本——前端镜像构建上下文仅 frontend/，无法读 snd/，与后端 codegen 同理）；模块控制台按「当前语言+模块」懒加载对应 res.json，FieldDef.label 按 YANG 路径查表（键形如 `/huawei-vlan:vlan/vlans/vlan/id`，源模块名取自左树 sourceModule 映射）；查不到回退 YANG 节点名（R08）
- 左树/模块列表双语消费：LeftTreeMenu 按当前语言取 zh/en 字段（③期已预留）
- pr-size/commit-msg 排除 `frontend/src/assets/snd-i18n/**`（vendored 副本，与 snd/** 同理，两处同步）
- F4 smoke 增语言切换冒烟（切 en → 导航/表单标签变英文）

## Capabilities

### New Capabilities

- `ui-i18n`: 界面双语契约（语言切换与持久化、chrome 文案全覆盖、YANG 字段标签查表与回退、Element Plus locale 联动）

### Modified Capabilities

- `snd-package`: SP-01 升级步骤补充「重跑 `make sync-snd-i18n`」（前端 res 副本与包同步）

## Impact

- 前端：main.ts/App.vue（i18n+ConfigProvider）、locales 两份、全部含中文文案的组件/视图/stores（52 文件）、useFieldLabel composable（新）、assets/snd-i18n（186 个 json 副本）
- 构建：Makefile 增 sync-snd-i18n；vite 懒加载 glob
- 测试：F1（locale util/字段查表回退/持久化）、F2（切换后组件文案、左树 en 渲染、label 查表）、F4（切换冒烟）
- 后端：零改动
- 体积：res 副本 ~4MB 入库（gzip 后网络传输每模块 KB 级，懒加载）
- 不动：后端 API 契约、FieldDef 结构（标签映射纯前端）
