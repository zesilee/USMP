# .claude 自优化审计报告（2026-08-25）

> 由 `/self-audit` 产出。扫描范围：`.claude/` 全部配置资产 + 根 `CLAUDE.md` + `openspec/SPEC_CONVENTIONS.md`（specs 正文经分类后全部属业务能力描述，划出本次审计对象，过期发现见附录）。
> 证据来源：git 提交历史、全仓库交叉引用 grep、hooks 运行日志、路径存在性逐条 ls 验证、与当前技术栈（Go1.22+Beego / React19+EviewUI适配层 / 自研netconfcore / 自研yanggen / yang-controller-runtime）比对。
> **本报告只是方案，未执行任何变更。审批方式见文末。**

## 总体画像

| 指标 | 值 |
|---|---|
| 资产总数 | 45 个入库文件（命令 11、技能 21+附属、agents 2、hooks 7、设置 1、附属文档 2）+ CLAUDE.md |
| 健康活跃 | opsx 命令簇、task.md、pre-tool-use.sh、yang-config-test-design、golang-patterns/testing 等 |
| 整体过期（内容与现实相反） | 3 个技能（yang-ygot-generate 100%、netconf-switch-protocol ~95%、frontend-yang-dynamic-form ~90%） |
| 纯重复副本 | 5 个 openspec-* 技能 ≡ 5 个 /opsx:* 命令（同 CLI 生成，diff 仅措辞） |
| 外来模板孤岛（2026-04-29 引入，零主文档引用，示例为别的项目） | plan.md、tdd-backend.md、tdd-frontend.md、planner.md、tdd-guide.md |
| 零引用孤儿 | hooks/pre-commit.sh、post-test.sh 及两份配套 review.md（2026-04-26 起无任何调用方） |
| 从未生效的配置 | post-task-sync 钩子（31 条日志全为空值跳过）；settings.json 的 plugins/superpowers 两键（非法字段+插件未安装） |
| 断链引用 | CLAUDE.md §5/§6.3/§7.3 引用的 superpowers 9 个技能——插件未安装，全部无法调用 |

---

## 一、命令与子代理

| # | 文件 | 功能简介 | 现状评估 + 证据 | 建议 | 风险 |
|---|---|---|---|---|---|
| 1 | `.claude/commands/plan.md` | 调 planner 代理出实施计划 | **僵尸/冲突**。2026-07-03 后未动；示例为预测市场项目（Supabase/Redis/BullMQ，Redis 示例与 R03 禁数据库顶牛）；引用不存在的 `/tdd`、`/build-fix` 命令与 `~/.claude/agents` 错误路径；与 §5 opsx 流程、原生 plan mode 职能冲突；除自簇互引外零引用 | **删除** | 低 |
| 2 | `.claude/commands/tdd-backend.md` | Go TDD 分步教程 | **僵尸**。2026-04-29 引入后零修改、全仓库零引用；自称 `/go-test` 但文件名对不上；引用不存在的 `/go-build`、`/go-review`；与 T01-T09、§5.6、golang-testing 技能重复 | **删除** | 低 |
| 3 | `.claude/commands/tdd-frontend.md` | 通用 TDD 教程（TS 示例） | **僵尸**。同上；示例"市场流动性评分"为外来项目；引用不存在的 `/e2e`、`/test-coverage`、`/build-fix` | **删除** | 低 |
| 4 | `.claude/agents/planner.md` | 规划子代理（自动激活） | **冲突/冗余**。frontmatter 声明 PROACTIVELY 自动激活，与 §5 explore→propose 权威流程抢活；主体为外来模板（Stripe/Supabase 订阅计费完整示例，含建库表——与 R03 精神相悖）；零主文档引用；原生 Plan 代理与 plan mode 已覆盖 | **删除** | 中：若你有手动召唤它的习惯请说明，改为精简保留 |
| 5 | `.claude/agents/tdd-guide.md` | TDD 子代理 | **冗余**。npm 中心（对 Go 后端失配）；要求 mock "Supabase, Redis, OpenAI"；职责已被 T01-T09 军规 + §5.6 分层 + pre-commit/CI 拦截全面取代；零主文档引用 | **删除** | 中（同上） |
| 6 | `.claude/commands/verify.md` | 全量验证（构建+测试+审计） | **低频/过期**。把前端写成 "Vue3 + TypeScript"（实际 React 19）；未对齐 make 目标与覆盖率棘轮 | **精简修正**：栈名改对、命令对齐 `make` 目标与 §6.2 门禁 | 低 |
| 7 | `.claude/commands/task.md` | 任务断点续做 | **活跃**。CLAUDE.md §12 SR01-SR05 直接引用 | **原样保留** | — |
| 8 | `.claude/commands/opsx/`（5 个） | §5 工作流权威载体 | **活跃**。CLAUDE.md §5 逐阶段引用 | **原样保留** | — |

