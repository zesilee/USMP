# snd-ygot-pipeline — P5-3 ygot 生成管线参数化

## Why

ygot 生成管线目前**不可复现且与产物脱节**：`huawei.go` 的 go:generate 声称输出 `all.gen.go`，实际仓库里是手工拆分的 6 个文件（commit 68731a8）；sed 后处理用 BSD/macOS 语法（`sed -i ''`），Linux 上直接报错——管线只在原开发者的 Mac 上跑通过；跨平台修复器 `fix/fix_enum.go` 已写好却从未接线。同时 CI 的 R04 检查**无条件封杀** `backend/internal/generated/` 的任何 diff，导致合法的重新生成也无法合入 main（门禁死锁）。P5-4 要接入首个非 Huawei 厂商，前提是「加厂商 = 加一条生成配置」，管线必须先参数化、可复现、可合入。

**去风险实证（explore spike，2026-07-09）**：在 Linux 上实跑 ygot v0.34.0（go.mod 锁版）→ 打 enum 补丁 + gofmt 后与已提交 6 文件对比：顶层声明集合完全一致（唯一差异是拆分时手加的 25 行 `var _` 占位符），ySchema gzip 字节 md5 一致。**管线可 100% 复现已提交内容**，本 change 无语义变更风险。

## What Changes

- **`make gen-yang` 厂商 manifest 驱动生成**：每厂商一条生成配置（YANG 模型路径 + 模块列表 + fakeroot 选项），`make gen-yang` 全量重生成、`make gen-yang VENDOR=huawei` 单厂商。自动校验 yang-models submodule 已初始化（缺失时给出可操作报错，R08）。openconfig 包对称纳入同一 target。
- **消灭 BSD sed 四连击**：接线现成 `fix/fix_enum.go` 作为跨平台后处理器（替换 `sed -i ''`），Linux/macOS 均可跑。
- **收敛手拆布局回单 `all.gen.go`**：用真实管线重新生成，删除手拆的 `types.go`/`device.go`/`enums.gen.go`/`ifm.gen.go`/`system.gen.go`/`vlan.gen.go`（已实证语义等价）。
- **R04 门禁进化：冻结 → regen-and-diff 漂移验证**：compliance 的「禁止任何 generated/ diff」改为 CI 重跑 `make gen-yang` 并断言无漂移（对齐 `gen-contract` 漂移门禁先例）——生成物改动合法当且仅当可由管线复现。**本地 pre-commit 钩子对称改造**（T09：L3 层同口径 regen-and-diff，替换同样死锁的本地冻结检查）；顺带修复钩子 R15 段对「包退役提交」的 no-packages 误判。
- **pr-size 排除后端生成物**：`backend/internal/generated/**` 与 `*.gen.go` 计入排除清单（对齐前端生成物排除先例 #60）。
- **删除死管线**：`internal/yang/generate.go`（输出目录空、零消费者）+ 空 `internal/yang/generated/`；**保留** `internal/yang/models/`（是 `generated/openconfig` 的 ygot 输入，后者被 `yangschema/load.go` 生产消费）。
- **修正注释/README 漂移**：huawei.go 示例行的 `-path=yang-modules`（实际 `yang-models`）、README/doc.go 中已不存在的布局描述。

## Capabilities

### New Capabilities
- `yang-codegen-pipeline`: ygot YANG→Go 生成管线——厂商 manifest 驱动的可复现生成（make target + 跨平台后处理），以及生成物漂移 CI 门禁（R04 的可验证形态）。

### Modified Capabilities

（无——本 change 不改任何运行时行为契约；device-driver-registry / yang-xml-codec / translation-engine 的既有 Requirement 均不变，生成物语义等价已实证。）

## Impact

- **构建工具链**：根 `Makefile` 新增 `gen-yang` target；`backend/internal/generated/huawei/huawei.go`（go:generate 重写）、`fix/fix_enum.go`（接线+泛化）、`generated/openconfig/openconfig.go`（对称改造）。
- **生成物**：`backend/internal/generated/huawei/` 6 个手拆文件 → 1 个 `all.gen.go`（约 -4916/+4826 行，纯生成物 churn，语义等价）。编译层面 package 内容不变，**零调用方改动**。
- **CI**：`.github/workflows/compliance.yml`（R04 check 重写为 regen-and-diff，需 checkout yang-models submodule + Go 环境）、`.github/workflows/pr-size.yml`（排除清单扩展）。
- **删除**：`backend/internal/yang/generate.go`、`backend/internal/yang/generated/`。
- **风险**：生成物等价性由既有防线兜底——xmlcodec golden 全量对拍、B2 模拟网元集成测试、全量单测；同 PR 内改门禁+改生成物可行（pull_request 事件使用 PR 分支的 workflow 定义）。
