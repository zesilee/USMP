---
name: retire-ygot-runtime
description: 发布二进制去 ygot/goyang 依赖的 7 阶段计划：阶段1已交付（Schema IR 自立），做后续阶段/碰 schema 加载链路前必读
metadata:
  type: project
---

# retire-ygot-runtime（进行中，2026-08-11 启动）

**目标**：`usmp-backend` 发布二进制 import 闭包零 openconfig/ygot、零 goyang（商用自主可控），参考 K8s runtime.Object/Scheme 范式（极小标记接口 + Scheme 注册 + 构建期生成样板；**不**引入 apimachinery 承载 YANG 类型）。change: `openspec/changes/retire-ygot-runtime`（7 阶段 26 任务）。

**阶段1 已交付**（worktree-retire-ygot-runtime，2026-08-11）：
- Schema IR 自有格式 v1：`schema.EncodeIR/DecodeIR`（gzip JSON、版本快速失败、确定性、key/parent 指针同一性重建）。IR blob 223KB `internal/yangschema/schema.ir.gz`（go:embed），`yangschema.Load()` 已切换。
- `tools/schemagen`：一期刻意复用旧链路（generated Schema()→ygotbridge）保证零漂移；二期自研生成器落地后切直读 YANG 源。
- `tools/ygotbridge`：原 `schema/entry.go` goyang→内部模型转换整体迁出（8 个 entry 测试随迁，alias_test.go 别名保正文零改动）；`schema/loader.go`+manager SchemeDir 是死代码已删。
- **对拍即新鲜度门禁**：`TestIRBlobMatchesLegacyChain` 字节比对 bridge(generated)==入库 blob——generated schema 变更未重跑 `go generate ./internal/yangschema` 即红。
- 实证：`go list -deps ./pkg/yang-runtime/schema` 零 openconfig；main 闭包还剩 6 个 openconfig 包（阶段2-6 清零）。

**Why**：商用交付自主可控 + 解除 ygot v0.29.20 天花板债（[[go-122-pin]]）。

**How to apply**（后续阶段注意）：
- 阶段2 自研生成器：结构约定必须字节级冻结（字段名/path+module tag/map-list/...Key），diff/xmlcodec/drivers 才免改；JSON 编解码走「构建期生成 per-type MarshalJSON/UnmarshalJSON」而非运行时反射引擎（风险局部化、逐类型 golden 可对拍）。
- 每阶段"并行→对拍绿→切换→删旧"；旧 `Validate()` 仅 intent/cr.go 在用；xmlcodec 的 yang.Entry 依赖点=drivers/huawei.go 的 `huawei.SchemaTree[...]`（阶段4 改读 IR）。
- 流程坑：commit-msg 钩子 ≤500 行/commit（测试+实现超限时拆「实现+冒烟」/「完整矩阵」两 commit）；pr-size >20 文件按 Infrastructure 限 3000、≤20 文件限 1000——**分支线性排列 docs 在前，可推前缀 commit 当独立 docs PR** 规避混算；pre-push 钩子跑全量 -race 数分钟，push 须后台+长超时。
