# retire-stacka-residue — Stack A 残余收敛：e2e 脚手架 / backend deploy·config 目录 / NativeDeviceConfig 物理删除

## Why

Stack A 代码主链已归零（retire-businessvlan-bridge，PR #186-#191），但三类残余载体仍在仓库：B0 e2e 的集群脚手架与 integration 测试腿、`backend/deploy`·`backend/config` 两个 Stack A 部署清单目录、NativeDeviceConfig 旧 CRD 类型闭包（roadmap D1 挂账）。它们零现行链引用（2026-07-17 审计实证：CI workflows、scripts/kind-deploy.sh、顶层 Makefile 均不消费），继续留存只制造「这还能用吗」的认知成本，且 backend/Makefile 里指向已删目录的 target 会以报错误导。

## What Changes

- **B0 物理删除收官**：删 `backend/test/e2e/`（脚手架残余：run_e2e.sh/kind 配置/simulator yaml/patches）与 `backend/test/integration/`（build-tag integration 的 HTTP e2e 腿）。B0 层 spec 定位「不进任何门禁、不再维护」不变，载体清零。
- **Stack A 部署目录退役**：删 `backend/deploy/`（kind-cluster + manifests + one-click/verify 脚本）与 `backend/config/`（kubebuilder 布局：NDC CRD manifest、samples、manager、rbac）。现行部署链是顶层 `deploy/` + `scripts/kind-deploy.sh`，不受影响。
- **NativeDeviceConfig 收敛（roadmap D1 终局）**：删 `api/core/v1` 的 `nativedeviceconfig_types.go`、`zz_generated.deepcopy.go`（全文件仅 NDC 块）、`types_common.go`（SyncState 零外部引用）；删 `docs/crd/nativedeviceconfig.md` + README 索引。Device/AuditRecord（在用）不动。
- **backend/Makefile 裁剪**：删 e2e-*/kind-*/deploy/one-click/docker-build/install-ginkgo/install-envtest 及指向已删目录的 test-integration/test-vlan/test-device 等 Stack A target；保留纯 Go 开发 target（build/run/test-unit/coverage/lint/clean/help）。
- **文档对齐**：CLAUDE.md §5.6 B0 行标注「已物理删除」。

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `system-architecture`: SC-01「Stack A 载体物理不存在」清单扩展——追加 NativeDeviceConfig 类型、backend/{deploy,config,test/e2e,test/integration} 目录。

> `business-crd`（LEGACY 历史契约）Purpose 横幅在 sync 时补记 NativeDeviceConfig 载体已删；无 requirement 变更不出 delta。

## Impact

- **删除量**：~5000 行（e2e 脚手架 ~1400、integration 550、backend/deploy ~1200、backend/config ~400、NDC Go+docs ~860、Makefile 裁剪 ~230），纯删除型——走已落地的 insertions≤50 豁免档（≤6000）。
- **不受影响**：`backend/simulator/netconfsim`（B2 集成测试与 staging 部署在用）、顶层 `deploy/`、`Makefile`、`api/core/v1` 的 Device/AuditRecord、`*_integration_test.go`（B2，在 pkg 内、非 build-tag integration）。
- **交付**：3 个 PR——①立项制品+文档对齐（插入型）②纯删除③sync+archive。
