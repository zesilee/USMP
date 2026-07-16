# USMP K8s 部署编排（业务网络配置）

> 背景：USMP 部署在 K8s 内作为 PaaS 底座组件（多实例，禁止本地存储）；业务意图
> 持久化 = K8s CRD（仅当持久化载体 + watch 事件源，Stack B 对账架构不变）。
> 详见 openspec/specs/business-intent-crd、system-architecture（SC-02/SC-06）。

## 目录

| 路径 | 内容 | 生成方式 |
|------|------|----------|
| `crds/businessvlanservices.*` | 业务意图 CRD manifest | **生成物**：`make gen-crd`（源 = `backend/internal/yang/models/usmp-business-vlan.yang`，勿手改，CI regen-and-diff 门禁） |
| `crds/devices.core.usmp.io.yaml` | 设备注册表 CRD（DS-01/04/05） | 手写（非 YANG 源，对齐 `backend/api/core/v1/device_types.go`，两处同步） |
| `rbac/business-intent-rbac.yaml` | ServiceAccount + 意图 CR/status/finalizers + Lease 权限 | 手写 |
| `rbac/device-store-rbac.yaml` | Device CR + 凭据 Secret（namespace 级）权限 | 手写 |

## 安装顺序（BIC-02：CRD 先于应用滚动）

```sh
kubectl apply -f deploy/crds/
kubectl apply -f deploy/rbac/
# 然后再滚动 USMP 新版本
```

## 运行时环境变量

| 变量 | 缺省 | 说明 |
|------|------|------|
| `USMP_INTENT_NAMESPACE` | `default` | 意图 CR、Device CR 与凭据 Secret 所在 namespace（USMP API 代理与 Lease 同域） |
| `USMP_INTENT_LEADER_ELECTION` | 关 | `1` 启用意图控制器 leader election（多副本时仅 leader 执行展开/2PC/清理；BIO-08 接缝，Lease `usmp-business-intent`） |
| `USMP_NATIVE_LEADER_ELECTION` | 关 | `1` 启用原生周期控制器（vlan/ifm/system/bgp/ni）统一 leader election（YR-08：多副本仅 leader 产生周期对账事件，Lease `usmp-native-controllers`，与意图面互不干扰；无集群透传） |
| `USMP_SEED_DEVICE` | 无 | 种子设备 `ip[:port],user,pass[,vendor]`（DS-03，仅无集群内存降级模式生效；集群模式设备集合来自 Device CR，该变量被忽略） |

## 设备注册表（DS-01/04/05）

集群可达时设备连接元信息持久化为 `Device` CR（跨副本 watch 共享、重启恢复），
凭据存同 namespace Secret（CR 仅存 `credentialsSecretRef` 引用，etcd 不落明文）；
无可达集群自动降级进程内存（重启即丢，R08）。

## 版本策略（v1alpha1）

`biz.usmp.io/v1alpha1` 为 storage version；alpha 期允许破坏性变更 + 重建实例，
进 beta 前不引入 conversion webhook（design D1）。

## 无集群降级

无可达 kubeconfig 时业务意图控制器与 `/api/v1/business/*` 自动降级
（日志提示 + API 503 信封），原生配置链路不受影响（BIO-01/R08）。
