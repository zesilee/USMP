# retire-stacka-residue — 设计

## Context

2026-07-17 审计（main@3711232）：

- `backend/test/e2e/`（Go 测试已随 PR #187 删除，余脚手架 6 文件）与 `backend/test/integration/`（537 行 build-tag HTTP e2e + README）= B0 两条腿的载体残余；`backend/Makefile` 101 处 e2e/kind 引用大多指向它们。
- `backend/deploy/`（25 文件）与 `backend/config/`（7 文件）：Stack A 时代 kind 部署与 kubebuilder 布局；grep CI workflows / scripts / 顶层 Makefile / docs（除历史 plan 文档）零消费。
- NativeDeviceConfig 闭包：`nativedeviceconfig_types.go`(89) + `zz_generated.deepcopy.go`（全文件 11 处引用全是 NDC；Device/AuditRecord 用手写 deepcopy 文件）+ `types_common.go`（SyncState 全仓零外部引用）+ CRD yaml(149) + sample(66) + docs(508)。

## Goals / Non-Goals

**Goals:** 上述载体物理清零；backend/Makefile 不留指向已删目录的死 target；每 PR 过现行门禁。

**Non-Goals:** 不动 netconfsim（B2/staging 在用）、顶层 deploy 链、Device/AuditRecord 类型；不重建任何 e2e 集群能力（现行 E2E = envtest + 顶层 e2e-local/staging smoke）。

## Decisions

### D1 integration 腿一并删除

CLAUDE.md §5.6 B0 行明示 `backend/test/{e2e,integration}` 同属退役层；integration 的 HTTP e2e 场景已被 B2（包内 `*_integration_test.go` + netconfsim）与 F4（staging smoke）覆盖。**备选**保留 integration 观察——它依赖手工起 8080 服务器（test-server 进程泄漏坑的来源之一），留着有害无益。

### D2 backend/config 整目录删除而非只删 NDC 件

目录内全部 7 件都是 Stack A kubebuilder 布局（manager.yaml 部署的 cmd/controller 已删、rbac 绑定其 SA、crd/bases 只剩 NDC、samples 只剩 NDC）。逐件删会留一个空壳目录结构。

### D3 zz_generated 与 types_common 随 NDC 整删

zz 文件 11 处符号全是 NDC（Device/AuditRecord 的 deepcopy 在各自手写文件）；SyncState 零外部引用。`groupversion_info.go` 保留（Device/AuditRecord 注册）。

### D4 backend/Makefile 裁剪保守边界

只删两类：引用已删路径的 target（test-integration/test-vlan/test-device/e2e-*/kind-*/deploy/one-click、install-ginkgo/install-envtest、integration-setup/check-server 若引用 e2e）与 docker-build*（构建现走顶层 staging/kind 链）。保留纯 Go 开发便捷 target。裁剪后 `make -C backend help` 须仍可用。

### D5 三 PR 交付

①立项制品 + CLAUDE.md/README 对齐（插入型，≤1000）→ ②纯删除（~5000 行，insertions≤50 豁免档）→ ③sync+archive。②依赖①先合（否则②的 PR diff 含制品插入超出豁免条件）。

## Risks / Trade-offs

- **[风险] 某开发者本地流程仍用 backend/Makefile 的 kind/e2e target** → 这些 target 指向的目录本 change 删除后必然报错；个人项目、现行文档（TEAM_HANDBOOK/README/CLAUDE.md）均指向顶层链，接受。
- **[风险] docker-build target 仍被隐性使用** → grep CI/scripts/docs 零引用；staging/kind 构建走顶层 compose 与 kind-deploy.sh 内嵌 docker build。
- **[取舍] 历史 plan 文档（docs/superpowers/plans/2026-05-04-*）引用将变死链** → 历史计划文档忠实记录当时状态，不改写。

## Migration Plan

部署零迁移（删除物均不在现行链）。回滚 = revert 对应 PR。

## Open Questions

（无）
