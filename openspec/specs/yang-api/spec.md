# yang-api - 行为契约

## 接口定义

### GET /api/v1/yang/modules

**描述**：返回所有已支持的YANG模块列表

**参数**：无

**响应**：
| 状态码 | 含义 | Body |
|--------|------|------|
| 200 | 成功 | `{"code":0,"data":[...],"success":true}` |

**数据模型 — YangModuleInfo**：
| 字段 | 类型 | 约束 | 示例 |
|------|------|------|------|
| name | string | 模块名 | "huawei-ifm" |
| title | string | 中文标题 | "华为接口管理" |
| vendor | string | 固定"huawei" | "huawei" |
| path | string | YANG根路径 | "/ifm" |
| description | string | 模块描述 | "Network interfaces configuration" |
| type | string | 根节点类型（数字字符串） | "1" |

### GET /api/v1/yang/schema/:module

**描述**：返回指定YANG模块的动态表单Schema定义

**参数**：
| 参数 | 位置 | 类型 | 必填 | 约束 | 示例 |
|------|------|------|------|------|------|
| module | path | string | 是 | YANG模块名 | "huawei-ifm" |

**响应**：
| 状态码 | 含义 | Body |
|--------|------|------|
| 200 | 成功 | `{"code":0,"data":{...},"success":true}` |

**数据模型 — YangSchema**：
| 字段 | 类型 | 约束 | 示例 |
|------|------|------|------|
| module | string | 模块名 | "huawei-ifm" |
| title | string | 中文标题 | "华为接口管理" |
| vendor | string | 固定"huawei" | "huawei" |
| fields | FieldDef[] | 表单字段定义 | 见下方 |
| listCols | FieldDef[] | 列表视图列定义 | 见下方 |

**数据模型 — FieldDef**：
| 字段 | 类型 | 约束 | 示例 |
|------|------|------|------|
| path | string | 字段路径 | "ifName" |
| type | string | 字段类型(string/number/enum/boolean) | "string" |
| label | string | 中文标签 | "接口名称" |
| placeholder | string | 可选，占位提示 | "例如: GE0/0/1" |
| required | bool | 可选，是否必填 | true |
| pattern | string | 可选，正则校验 | "^[0-9]+$" |
| default | any | 可选，默认值 | 1500 |
| options | Option[] | 可选，enum选项 | 见下方 |
| group | string | 可选，分组名 | "基本信息" |
| minimum | int | 可选，最小值 | 1 |
| maximum | int | 可选，最大值 | 4094 |
| readonly | bool | 可选，只读标记 | false |

**数据模型 — Option**：
| 字段 | 类型 | 约束 | 示例 |
|------|------|------|------|
| label | string | 选项中文显示 | "启用" |
| value | any | 选项值 | "up" |

## 业务规则

### BR-01: 模块列表-有已加载模块

- Given: Manager中已加载YANG模块
- When: 调用 GET /api/v1/yang/modules
- Then: 从Schema.Modules()遍历返回实际模块列表

### BR-02: 模块列表-无已加载模块

- Given: Manager中无已加载YANG模块
- When: 调用 GET /api/v1/yang/modules
- Then: 返回硬编码的示例模块列表（huawei-ifm + huawei-vlan）

### BR-03: Schema-已知模块

- Given: 请求模块名为 "huawei-ifm" / "Interfaces" / "huawei-vlan" / "VLANs"
- When: 调用 GET /api/v1/yang/schema/:module
- Then: 返回预定义的完整Schema（含fields和listCols）

### BR-04: Schema-未知模块

- Given: 请求模块名不匹配任何预定义模块
- When: 调用 GET /api/v1/yang/schema/:module
- Then: 返回通用Schema（仅含name+description两个字段）

## 数据模型

### YangModuleInfo 示例

```json
{
  "name": "huawei-ifm",
  "title": "华为接口管理",
  "vendor": "huawei",
  "path": "/ifm",
  "description": "Network interfaces configuration",
  "type": "1"
}
```

### YangSchema 示例（huawei-vlan）

```json
{
  "module": "huawei-vlan",
  "title": "华为 VLAN 配置",
  "vendor": "huawei",
  "fields": [
    {"path": "vlanId", "type": "number", "label": "VLAN ID", "required": true, "minimum": 1, "maximum": 4094, "group": "基本信息"},
    {"path": "vlanName", "type": "string", "label": "VLAN 名称", "placeholder": "例如: VLAN-100", "group": "基本信息"},
    {"path": "description", "type": "string", "label": "描述", "group": "基本信息"},
    {"path": "portList", "type": "string", "label": "端口列表", "placeholder": "例如: GE0/0/1,GE0/0/2", "group": "端口配置"}
  ],
  "listCols": [
    {"path": "vlanId", "type": "number", "label": "VLAN ID"},
    {"path": "vlanName", "type": "string", "label": "VLAN 名称"}
  ]
}
```