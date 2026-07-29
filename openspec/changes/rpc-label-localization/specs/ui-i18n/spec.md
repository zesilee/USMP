## MODIFIED Requirements

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
