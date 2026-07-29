# ui-i18n — 界面中英文

## Purpose

界面双语（zh-cn/en-us）：语言切换持久化并联动 ElementPlus locale；chrome 文案经 vue-i18n 词表全覆盖；YANG 字段标签按语言经 snd res 副本查表（懒加载、缺档回退节点名）。翻译数据权威 = snd 包（左树双语字段 + i18n res）。
## Requirements
### Requirement: UI-01 语言切换与持久化

界面 SHALL 支持 zh-cn/en-us 双语：Header 或设置页提供切换入口；选择 SHALL 持久化于 localStorage 并在下次加载生效；默认 zh-cn。切换 SHALL 即时生效（无需刷新），且 SHALL 同步 Element Plus 组件库 locale（ElConfigProvider 响应式联动）。

#### Scenario: 切换即时生效并持久化
- **WHEN** 用户切换到 en-us
- **THEN** 导航/按钮/ElementPlus 控件文案 SHALL 即时变为英文；刷新后 SHALL 保持 en-us

### Requirement: UI-02 界面文案全覆盖

全部界面 chrome 文案（导航、布局、视图标题、按钮、提示、空态、校验消息模板）SHALL 经 vue-i18n locale 文件提供 zh-cn/en-us 两份；`frontend/src` SHALL NOT 残留硬编码界面中文（用户业务数据除外；以中文字符扫描清点为完成判据）。左树与模块双语名 SHALL 按当前语言取 zh/en 字段。

#### Scenario: 左树随语言切换
- **WHEN** 语言为 en-us
- **THEN** 左树分组 SHALL 显示 en-us 名（如 Ethernet Switching），zh-cn 时显示中文

#### Scenario: 无残留硬编码（负路径）
- **WHEN** 扫描 frontend/src 界面代码中的中文字符（排除 locales/assets/测试）
- **THEN** SHALL 为零残留

### Requirement: UI-03 YANG 字段标签本地化（snd res 查表）

模块控制台标签 SHALL 按「当前语言 + 模块源名」懒加载入库 res 副本（`frontend/src/assets/snd-i18n/{locale}/{sourceModule}-res.json`，由 `make sync-snd-i18n` 自 snd 包同步）并按 YANG 路径查表；查表任一环节缺失 SHALL 回退原始 YANG 节点名标签，SHALL NOT 出现空标签或加载失败（R08）。res 副本目录 SHALL 视为 vendored 生成物（体积门禁排除、升级随包同步）。本地化 SHALL 覆盖：

- **配置字段标签**：键 = 源模块前缀 + FieldDef.path（如 `/huawei-vlan:vlan/vlans/vlan/id`）。
- **rpc 标签**：rpc 为模块顶层语句，键**无根容器段**，= `/<sourceModule>:<rpcName>`（如 `/huawei-ifm:restart-if`）。
- **rpc input 叶标签**：键 = `/<sourceModule>:<rpcName>/input/<叶名>`（如 `/huawei-ifm:restart-if/input/if-name`）。

源模块名取自左树映射（缺失按 `huawei-<root>` 约定回退）。

#### Scenario: 中文标签查表
- **WHEN** 语言 zh-cn，vlan 控制台渲染 `/vlan/vlans/vlan/id` 字段
- **THEN** 标签 SHALL 为 res 中 `/huawei-vlan:vlan/vlans/vlan/id` 的 name（「VLAN标识」）

#### Scenario: rpc 标签与 input 叶查表
- **WHEN** 语言 zh-cn，huawei-ifm 控制台渲染 `restart-if` rpc
- **THEN** rpc 标签 SHALL 为 res 中 `/huawei-ifm:restart-if` 的 name（「重启接口」）
- **AND** 其 input 叶 `if-name` 标签 SHALL 为 `/huawei-ifm:restart-if/input/if-name` 的 name（「重启接口名」）

#### Scenario: 查不到回退（负路径）
- **WHEN** 某字段/rpc 路径在 res 无键或 res 文件缺失
- **THEN** SHALL 回退原始 YANG 节点名标签，SHALL NOT 报错

