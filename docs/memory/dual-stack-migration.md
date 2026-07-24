---
name: dual-stack-migration
description: "USMP 代码处于双栈并存的半迁移态——Stack A(K8s CRD+Actor,legacy) 与 Stack B(yang-controller-runtime,R01 权威) 并行运行"
metadata: 
  node_type: memory
  type: project
  originSessionId: 7bbe5a84-45aa-4158-85b4-61c82d05eb0f
---

USMP 后端存在**两套并存、相互冲突的架构栈**,处于未完成迁移的中间态(截至 2026-07-04 仍如此):

- **Stack A — K8s CRD 栈（legacy）**：入口 `backend/cmd/controller/main.go`（controller-runtime/Operator）；用 `api/v1` 旧 CRD + `pkg/yang-runtime/actor/`（Actor 邮箱 + 2PC + 版本快照）+ etcd；控制器在 `backend/controllers/*`。**当前生产入口实际跑的是这套。**
- **Stack B — yang-controller-runtime 栈（R01 权威）**：入口 `backend/main.go`（自研 `manager.New` + Gin :8080）；用 `api/biz/v1`+`api/core/v1` 新 CRD（唯一有生成 CRD YAML）+ `GenericReconciler` 反射 diff + `internal/cache` TTL+LRU；reconciler 在 `internal/controller/{vlan,ifm,system,interfaces}`。

**硬冲突**：R01 强制 Stack B 且明文禁 Actor 模型,但 Actor 子系统是最大且生产在用；R03 禁 DB 但 Stack A 依赖 etcd；`api/v1` 与 `api/biz/v1` **抢注同一 group `biz.usmp.io/v1`** 且 schema 不兼容,无法同 scheme 注册。

**权威裁定**：Stack B 为目标,Stack A 标 legacy 待退役(但尚未迁移)。完整反向还原文档见 `openspec/specs/system-architecture/design.md`(锚点),各子系统 design/spec/tasks 三件套齐全;10 项迁移债/空转件登记在 `system-architecture/tasks.md`(plugin 空转、schema 运行时为空、gNMI Get/Set 空壳、NativeDeviceConfig 下发 TODO 等)。

改动架构或新增控制器前必读该总览,先确认落在哪个栈。相关：[[openspec-brownfield-reverse-fill]]。