## 二、Hooks

| # | 文件 | 功能简介 | 现状评估 + 证据 | 建议 | 风险 |
|---|---|---|---|---|---|
| 9 | `hooks/pre-commit.sh` + `hooks/pre-commit-review.md` | 提交前提示 AI 做代码检视 | **孤儿**。2026-04-26 起全仓库零引用；脚本只 cat 一段提示词、无退出码非 0 路径、拦不住任何东西；职能已被 `.githooks/pre-commit`（162 行真拦截：R15/R16/R18/R04）取代 | **删除** | 低 |
| 10 | `hooks/post-test.sh` + `hooks/post-test-review.md` | go test 后触发检视 | **孤儿**。同上，零引用 | **删除** | 低 |
| 11 | `hooks/post-task-sync.sh` + settings.json 的 PostToolUse 块 | Task 变更自动同步到 openspec/tasks/ | **僵尸（从未生效）**。日志 31 行全部为 `Skipping: `（`$TOOL_NAME` 恒为空）——settings 用参数传 `$TOOL_INPUT`，脚本却读环境变量，且真实钩子协议是 stdin JSON。功能从未跑通过一次，因 `set +e` 静默至今 | **修复（推荐）**：改为从 stdin 解析 tool_name/tool_input。备选：整体删除（`/task sync` 手动同步仍在，§12 不受影响） | 中 |
| 12 | `hooks/pre-tool-use.sh` | Bash 前置拦截：R13 直推 main、W07 force push、R04 手改 generated、W01 checkout main、R16 写 .env、rm -rf / | **活跃·高价值护栏** | **原样保留**（红线保护条目） | — |
| 13 | `.claude/HOOKS.md` | hooks 说明文档 | **漂移**。描述的 Stop 钩子与 go test 后钩子在 settings.json 中不存在；实际配置的两个钩子它反而没写 | **精简重写**：对齐真实钩子清单（随 #9-#11 结果同步） | 低 |

## 三、设置

| # | 条目 | 现状评估 + 证据 | 建议 | 风险 |
|---|---|---|---|---|
| 14 | `settings.json` 的 `"plugins": ["superpowers"]` 与 `"superpowers": {autoPlan/autoTest/autoReview}` | **失效**。两个键均非 Claude Code 合法 settings 字段；superpowers 插件在本机插件清单中不存在（仅市场缓存可见）。三个开关无任何生效载体 | **删除这两个键**（与 #30 联动决策） | 低（本来就不生效） |
| — | `settings.json` 其余（permissions/attribution/hooks 引用） | 正常 | 保留（hooks 块随 #11 调整） | — |
| — | `settings.local.json` | 本机文件、未入库、内容正常 | 不动 | — |

## 四、技能（21 个）

### 4a. 建议删除

| # | 技能 | 现状评估 + 证据 | 建议 | 风险 |
|---|---|---|---|---|
| 15 | `yang-ygot-generate` | **100% 过期，有害**。全篇教"用 openconfig/ygot 生成+运行时 EmitXML/Unmarshal"——运行时零 ygot 是已交付红线，`backend/ygot_retirement_guard_test.go` 专门拦这个；照它做 CI 必红。输出路径 `yang_models/` 不存在 | **删除**，CLAUDE.md §7.1 对应行改指 regen-and-diff 流程（见 #31） | 中（需同步改 §7.1 一行） |
| 16 | `openspec-apply-change` / `-archive-change` / `-explore` / `-propose` / `-sync-specs`（5 个） | **纯重复**。与 `/opsx:*` 命令同一 CLI 生成（generatedBy 1.4.1），diff 仅措辞；技能正文自己都在引用 `/opsx:*`；双份并存造成双重触发面 | **删除整簇**（保留 opsx 命令为唯一入口；openspec CLI 随时可再生成） | 低 |
| 17 | `tdd-workflow` | **重复簇核心 + ~35% 过期**。766 行全库最长；§4 前端整节是 Vue3+Element-Plus 世界观（`@vue/test-utils`、`el-dialog`）；后端引用不存在的 `test/netconf-simulator` 与 ygot 类型；其有效内容（表格驱动/race/覆盖率/集成模板）已被 CLAUDE.md §5.6、frontend/TESTING.md、golang-testing 覆盖。CLAUDE.md 未引用它 | **删除**（有效增量并入 #18） | 中 |

### 4b. 建议重写/精简修正

