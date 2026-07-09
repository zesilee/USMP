# tasks — snd-ygot-pipeline

> TDD（T01/T05）：每组先写测试/验证证据再实现。手写代码单 commit ≤500 行；生成物 churn 由 pr-size 排除（任务 1.1 先行）。测试层（§5.6）：构建管线改动 → B1（genfix 表格驱动）+ CI 门禁自验证 + 全量既有测试（xmlcodec golden / B2）作为生成物等价防线。

## 1. 门禁先行（D4/D5）

- [x] 1.1 pr-size.yml 排除 `backend/internal/generated/**/all.gen.go`（先例 #60），commit
- [x] 1.2 compliance.yml「Check Generated Code (R04)」重写为 regen-and-diff：相关路径变更时 init yang-models submodule + `make gen-yang` + `git diff --exit-code backend/internal/generated/`；无关 PR 跳过（CG-03 三场景），commit（依赖 2.x 的 make target 存在——本任务可与 2.x 同 commit 或紧随其后）

## 2. 管线实现（D1/D2，CG-01/CG-02）

- [x] 2.1 【spike，R3】openconfig 包 regen 对比：跑 ygot 生成对比现 `generated/openconfig/all.gen.go`；零 diff → 直接纳入；有 diff → 按 design R3 处置路径决策并记录
- [x] 2.2 【测试先行】`scripts/genfix` B1 表格驱动测试（红）：`|`→`_OR_` 修复正常/FlexE 变体/YANG 原值字符串不动/无匹配幂等/二次执行 no-op
- [x] 2.3 实现 `scripts/genfix/main.go`（泛化自 fix/fix_enum.go，去 Huawei 绑定），测试绿；删除 `backend/internal/generated/huawei/fix/`
- [x] 2.4 编写 `gen.conf`×2（huawei / openconfig）+ `scripts/gen-yang.sh`（扫描 gen.conf → ygot generator → genfix → gofmt；submodule 前置校验含修复指引）+ 根 Makefile `gen-yang` target（含 `VENDOR=` 单包）
- [x] 2.5 删除 go:generate 事实源：`generated/huawei/huawei.go`、`generated/openconfig/openconfig.go`（build-tag generate 文件）；删除死管线 `internal/yang/generate.go` + 空 `internal/yang/generated/`（保留 `internal/yang/models/`）
- [x] 2.6 更新 `generated/README.md` + `huawei/doc.go`：布局（单 all.gen.go）、入口（make gen-yang）、加厂商步骤（加目录+gen.conf）；修正 yang-modules→yang-models 漂移

## 3. regen 收敛（D3）

- [x] 3.1 `make gen-yang` 重出 `huawei/all.gen.go`，删除手拆 6 文件（types/device/enums/ifm/system/vlan）；openconfig 按 2.1 结论处置
- [x] 3.2 等价性验证证据：`go build ./...` + 全量 `go test ./...`（含 xmlcodec golden、B2 需非 short）全绿；声明集合 diff + ySchema md5 对比记录进 PR 描述
- [x] 3.3 重复执行 `make gen-yang` 验证零漂移（CG-01 场景 1 = CI 门禁的本地预演）

## 4. 收尾

- [x] 4.1 全量测试 + `go vet` + gofmt 检查；每 commit What/Why/How 三段式；`go-code-review-check` 通过
- [x] 4.2 推送 + PR（§6.3 选项 B），CI 全绿（新 R04 regen-and-diff 门禁在本 PR 自举生效）自助 merge
- [x] 4.3 合入后：`/opsx:sync`（delta → 主 spec 新能力 yang-codegen-pipeline）+ `/opsx:archive` + 路线图勾选 P5-3 + `/task sync`
