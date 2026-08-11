---
name: retire-ygot-runtime
description: 发布二进制去 ygot/goyang 依赖的 7 阶段计划：阶段1已交付（Schema IR 自立），做后续阶段/碰 schema 加载链路前必读
metadata:
  type: project
---

# retire-ygot-runtime（进行中，2026-08-11 启动）

**目标**：`usmp-backend` 发布二进制 import 闭包零 openconfig/ygot、零 goyang（商用自主可控），参考 K8s runtime.Object/Scheme 范式（极小标记接口 + Scheme 注册 + 构建期生成样板；**不**引入 apimachinery 承载 YANG 类型）。change: `openspec/changes/retire-ygot-runtime`（7 阶段 26 任务）。

**阶段1 已交付并合入 main**（PR#307 提案三件套 + PR#308 代码，2026-08-11）：
- Schema IR 自有格式 v1：`schema.EncodeIR/DecodeIR`（gzip JSON、版本快速失败、确定性、key/parent 指针同一性重建）。IR blob 223KB `internal/yangschema/schema.ir.gz`（go:embed），`yangschema.Load()` 已切换。
- `tools/schemagen`：一期刻意复用旧链路（generated Schema()→ygotbridge）保证零漂移；二期自研生成器落地后切直读 YANG 源。
- `tools/ygotbridge`：原 `schema/entry.go` goyang→内部模型转换整体迁出（8 个 entry 测试随迁，alias_test.go 别名保正文零改动）；`schema/loader.go`+manager SchemeDir 是死代码已删。
- **对拍即新鲜度门禁**：`TestIRBlobMatchesLegacyChain` 字节比对 bridge(generated)==入库 blob——generated schema 变更未重跑 `go generate ./internal/yangschema` 即红。
- 实证：`go list -deps ./pkg/yang-runtime/schema` 零 openconfig；main 闭包还剩 6 个 openconfig 包（阶段2-6 清零）。

**Why**：商用交付自主可控 + 解除 ygot v0.29.20 天花板债（[[go-122-pin]]）。

**阶段2 基本完成**（2026-08-11，任务2.1-2.4 ✓，剩 2.5 make 接线）：object 运行库 + `tools/yanggen` 全链交付；**native 包已入库并存**（`internal/generated/native/{huawei,business}`，9.3 万行、零 openconfig、两次生成字节一致）；**结构对拍三层全绿**（`tools/yanggen/parity_test.go`：类型集/逐字段 tag·形状/枚举值表 vs ygot 基准零差异）。生成约定权威=`openspec/changes/retire-ygot-runtime/codegen-conventions.md`。**对拍实证冻结的四条规则**（改生成器前必读）：①生成域=整个依赖闭包（未列出的被 import 模块顶层容器也入 Device）；②typedef 枚举按**使用方叶所属模块**命名（非定义模块——row-status 每模块一份）；③内联枚举按 AST 节点去重（grouping 复用单枚举首实例命名）；④union 同型折叠（全成员同 Go 型→裸类型，265 叶折叠后仅 6 接口）。踩坑：goyang EnumType Names()/Values() 各自排序不可 zip（NameMap 权威）；净化 token 无尾下划线；leafref 须按**数据树**语义解析（choice/case 不占层级，Entry.Find 会错位）；跨模块 leafref 需全闭包顶层容器索引。

**阶段3 进行中**（2026-08-11）：3.1 生成式 JSON 方法已交付（json_emit.go：每 struct MarshalJSON/UnmarshalJSON + 每 union helper；object 包 json.go 原语）；3.2 首轮双路径对拍绿（json_parity_test：vlan/ifm/bgp/system 语义 DeepEqual + 拒绝性一致）。**关键行为冻结（对拍实证）：ygot 现网 EmitJSON 未开 AppendModuleName → JSON 键永不带模块前缀（含 Device 顶层与 augment 边界），解码双形态兼容**。剩：3.2 扩样本面、3.3 切 driver 注册表/api 调用点、3.4 e2e。踩坑：零字段容器 unused raw、bool 键列表排序、ygot 拒绝性=未知键报错口径一致。

**How to apply**（后续阶段注意）：
- 阶段2 自研生成器：结构约定必须字节级冻结（字段名/path+module tag/map-list/...Key），diff/xmlcodec/drivers 才免改；JSON 编解码走「构建期生成 per-type MarshalJSON/UnmarshalJSON」而非运行时反射引擎（风险局部化、逐类型 golden 可对拍）。
- 每阶段"并行→对拍绿→切换→删旧"；旧 `Validate()` 仅 intent/cr.go 在用；xmlcodec 的 yang.Entry 依赖点=drivers/huawei.go 的 `huawei.SchemaTree[...]`（阶段4 改读 IR）。
- 流程坑：commit-msg 钩子 ≤500 行/commit（测试+实现超限时拆「实现+冒烟」/「完整矩阵」两 commit）；pr-size >20 文件按 Infrastructure 限 3000、≤20 文件限 1000——**分支线性排列 docs 在前，可推前缀 commit 当独立 docs PR** 规避混算；pre-push 钩子跑全量 -race 数分钟，push 须后台+长超时。
