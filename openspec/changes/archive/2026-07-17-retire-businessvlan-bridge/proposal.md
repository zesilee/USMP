# retire-businessvlan-bridge — 旧 BusinessVlan/BusinessInterface CRD 桥接退役

## Why

业务意图控制器（`internal/intent`，BusinessVlanService CR）已接管意图面并稳定运行（business-network-config 已全量交付归档，2026-07-16），旧 BusinessVlan/BusinessInterface CRD 桥接（`internal/crdsource`）按「渐进替换：并行→切换→删除」军规只剩最后一步——物理删除。旧桥接及其依赖链（`api/biz/v1` 类型、`pkg/translator` 意图翻译、`pkg/yang-runtime/actor`）继续留存会持续产生维护噪音（编译、测试、race 门禁都在跑死代码），且 `biz.usmp.io` 组下新旧 Kind 并存徒增认知负担。

**前置已确认（仓库证据）**：现行部署链 `scripts/kind-deploy.sh` 只安装 `deploy/crds/`（devices/auditrecords/businessvlanservices），旧 BusinessVlan/BusinessInterface CRD manifest 从未进 `deploy/crds/`（`crdsource/register_envtest_test.go` 亦以此为前提验证降级路径）；旧 CRD 清单仅存于已退役的 Stack A 目录（`backend/config/crd/bases/`、`backend/deploy/scripts/`）。无现行环境在用旧 CR。

## What Changes

- **删除旧桥接**：`internal/crdsource` 整包删除；`backend/main.go` 摘除 `RegisterIntentSources` 调用与「并行保留」注释。**前置**：`crdsource.StartCache` 被新意图 cache 复用（main.go:145），先迁移到 `internal/intent`。
- **devices-api 厂商门禁换后端**：`device_handler.go` 的 vendor 校验（BR-04）从 `translator.VendorFromString/IsVendorSupported` 切换到 `pkg/yang-runtime/driver` 注册表查询（新增大小写无关的 vendor-supported 查询）。**行为不变**：huawei 通过、未知厂商 400 拒绝。
- **删除意图翻译引擎**：`pkg/translator` 整包删除（唯一调用方是 crdsource；vendor 门禁职责移交 driver 注册表）。**BREAKING（仅内部 API）**：translation-engine 能力退役。
- **删除旧 CRD 类型**：`backend/api/biz/v1` 整包删除（BusinessVlan/BusinessInterface/BusinessRoute/BusinessSwitch）。新意图控制器自定义 scheme，不受影响。
- **删除 Stack A 残留载体**：退役 B0 e2e 测试（`backend/test/e2e`，build-tag 隔离、不进门禁）、旧 CRD 清单（`backend/config/crd/bases/` 中 biz.usmp.io 四件 + samples）、`backend/hack/crd-injector` 与 backend/Makefile 对应 Stack A target。
- **actor 包物理删除**：`pkg/yang-runtime/actor` 零外部引用（本次审计实证），随本 change 一并物理删除，勾销 arch-optimization-roadmap 机械债。
- **TM04 门禁适配**：actor 包 4718 行无法在不破坏包内编译的前提下拆成 ≤1000 行 PR；为纯删除型退役 PR 增设 pr-size 豁免（insertions 极小、deletions 为主时放宽上限）。**契约变更，需在 PR review 中显式确认**。

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `device-driver-registry`: 新增 vendor-supported 查询契约（大小写无关、并发安全、未注册返回 false 不 panic）。
- `translation-engine`: 能力整体退役——全部 requirements REMOVED，主 spec 转为历史契约（同 business-crd/actor-transaction 先例）。
- `system-architecture`: SC-01 中「actor 残留代码属待物理删除机械债」表述随物理删除达成而修订。

> 另一：`devices-api` BR-04 的行为契约（缺省 huawei、大小写无关、无已注册驱动的厂商 400 拒绝）**不变**，其条文本就不指名解析后端，故不出 delta——后端由 translator 注册表换为 driver 注册表属实现细节，由既有 B3 测试兜底等价性。
> 另二：`business-crd`、`actor-transaction` 两份既有 LEGACY 历史契约 spec 的 Purpose 横幅在 sync 时更新「代码载体已物理删除」事实（无 requirement 变更，不出 delta）。

## Impact

- **代码**：删除 `backend/internal/crdsource/`（509 行）、`backend/pkg/translator/`（~1340 行）、`backend/api/biz/v1/`（856 行）、`backend/pkg/yang-runtime/actor/`（4718 行）、`backend/test/e2e/`（693 行）；修改 `backend/main.go`、`backend/internal/intent/`（接收 StartCache）、`backend/internal/api/device_handler.go`、`backend/pkg/yang-runtime/driver/registry.go`（+vendor 查询）。
- **清单/工具**：删除 `backend/config/crd/bases/biz.usmp.io_business{vlans,interfaces,routes,switches}.yaml`、`backend/config/samples/biz_v1_*.yaml`、`backend/hack/crd-injector/`；裁剪 `backend/Makefile`、`backend/deploy/scripts/{one-click-deploy,verify-deployment}.sh` 中旧 CRD 引用。
- **CI**：`.github/workflows/pr-size.yml` 增加纯删除型 PR 豁免（TM04 契约变更）。
- **交付体积**：总删除量 ~8000+ 行，按 retire-stack-a-entry（PR#47-49）先例分 3 波 PR 交付（见 tasks.md）。
- **不受影响**：`internal/intent`（BusinessVlanService，自定义 scheme）、`deploy/crds/`、`tools/crdgen`、生产数据路径（桥接本就与新控制器并行、非唯一路径）。
