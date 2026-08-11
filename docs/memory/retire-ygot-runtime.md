---
name: retire-ygot-runtime
description: 发布二进制去 ygot/goyang 已全量交付归档（2026-08-11，PR#307-#312）：碰 yanggen/schemagen/object/validate/xmlcodec、改生成器约定、或再做大分支拆 PR 前必读
metadata:
  type: project
---

# retire-ygot-runtime（**已全量交付归档**，2026-08-11 单日完成，PR#307-#312）

**最终态实证**：`go list -deps`/`go version -m` 双口径 0 openconfig；主 go.mod/go.sum 0 openconfig（7.2 拍板=**tools 拆独立 go module** `backend/tools/go.mod`）；守护测试 `backend/ygot_retirement_guard_test.go`（闭包+go.mod 双守护）拦回引；旧 25 万行 ygot 包已物理删除；gen-yang=yanggen（businessdemo 留 ygot demo 路径）；schemagen 直读 YANG 源（desc 剥离冻结、模块 namespace 补全采纳）。change 已归档 `openspec/changes/archive/2026-08-11-retire-ygot-runtime/`（含 codegen-conventions.md=生成约定权威、switch-plan.md）。

**目标与范式**：参考 K8s runtime.Object/Scheme（极小标记接口 + 构建期生成样板，**不**引入 apimachinery）。Why：商用自主可控 + 解除 ygot v0.29.20 天花板债（[[go-122-pin]]）。

**交付脉络**（PR#307 提案 / #308 阶段1 Schema IR / #309 阶段2 object+yanggen+native 并存 / #310 阶段3 JSON+S1S2 xmlcodec IR 化 / #311 S3S4+校验器+删旧包+tools 拆 module / #312 守护门禁+spec sync+归档）。

**改生成器前必读（对拍实证冻结）**：①生成域=整个依赖闭包（被 import 模块顶层容器也入 Device）；②typedef 枚举按**使用方叶所属模块**命名；③内联枚举按 AST 节点去重；④union 同型折叠（265→6 接口）；⑤JSON 键**永不带模块前缀**（ygot 现网未开 AppendModuleName，解码双形态兼容）；⑥validate 冻结 ygot 快照语义（mandatory 不校验、min-elements 仅对存在的空 list）。踩坑：goyang EnumType Names()/Values() 不可 zip（NameMap 权威）；leafref 按数据树语义解析（choice/case 不占层级）。

**tools 独立 module 四坑**：①tidy 会拉最新依赖并写 `toolchain go1.24.6`——require pin 从 backend/go.mod 整体复制 + GOTOOLCHAIN=local + **删两边 toolchain 指令**；②跨模块 go:generate 用 `go -C ../../tools run`；③Makefile/脚本 `cd backend/tools` 后所有相对路径要重推（gen-crd2yang 的 `-output` 少 `../` 曾红 CI）；④pre-commit 对 build-tag-only 目录要 `go list | grep -q .` 守护。

**大分支拆 PR 方法论（本次 38 commit/5760 计数行 → 4 PR 全绿合入）**：
- 中途「提交完整性修复」会造出**不可编译的前缀树**——切点必须实测 `git show <sha>:file` 验证；坏切点用**乱序重放**：纯 docs 提交与功能提交路径零交叠时，按原顺序 cherry-pick 功能段成 C1、docs 段后置成 C2，零冲突且终态树逐字节一致（`git diff` 空验证）。
- 压线超限找**可无损后置的纯新增文件**（守护测试 49 行从 C1 挪 C2 解决超 27 行）；CI commit-lint 只查格式不查体积（500 行/commit 仅本地钩子管）。
- swag 契约会因**包名歧义消失**漂移（删旧包后 `status.Status` 回落短名）——删大包后必跑 `make gen-contract`。
- 后台 `git push`/后台 docker e2e 会被杀：e2e 用 `nohup … &` + Monitor 盯 PID；push 前台跑（e2e 刚验绿可 USMP_SKIP_E2E=1 让 -race 套件塞进前台超时）。
- 分支保护要求 up to date：`gh api …/pulls/<n>/update-branch -X PUT`；旧 head 的红 run 在 main 前进后 `gh run rerun --failed` 即绿（regen 步骤按运行时 diff 判定）。
