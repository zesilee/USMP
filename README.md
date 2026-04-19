# USMP - Universal Switch Management Platform

> 无数据库、高并发、模型驱动的交换机设备管理平台

## 📖 项目简介

USMP 是一个基于 **Actor 模型** + **YANG 模型驱动** 的网络交换机配置管理平台，实现了设备级和配置对象级的故障隔离，完全运行在内存中，**无需任何数据库**。

### 核心设计理念

| 设计原则 | 说明 |
|---------|------|
| **无数据库** | 仅 JSON 存储设备元信息，运行配置全靠 NETCONF 实时拉取 + TTL+LRU 内存缓存 |
| **双层 Actor 架构** | ManagerActor → DeviceActor → YANG Object Actor，故障隔离，高并发 |
| **模型驱动全流程** | YANG → ygot 自动生成 Go 结构体 → 后端暴露模型结构 → 前端自动生成动态表单 |
| **缓存一致性** | 配置下发成功后主动失效对应缓存，下次读取自动拉取最新配置 |
| **NETCONF 标准** | 遵循 RFC6241，支持 get-config/edit-config/commit，断线自动重连 |

## 🏗️ 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                 Vue3 前端 · 自动生成动态表单                    │
└─────────────────────────────────────────────────────────────┘
                              ↓ HTTP REST API
┌─────────────────────────────────────────────────────────────┐
│                     Gin · 后端 API 层                         │
└─────────────────────────────────────────────────────────────┘
                              ↓ 消息路由
┌─────────────────────────────────────────────────────────────┐
│               ManagerActor · 单例设备管理                     │
│  • 注册/查找/销毁 DeviceActor                                  │
│  • 从本地 JSON 加载设备元信息                                  │
└─────────────────────────────────────────────────────────────┘
           ┌───────────────┬───────────────┬───────────────┐
           ↓               ↓               ↓               ↓
┌──────────────────┐┌──────────────────┐┌──────────────────┐
│  DeviceActor 1   ││  DeviceActor 2   ││  DeviceActor N   │  ← 每设备一个
│  (192.168.1.1)   ││  (192.168.1.2)   ││  (192.168.1.N)   │
└──────────────────┘└──────────────────┘└──────────────────┘
           │
           ├───────────────┬───────────────┬───────────────┐
           ↓               ↓               ↓               ↓
┌────────────────────┐┌────────────────────┐┌────────────────────┐
│ YANG Object Actor  ││ YANG Object Actor  ││ YANG Object Actor  │  ← 每YANG对象一个
│ /interfaces        ││ /vlans             ││ /system            │
└────────────────────┘└────────────────────┘└────────────────────┘
           │
           ↓ 缓存未命中 → NETCONF 拉取
┌─────────────────────────────────────────────────────────────┐
│              TTL+LRU · 内存缓存 (无数据库)                      │
│              Key = 设备IP + YANG路径 · TTL自动过期             │
└─────────────────────────────────────────────────────────────┘
           │
           ↓ 配置读写
┌─────────────────────────────────────────────────────────────┐
│              NETCONF · RFC6241 标准协议                        │
│          get-config / edit-config / commit / auto-reconnect    │
└─────────────────────────────────────────────────────────────┘
           │
           ↓ TCP/SSH 连接