| # | 技能 | 现状评估 + 证据 | 建议 | 风险 |
|---|---|---|---|---|
| 18 | `tdd-test-driven-dev` | **过期+劣质示例**。被 CLAUDE.md §7.1 引用（T01 载体）但内容差：import 占位符"你的项目路径"从未替换；并发示例在子 goroutine 调 t.Errorf + Magic Sleep（恰是反模式）；与缓存技能构造函数签名互相矛盾 | **重写为薄权威**（~60 行）：指向 §5.6 分层表 + T01-T09 + frontend/TESTING.md，正确示例对齐 backend 真实包 | 中 |
| 19 | `netconf-switch-protocol` | **~95% 过期**。全篇 scrapligo API（NC-01 明令禁回引、守护测试拦截）+ ygot 对接 + OpenConfig 示例（主力是华为 YANG）；示例还有 goroutine 泄漏。被 §7.1 引用（R02 载体） | **重写为薄指针**（~40 行）：SSH830/RFC6241 约定 + netconfcore 真实入口（backend/pkg/yang-runtime/client）+ 指向 docs/netconf-core-field-validation.md | 中 |
| 20 | `frontend-yang-dynamic-form` | **~90% 过期**。description 即"基于Vue3+Element-Plus"；7 个 `.vue` 组件全不存在（仓库 .vue 文件数为 0）；`web/src/*` 路径全 MISS。被 §7.2 引用（R05 功能型触发载体） | **重写**：对齐 React19 + `src/ui` 适配层军规（FA-01~04）+ `src/form` 真实结构 + YANG 类型→控件映射现实版 | 中高（R05 的执行载体，重写需准确） |
| 21 | `go-code-review-check` | **~45% 过期**。description 与正文含 3 整节退役内容：ygot 类型安全、Proto.Actor 模型、Gin API 规范（backend/go.mod 已验证：无 gin、beego v2.3.0）。但它是 §6.2 提交门禁（T04）载体，架构/缓存/Go 通用节仍有效 | **精简修正**：删 3 个退役节，Gin 节换成 beego 规范（含 no_gin_guard、wildcardPath 等真坑），路径补 `backend/` 前缀 | 中（门禁技能，改动需核对） |
| 22 | `netconf-sim-integration-test` | **~30% 过期**。核心路径 `test/netconf-simulator` 反复出现 4 次（实际 `backend/simulator/netconfsim`）；import 全部缺 `/backend` 前缀；引用已删除的 `internal/generated/openconfig` 类型；断言 API 名与 yang-config-test-design 互相矛盾。被 §7.1 引用（T02 载体） | **精简修正**：路径/import/断言名对齐真实代码（testsupport/asserts.go） | 中 |
| 23 | `yang-controller-runtime-dev` | **~20% 过期**。架构描述与现状最贴合（保留价值高）；但 import 缺 `/backend`、`test/netconf-simulator` 路径错、末尾"与原有 Actor 架构对比"整表已无信息价值且误导 Actor 仍是活代码 | **精简修正**：修路径、删 Actor 对比表 | 低 |
| 24 | `go-ttl-lru-memory-cache` | **示例带 bug**。方向正确（R03 载体），但示例代码在 RLock 下写 map（数据竞态——恰违反 R09）、接收者类型名拼写不一致、构造函数签名与 #18 矛盾 | **精简修正**：删错误示例，指向 backend/internal/cache 真实实现 + 关键约定（Key=IP+路径、TTL30s、下发后失效） | 低 |
| 25 | `e2e-testing` | **~20% 过期**。`web/tests/` 目录不存在（实际 `frontend/tests/`）；与 frontend/TESTING.md F4 层职责重叠 | **精简修正**：路径改对，定位为 F4 补充、以 TESTING.md 为权威 | 低 |
| 26 | `git-what-why-how-commit` | **规则有效、示例过期**。示例引用不存在的 `cmd/test-server/main.go`、`web/tests/`。另与 `.claude/COMMIT_CONVENTION.md` 内容高度重复（两份 What/Why/How 规范） | **精简修正示例 + 合并**：技能为唯一权威，吸收 COMMIT_CONVENTION.md 的 type 表与优秀示例（见 #32） | 低 |
| 27 | `yang-config-test-design` | **最新鲜**。9 条索引路径 7 条验证存在；唯一含真机 bug 复盘。仅 2 个前端路径过期（useDeviceConfig.ts / DeviceConfigPage.vue）+ 一处 `el-form` 措辞 | **微修**这 3 处 | 低 |
| 28 | `frontend-patterns` | **方向正确、库不匹配**。React+TS 通用模式（origin: ECC 外来），但通篇 Next.js/SWR/React Query/Zustand/Framer Motion 均不在 frontend/package.json；项目实际 openinula 运行时 + 自研 store | **降级标注**：头部加"⚠️ 通用参考，非本项目规范；组件库经 src/ui 适配层、store 用自研 createStore，以 frontend/TESTING.md 与适配层军规为准" | 低 |

