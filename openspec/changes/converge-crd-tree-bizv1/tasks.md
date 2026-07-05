# converge-crd-tree-bizv1 — tasks（D1 收敛到 api/biz/v1）

> TDD（R06）：先补 translator 单测锁定新映射，再改。单 commit ≤500、PR ≤800。

## 1. translator + crdsource 原子迁移（一 PR）

- [ ] 1.1 先写测试：`pkg/translator` 单测——VLAN(biz/v1: MacLearning/BroadcastDiscard/UnknownMulticastDiscard/AdminStatus→huawei ygot)、Interface(biz/v1: IfName/MTU/access|trunk|hybrid→L2 ygot)、错误路径
- [ ] 1.2 重写 `huawei_vlan.go`：bizv1 导入→api/biz/v1；映射 biz/v1 字段；删 Type/convertVlanType/MacLearningEnabled/StatisticEnabled/BroadcastDiscardEnabled/TaggedPorts 相关
- [ ] 1.3 重写 `huawei_interface.go`：IfName/TrunkVlans；删 L3/IpAddress/Netmask/NativeVlan/Speed/Duplex 分支；模式收敛 L2
- [ ] 1.4 `huawei.go`：bizv1 导入→api/biz/v1；Route map 用 biz/v1 字段（Destination/NextHop/Preference/Description/BfdEnabled）；`translator.go` 注释
- [ ] 1.5 crdsource：`businessvlan.go`/`businessinterface.go`/`register.go`（AddToScheme/VlanObject/InterfaceObject）+ 测试 → api/biz/v1
- [ ] 1.6 `go build ./...` + `go test ./...` 全绿

## 2. 删除 api/v1（随后 PR，迁移后零引用）

- [ ] 2.1 grep 确认 `api/v1` 无非自身引用
- [ ] 2.2 删除 `api/v1/*`（6 文件；NativeDeviceConfig 死类型，真身 api/core/v1）
- [ ] 2.3 `go build ./...` + `go test ./...` 绿

## 3. 收尾

- [ ] 3.1 `system-architecture/tasks.md` 勾除 D1
- [ ] 3.2 满足 R04（ygot 映射对齐生成模型）/R06（TDD）
