# tasks — global-ha-multi-instance

> 四波次对应 design D8，每波次一个 PR（TM04 ≤1000 行）、独立 worktree apply、测试先行（T05/T06）。
> 波次内顺序 = 依赖顺序；W3/W4 与 W2 无代码依赖但串行提交避免 main.go 冲突。

## 1. W1 兜底收敛 + 种子设备迁 env（纯重构，行为不变）

- [x] 1.1 测试设计：`device.ResolveConn` 表格驱动用例（已注册/未注册/空 Vendor/并发 race）+ 9 个 reconciler 与 intent tx、config handler、device API 现有测试面盘点（回归防线确认）——红灯先行（B1）
- [x] 1.2 实现 `device.ResolveConn(store, id)`（DS-06）：未注册 → `{IP, Protocol: AUTO}`+空凭据+统一日志
- [x] 1.3 收敛消费方：9 个 `internal/controller/*/reconciler.go` resolveConn、`intent/tx.go`、`config_handler.go` 读/删、`device_handler.go` 探活/AddDevice 全部改调 helper，删除内联副本
- [x] 1.4 种子设备迁移：删除 `NewDeviceHandler` 硬编码 `192.168.1.1`，实现 `USMP_SEED_DEVICE` 解析（仅内存 store 时生效；未设则空库启动）；补 B1 用例（设/未设/格式错） + 更新受影响的 handler 测试与 E2E 起手数据（compose/staging env 注入种子）
- [x] 1.5 全量验证：`go test ./... -race` 全绿 + `make e2e-local`（涉 staging 行为）→ review → commit → PR-W1

## 2. W2 Device CRD + crdStore + 装配降级

- [x] 2.1 Spike（半天封顶）：Device CRD manifest 生成方式定案（controller-gen vs 扩 crdgen）+ namespace env 复用 `USMP_INTENT_NAMESPACE` 定案（design Open Questions）
- [x] 2.2 类型脚手架：`backend/api/core/v1` 新增 `Device` types（spec: managementIP/port/protocol/timeoutSeconds/vendor/credentialsSecretRef）+ deepcopy 生成 + scheme 注册
- [x] 2.3 测试设计：crdStore 完备矩阵（envtest：Put 双资源落库/写序 Secret 先行/Get 还原凭据/Secret 缺失空凭据降级/Delete 反向/apiserver 拒绝时镜像不变更/watch 跨 client 可见/重启重建镜像/并发 race；无集群降级选择内存版）——红灯先行（B1+envtest 集成，参照 intent envtest 先例）
- [x] 2.4 实现 `device.NewCRDStore`：watch 镜像读路径 + write-through 写穿（DS-01/04/05），凭据 Secret 读写与失效跟随
- [x] 2.5 装配：`manager.WithDeviceStore` option + `main.go` 按 `ctrlcfg.GetConfig()` 成败选 store（复用 crdsource 降级模式）；devices-api 持久化失败 5xx 信封（BR-13）+ B3 用例
- [x] 2.6 部署物料：`deploy/crds/` device manifest、`deploy/rbac/` Device+Secret 权限、README 环境变量表与安装顺序更新
- [ ] 2.7 双路径验证：envtest 全绿 + 无集群模式 `go test ./... -race` 全绿 + `make e2e-local` 全绿 → review → commit → PR-W2

## 3. W3 选主泛化 + 原生面 gate

- [x] 3.1 测试设计：泛化 `leader.GateSources` 用例（开关关透传/无集群透传/仅 leader 启动 inner/失主停止/两 gate 独立 Lease 互不干扰；envtest 双副本仅 leader 入队）——迁移并扩展 `intent/leader_test.go`，红灯先行
- [x] 3.2 提取 `pkg/yang-runtime/leader`：参数化 Lease name/开关 env 的 `GateSources`（YR-08），Lease 参数照搬 intent（15s/10s/2s、ReleaseOnCancel）
- [x] 3.3 原生面接线：`main.go` 5 个周期 source 统一包 `usmp-native-leader gate`（`USMP_NATIVE_LEADER_ELECTION`，缺省关）；intent 改调泛化实现并删除本地副本（行为等价验证）
- [x] 3.4 部署物料：rbac 加 `usmp-native-controllers` Lease、README 环境变量表更新
- [x] 3.5 全量验证：`go test ./... -race` 全绿（含 envtest）→ review → commit → PR-W3

## 4. W4 audit 接口化 + CRD 后端 + 本地文件退役

- [ ] 4.1 测试设计：audit 接口抽取回归（现内存/文件行为）+ CRD 后端矩阵（envtest：写 create CR/失败不阻断/跨 client 可见/重启保留/超限删最旧/并发清理容忍 NotFound/无集群降级内存；`GET /logs` 契约回归 B3）——红灯先行
- [ ] 4.2 `audit.Store` 接口化：现实现拆为内存降级路径（去文件写），`USMP_AUDIT_FILE` 弃用警告（OA-05）
- [ ] 4.3 类型脚手架：`core/v1 AuditRecord` types + deepcopy + scheme + `deploy/crds`/`rbac` 物料
- [ ] 4.4 实现 CRD 后端：create-per-record + label 索引 + watch 镜像 List + 超限清理（OA-01/02/03/04），`manager` 装配按集群可达性选后端
- [ ] 4.5 全量验证：`go test ./... -race` 全绿 + `GET /logs` 前端契约回归（契约生成无漂移）→ review → commit → PR-W4

## 5. 收官

- [ ] 5.1 双路径验证汇总：staging 多副本演练（两副本 + 两开关开 → 单 leader 日志、设备/审计跨副本可见、杀 leader 接管）证据留档
- [ ] 5.2 `/opsx:sync` delta → 主 spec；更新 `openspec/tasks/global-ha-multi-instance.md` 状态与记忆（k8s-paas-deployment-constraints follow-up 勾销本项）
- [ ] 5.3 `/opsx:archive` 归档 change