### 4c. 原样保留

| # | 技能 | 理由 |
|---|---|---|
| 29 | `golang-patterns`、`golang-testing`、`web-design-engineer` | 前两者零过期、零框架耦合、有独有内容（fuzz/golden/benchmark）；web-design-engineer 被 §7.2 引用且附属 references 路径有效 |

## 五、CLAUDE.md 与附属文档

| # | 条目 | 现状评估 + 证据 | 建议 | 风险 |
|---|---|---|---|---|
| 30 | CLAUDE.md 中 superpowers 引用（§5 hotfix 说明旁、§6.1/§6.3 完成分支、§7.3 整表 9 个技能、settings.json #14） | **断链**。插件未安装（本机 installed_plugins.json 无条目），9 个 `superpowers:*` 技能全部无法调用；worktree 能力实际由原生 EnterWorktree/ExitWorktree 工具承担，§6.3 流程步骤本就完整写在表格里 | **二选一，需你拍板**：<br>**方案 B（推荐）**：摘除 superpowers 引用——§7.3 表删除或替换为原生等价物（plan mode、EnterWorktree、systematic debugging 内化到流程文字），§6.3 去掉技能名、保留已写全的步骤表。理由：核心能力已被原生工具+opsx 覆盖，少一个外部依赖。<br>**方案 A**：重新安装 superpowers 插件恢复原状（一条命令），CLAUDE.md 不动 | **高**（动 §6/§7 工作流正文；红线本体 R/W/T 条款不动） |
| 31 | CLAUDE.md §7.1/§7.2 技能映射表 | 随技能处理需同步 3 行：`yang-ygot-generate` 行改为"regen-and-diff 生成流程（tools/yanggen）"；TDD 行确认指向重写后的 #18；其余行技能名不变（原地重写） | **同步修正**（跟随 #15/#18 执行） | 中 |
| 32 | `.claude/COMMIT_CONVENTION.md` | **重复**。与 git-what-why-how-commit 技能两份同一规范（2026-04-26 后未动）；好内容（type 表、正反示例）值得留 | **合并入 #26 技能后删除本文件**；若 .githooks/commit-msg 注释引用它则同步改指技能 | 低 |
| — | CLAUDE.md 其余全部（§1-§6、§8-§13 红线/军规/流程） | 高价值、与现状一致（§3 技术栈行已是 beego/EviewUI 口径） | **原样保留** | — |
| — | `openspec/SPEC_CONVENTIONS.md` | 活跃（R17 载体，2026-07-08 更新） | **原样保留** | — |

---

## 附录：范围外发现（本次不执行，建议另立 docs 任务走 R17 流程）

openspec/specs 正文分类后全部属业务能力描述（41/41），不在本次审计动刀范围，但扫描顺带发现真实过期内容：

1. `specs/frontend/design.md` — 整份仍是 Vue3+Element Plus+Pinia+useK8sCRD 技术栈，与同目录 spec.md（openinula+EviewUI）直接矛盾（最大过期块）。
2. `specs/system-architecture/design.md` — §2/§6/§7 仍以"双栈并存待决策"现在时描述，与其 spec.md/tasks.md"已切栈已删除"内部矛盾；多处把 Gin 写成现役（backend 已验证为 beego）。
3. `specs/device-protocol/design.md:19` — scrapligo 以现在时写为当前传输实现（NC-01 禁回引口径）。
4. `specs/frontend-ui-adapter/spec.md:11` — FA-01 正文"当前实现为 antd"与同文件头部"已切 EviewUI"自相矛盾。
5. `business-crd`/`translation-engine` 的 design.md 缺 LEGACY 头（spec.md 已有）。
6. 3 个 spec 的 Purpose 为 `TBD - created by archiving change` 占位（console-derivation-golden、schema-fixture-pipeline、yang-rpc）。
7. 建议将 `openspec/.openspec.yaml`（workflow/constraints 段）纳入下轮审计范围。

---

## 审批方式

- 回复「执行 1,2,3」式编号清单，或「除 30 外全部执行」。
- 含**删除**动作的条目（1-5、9、10、15、16、17、32）与**高风险**条目（30）需明确点名，不接受笼统的「都行」。
- #30 需在方案 A/B 之间拍板。#11 需在修复/删除之间拍板（默认推荐修复）。
- 执行后将按 Step 4 自检（JSON 语法、hooks 引用完整性、残留引用 grep、frontmatter 合法性），并输出优化结果汇总 + 冲突自检。
