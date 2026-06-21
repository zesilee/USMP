# OpenSpec Brownfield 反向补齐 - REST API 层设计

## 背景

USMP 项目已有完整代码实现但缺少规范文档。`openspec/specs/` 目录为空，现有 spec 散落在 `spec/` 和 `docs/superpowers/specs/` 中且不遵循 OpenSpec 格式。需要对已有代码做事实还原，补齐行为契约、架构设计、补全清单。

## 核心原则（Brownfield 渐进式）

1. **不修改现有代码**，只做事实还原
2. **每次只补一个模块/接口/领域**
3. **输出三件套**：spec.md（行为契约）、design.md（架构设计）、tasks.md（补全清单）
4. **Given-When-Then** 描述业务规则
5. **接口包含**：路径、方法、参数、响应、错误码
6. **数据模型包含**：字段、类型、约束、示例

## Change 分解（方案 A：按业务域分组）

| Change | 覆盖端点 | 核心业务 |
|--------|----------|----------|
| **devices-api** | `GET/POST /api/v1/devices`, `DELETE /api/v1/devices/:ip`, `GET /api/v1/devices/:ip/status` | 设备注册/删除/状态查询 |
| **config-api** | `GET/POST /api/v1/config/:ip/*path` | 按YANG路径读写设备配置 |
| **yang-api** | `GET /api/v1/yang/modules`, `GET /api/v1/yang/schema/:module` | YANG模块列表与Schema查询 |
| **sse-api** | `GET /watch/:device/:module` (SSE) | 配置变更实时推送 |

执行顺序：devices-api → config-api → yang-api → sse-api（前者是后者依赖基础）

## 输出物格式

### 存放位置

```
openspec/specs/
├── devices-api/
│   ├── spec.md
│   ├── design.md
│   └── tasks.md
├── config-api/
│   ├── spec.md
│   ├── design.md
│   └── tasks.md
├── yang-api/
│   ├── spec.md
│   ├── design.md
│   └── tasks.md
└── sse-api/
    ├── spec.md
    ├── design.md
    └── tasks.md
```

### spec.md 模板

```markdown
# {change-name} - 行为契约

## 接口定义

### {METHOD} {PATH}

**描述**：一句话说明

**参数**：
| 参数 | 位置 | 类型 | 必填 | 约束 | 示例 |
|------|------|------|------|------|------|

**响应**：
| 状态码 | 含义 | Body |
|--------|------|------|

## 业务规则

### {规则名称}

- Given: 前置条件
- When: 触发动作
- Then: 期望结果

## 数据模型

### {ModelName}

| 字段 | 类型 | 约束 | 示例 |
|------|------|------|------|

**示例**：
```json
{ ... }
```
```

### design.md 模板

```markdown
# {change-name} - 架构设计

## 请求处理流程
（从代码还原：handler → service → cache/NETCONF → response）

## 依赖关系
（该API依赖的内部模块、缓存、外部协议调用）

## 错误处理策略
（当前代码中的错误处理方式，不评价、不修改）
```

### tasks.md 模板

```markdown
# {change-name} - 补全清单

## spec 与代码差异
- [ ] {差异项}：spec 描述了 X 行为，代码中实际行为是 Y

## 后续改进建议
- [ ] {可选改进项}
```

## 首个 Change 详细设计：devices-api

### 覆盖端点

| 端点 | Handler | 说明 |
|------|---------|------|
| `GET /api/v1/devices` | `ListDevices` | 返回所有已注册设备 |
| `POST /api/v1/devices` | `AddDevice` | 注册新设备 |
| `DELETE /api/v1/devices/:ip` | `RemoveDevice` | 删除设备 |
| `GET /api/v1/devices/:ip/status` | `GetStatus` | NETCONF连通性探测 |

### 预估业务规则：8-10条

- 设备列表返回格式与空列表处理
- 设备注册：IP唯一性校验、必填字段校验、重复注册处理
- 设备删除：存在性校验、删除后缓存清理
- 设备状态：在线/离线判定、NETCONF连接探测、超时处理

### 预估数据模型：2个

- **DeviceInfo**：IP、端口、用户名、密码（脱敏）
- **DeviceStatus**：在线状态、延迟、错误信息

### 依赖关系

- `POST /devices` → 写入 JSON 元信息文件
- `GET /devices/:ip/status` → 触发 NETCONF 连接探测
- `DELETE /devices/:ip` → 清理连接池 + 失效缓存

### 待确认差异项

- 设备凭据是否在 GET 响应中脱敏
- 重复注册是覆盖还是报错
- 删除设备时连接池/缓存的清理行为
