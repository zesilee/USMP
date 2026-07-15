# USMP K8s 部署编排（业务网络配置）

> 背景：USMP 部署在 K8s 内作为 PaaS 底座组件（多实例，禁止本地存储）；业务意图
> 持久化 = K8s CRD（仅当持久化载体 + watch 事件源，Stack B 对账架构不变）。
> 详见 openspec/specs/business-intent-crd、system-architecture（SC-02/SC-06）。

## 目录

| 路径 | 内容 | 生成方式 |
|------|------|----------|
| `crds/` | 业务意图 CRD manifest | **生成物**：`make gen-crd`（源 = `backend/internal/yang/models/usmp-business-vlan.yang`，勿手改，CI regen-and-diff 门禁） |
| `rbac/` | ServiceAccount + 意图 CR/status/finalizers + Lease 权限 | 手写 |

## 安装顺序（BIC-02：CRD 先于应用滚动）

```sh
kubectl apply -f deploy/crds/
kubectl apply -f deploy/rbac/business-intent-rbac.yaml
# 然后再滚动 USMP 新版本
```

## 运行时环境变量

| 变量 | 缺省 | 说明 |
|------|------|------|
| `USMP_INTENT_NAMESPACE` | `default` | 意图 CR 所在 namespace（USMP API 代理与 Lease 同域） |
| `USMP_INTENT_LEADER_ELECTION` | 关 | `1` 启用意图控制器 leader election（多副本时仅 leader 执行展开/2PC/清理；BIO-08 接缝。注意：存量原生周期控制器尚未纳入选主，全局 HA 另立任务） |

## 版本策略（v1alpha1）

`biz.usmp.io/v1alpha1` 为 storage version；alpha 期允许破坏性变更 + 重建实例，
进 beta 前不引入 conversion webhook（design D1）。

## 无集群降级

无可达 kubeconfig 时业务意图控制器与 `/api/v1/business/*` 自动降级
（日志提示 + API 503 信封），原生配置链路不受影响（BIO-01/R08）。