┌─────────────────────────────────────────────────────────────┐
│                    物理交换机设备                              │
└─────────────────────────────────────────────────────────────┘
```

## 🛠️ 技术栈

| 层级 | 技术 | 用途 |
|------|------|------|
| 后端语言 | Go 1.21+ | 静态类型，高并发 |
| Actor 模型 | [protoactor-go](https://github.com/asynkron/protoactor-go) | 双层 Actor 架构，故障隔离 |
| Web 框架 | [Gin](https://github.com/gin-gonic/gin) | REST API |
| YANG 代码生成 | [ygot](https://github.com/openconfig/ygot) | YANG → Go 强类型结构体 |
| NETCONF 客户端 | [scrapligo](https://github.com/scrapli/scrapligo) | NETCONF 协议对接 |
| 缓存 | 自研 TTL+LRU | 内存缓存，无外部依赖 |
| 前端 | Vue3 + TypeScript + Element Plus | 动态表单自动生成 |

## 🚀 快速开始

### 编译后端

```bash
git clone https://github.com/leezesi/usmp.git
cd usmp
go mod tidy
go build -o usmp .
```

### 运行

```bash
./usmp
```

默认监听 `0.0.0.0:8080`，API 基础路径 `/api/v1`。

### 启动前端（开发模式）

```bash
cd web
npm install
npm run dev
```

## 📁 项目结构

```
usmp/
├── main.go                          # 程序入口
├── go.mod/go.sum                    # Go 模块定义
├── internal/
│   ├── actor/
│   │   ├── manager.go             # ManagerActor 顶层设备管理
│   │   ├── manager_test.go
│   │   ├── device.go              # DeviceActor 单设备实例
│   │   ├── device_test.go
│   │   ├── yang_object.go         # YANG Object Actor 基类
│   │   ├── yang_object_test.go
│   │   ├── messages.go            # 所有 Actor 消息定义
│   │   └── types.go               # 公共类型导出
│   ├── cache/
│   │   ├── ttl_lru.go             # TTL+LRU 内存缓存实现
│   │   └── ttl_lru_test.go
│   ├── netconf/
│   │   ├── client.go              # NETCONF 客户端
│   │   ├── session.go             # 会话管理 + 自动重连
│   │   ├── xml_convert.go         # ygot ↔ XML 双向转换
│   │   └── messages.go           # NETCONF XML 消息模板
│   ├── types/
│   │   └── types.go               # 公共类型定义（解决导入循环）
│   ├── yang/
│   │   ├── generate.go            # ygot 代码生成脚本
│   │   ├── models/                # YANG 源文件
│   │   └── generated/             # 自动生成的 Go 结构体
│   ├── api/
│   │   ├── server.go              # Gin 服务器初始化
│   │   ├── device_handler.go      # 设备相关 API
│   │   ├── config_handler.go      # 配置读写 API
│   │   ├── yang_handler.go        # YANG 模型 API
│   │   └── response.go            # 统一 API 响应格式
│   └── config/
│       └── devices.json           # 设备元信息存储（初始为空）
├── web/                            # Vue3 前端项目
│   ├── package.json
│   ├── vite.config.ts
│   ├── index.html
│   └── src/
│       ├── main.ts
│       ├── App.vue
│       ├── api/index.ts           # API 请求封装
│       ├── types/yang.ts          # 前端类型定义
│       └── components/
│           ├── DeviceTree.vue     # 设备+YANG 树形菜单
│           ├── DynamicForm.vue    # 动态表单组件
│           └── YangNodeRenderer.vue # YANG 节点渲染器
└── README.md
```

## 🌐 API 端点

### 设备管理

| 方法 | 路径 | 功能 |
|------|------|------|
| GET | `/api/v1/devices` | 列出所有设备 |
| POST | `/api/v1/devices` | 添加设备 |
| DELETE | `/api/v1/devices/:ip` | 删除设备 |
| GET | `/api/v1/devices/:ip/status` | 获取设备状态 |

### 配置读写

| 方法 | 路径 | 功能 |
|------|------|------|
| GET | `/api/v1/config/:ip/:path` | 获取配置（自动缓存）|
| POST | `/api/v1/config/:ip/:path` | 下发配置 |

### YANG 模型

| 方法 | 路径 | 功能 |
|------|------|------|
| GET | `/api/v1/yang/modules` | 列出所有支持的 YANG 模块 |

## ✅ 测试

运行所有单元测试：

```bash
go test ./... -v
```

当前测试状态：

```
ok  	github.com/leezesi/usmp/internal/cache	0.767s
ok  	github.com/leezesi/usmp/internal/actor	0.719s
```

## 🎯 核心特性

### 1. 故障隔离
- 一台设备故障不影响其他设备
- 一个 YANG 配置对象故障不影响其他对象
- Actor 模型天然隔离，避免跨设备并发竞态

### 2. 无数据库
- 无需安装运行 MySQL/Redis 等任何数据库
- 设备元信息存在本地 JSON 文件
- 运行配置全在内存，重启自动从设备重新拉取

### 3. 自动重连
- NETCONF 连接断开后自动指数退避重连
- 不丢失配置请求，排队等待重连完成

### 4. 动态表单
- 前端完全根据 YANG 模型自动生成表单控件
  - `boolean` → Switch 开关
  - `enumeration` → Select 下拉框
  - `int`/`uint` → Input 数字框
  - `string` → Input 文本框
  - `list` → Table 表格
- 无需手写表单，新增 YANG 模块自动生效

## 📝 开发流程

本项目严格遵循 TDD 测试驱动开发：

1. `Plan` → 需求拆分，每个迭代一个原子功能
2. `Test` → 先写单元测试
3. `Code` → 实现功能（单次 ≤ 500 行）
4. `Review` → 自动代码评审
5. `Commit` → 标准 What/Why/Commit 提交信息

## 📄 License

MIT License

## 🙏 致谢

- [protoactor-go](https://github.com/asynkron/protoactor-go) - Actor 模型实现
- [openconfig](https://github.com/openconfig) - YANG 模型标准
- [ygot](https://github.com/openconfig/ygot) - YANG Go 代码生成
- [scrapligo](https://github.com/scrapli/scrapligo) - NETCONF 客户端
