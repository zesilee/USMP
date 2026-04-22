# OpenConfig VLAN 管理功能 - 需求设计文档

## 功能概述

基于 `yang-controller-runtime` 框架开发 VLAN 配置管理控制器，使用 **OpenConfig VLAN YANG 模型**，实现交换机 VLAN 的声明式配置管理。

### 功能目标

- 用户在 desired state 中定义 VLAN 配置（VLAN ID、名称、状态、接口关联）
- 控制器自动比对 desired ↔ actual，将差异推送到交换机
- 支持周期轮询，保持配置一致性

### YANG 模型来源

- `openconfig-vlan.yang` - OpenConfig VLAN 主模型
- `openconfig-vlan-types.yang` - VLAN 类型定义
- 来源: [openconfig/public](https://github.com/openconfig/public/tree/master/release/models/vlan)

## YANG 模型结构

```
openconfig-vlan
└── vlans
    └── vlan[vlans-id]  (list)
        ├── id          (leaf) - VLAN ID (1-4094)
        ├── name        (leaf) - VLAN 名称
        ├── status
        │   ├── admin-status (leaf) - UP/DOWN
        │   └── oper-status  (leaf) - ACTIVE/INACTIVE
        └── tagged-ports
            └── port[port-id]  (list) - tagged 端口列表
        └── untagged-ports
            └── port[port-id]  (list) - untagged 端口列表
```

## 功能需求

### 必须实现

1. **VLAN 基础配置**
   - 创建/删除 VLAN
   - 修改 VLAN 名称
   - 设置 admin-status (UP/DOWN)

2. **端口关联**
   - 添加 tagged 端口到 VLAN
   - 添加 untagged 端口到 VLAN
   - 移除端口从 VLAN

3. **生命周期**
   - 周期轮询（默认 5 分钟）从交换机同步实际配置
   - 自动 reconcile 差异，保持 desired ↔ actual 一致

4. **错误处理**
   - 设备离线自动重试（指数退避）
   - 配置下发失败返回明确错误信息

### 可选扩展（第二期）

- VLAN 中继配置
- Q-in-Q 支持
- private VLAN

## 架构设计

遵循 `yang-controller-runtime` 标准架构：

```
Manager
  └── VlanController
        ├── Source: PeriodicSource (5分钟轮询所有设备)
        ├── Predicate: PathPrefix("/vlans")
        ├── Reconciler: VlanReconciler
        │   ├── Get desired from ConfigStore
        │   ├── Get actual from device
        │   ├── Diff with diff engine
        │   └── Apply changes via client.Set()
        └── Queue: RateLimitingQueue with exponential backoff
```

### 依赖

- `pkg/yang-runtime/*` - 框架核心
- `openconfig-vlan.yang` 生成的 Go 结构体 (ygot)
- 设备连接池提供 NETCONF/gNMI 客户端

## 迭代计划

严格遵循小步迭代，**每次 ≤ 500 行，完成即提交**。

### 迭代 1: 生成 Go 结构体 (ygot)
- 配置 go generate 生成 openconfig-vlan Go 结构体
- 验证代码生成成功
- 约 100 行（go.mod + generate 注释）

### 迭代 2: 实现 VlanReconciler 骨架
- 创建 `internal/controller/vlan/reconciler.go`
- 定义 `VlanReconciler` 结构体
- 实现 `Reconcile()` 接口骨架
- 约 100 行

### 迭代 3: 实现 desired state 读取
- 从 ConfigStore 读取 desired VLAN 配置
- 转换为 ygot 结构体
- 约 150 行

### 迭代 4: 实现 actual state 获取
- 从设备读取实际 VLAN 配置
- 通过 client.Get() 获取 /vlans
- 解析为 ygot 结构体
- 约 150 行

### 迭代 5: 集成 diff engine 并应用变更
- 调用 diff engine 计算差异
- 构建 client.Change 列表
- 调用 client.Set() 推送变更到设备
- 处理错误和重试
- 约 200 行（拆分，实际可能两次迭代）

### 迭代 6: 创建 Controller 注册到 Manager
- 在 `main.go` 中创建并注册 VlanController
- 配置 PeriodicSource 和 Predicate
- 验证编译通过
- 约 50 行

### 迭代 7: 单元测试
- 编写 VlanReconciler 单元测试
- 覆盖正常/错误场景
- 约 200 行

总计: 约 950 行 → 拆分 7 个迭代，每个都 ≤ 500 行 ✅

## 验收标准

1. 编译通过，无错误警告
2. 所有单元测试通过
3. 能够正确：
   - 创建 VLAN
   - 修改 VLAN 名称
   - 删除 VLAN
   - 添加/移除端口到 VLAN
   - 自动 reconcile 周期轮询差异
4. 符合 `yang-controller-runtime-dev` 开发规范

## 参考

- [OpenConfig VLAN Model](https://github.com/openconfig/public/blob/master/release/models/vlan/openconfig-vlan.yang)
- [yang-controller-runtime 开发技能](.claude/skills/yang-controller-runtime-dev.md)
