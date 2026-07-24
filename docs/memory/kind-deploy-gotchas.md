---
name: kind-deploy-gotchas
description: kind/真集群部署 USMP 前必读：一键脚本入口 + 两个 compose 掩盖不了真集群才炸的坑（旧桥接 CRD 缺失崩溃、非 root 绑 830 需 sysctl）
metadata: 
  node_type: memory
  type: project
  originSessionId: 2ddae1d6-591f-4184-88c3-637c460a037e
---

**入口**：`./scripts/kind-deploy.sh`（WSL 一键：kind 集群 + 三镜像 + CRD/RBAC + backend×2 双选主开关 + HA 验收快照；`down` 清理；宿主端口 8080/3002 与 compose staging 冲突需 `USMP_KIND_STOP_STAGING=1` 或先 staging-down）。

2026-07-16 首次真集群部署实测踩出两个 compose（无集群降级路径）永远掩盖的坑：

1. **旧桥接 CRD 缺失 → mgr.Start Fatalf**（PR#182 已修）：`crdsource.RegisterIntentSources` 旧 BusinessVlan/BusinessInterface 桥接只降级「无集群」，集群可达但 `biz.usmp.io/v1` 旧 CRD 未装（其 manifest 从未进 deploy/crds）时 informer `no matches for kind` 崩溃循环。已改为 RESTMapper 探测缺失即跳过。**教训：任何「集群可达即注册」的源都要同时降级「CRD 未安装」**；也坐实 [[k8s-paas-deployment-constraints]] 里 retire-businessvlan-bridge 应提优先级。

2. **非 root 镜像绑特权端口**（PR#183 已修）：simulator 镜像 `USER usmp` 监听 830——Docker 默认放容器内 `net.ipv4.ip_unprivileged_port_start=0` 所以 compose 能跑，K8s/containerd 不放 → `bind: permission denied`。修法 = Pod `securityContext.sysctls` 放开该 sysctl（K8s ≥1.22 安全白名单，免 kubelet 配置）；加 capability 对非 root 进程不生效，别走弯路。

3. **同 tag 换镜像 apply 不滚动**（PR#184 已修）：`latest` + `imagePullPolicy: Never` 下重跑脚本载入新镜像但 podSpec 不变 → 旧 pod（含崩溃循环旧代码）继续跑。脚本已在 apply 后显式 `rollout restart` 三个 Deployment。**教训：kind load 换镜像后必须 restart，别信 apply**。

**How to apply**：往 K8s 搬任何「compose 里跑得好好的」组件前，先过这两条清单；kind 验收演练五步清单在脚本尾部输出与归档任务文件（openspec/tasks/archive/global-ha-multi-instance.md）。
