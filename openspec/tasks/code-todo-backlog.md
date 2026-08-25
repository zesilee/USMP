---
id: code-todo-backlog
title: 代码内 TODO 台账（注释清理重构收割）
status: pending
priority: low
branch: (未开工——各项开工时另立 worktree/change)
worktree: (无活跃)
change: (按项另立)
updated: 2026-08-25
source: comment-cleanup 全仓注释清理重构（8 批子代理，256 个非测试源文件）
---

## 目标

把散落在代码里的未完成标记收口到本台账，避免「TODO 写在代码里、没人看、跟着文件一起腐烂」。
代码里保留标记词本身（便于就地看到），但**任务描述与优先级以本文件为准**。

每项开工时各自走 §5 工作流（propose→apply），禁止在本任务下直接堆代码。

## A. 未完成 TODO（代码内仍保留标记）

### A1. NETCONF 通知订阅未实现

- 位置：`backend/pkg/yang-runtime/client/netconf.go:311` `NETCONFClient.Subscribe`
- 现状：直接返回 `subscription not implemented for NETCONF`。
- 背景：NETCONF 没有 gNMI 那样的内建订阅通道，要做设备状态变更推送得按
  RFC5277 实现 `<create-subscription>` + notification 流。
- 影响面：无调用方依赖，属能力缺口而非缺陷；gNMI 本身也是规划能力（§1 R02）。
- 层次：B1 + B2（模拟网元需补 notification 流才能端到端验）。

### A2. Schema.Validate 只查路径存在性

- 位置：`backend/pkg/yang-runtime/schema/schema.go:87` `DefaultSchema.Validate`
- 现状：路径不存在则报错，存在就直接 `return nil`，不做任何约束校验。
- 背景：真正的 YANG 约束校验（when/must/pattern/range/leaf-list）落在
  `pkg/yang-runtime/validate` 包（见 [[yang-constraint-engine]]），`Schema` 接口
  这条通道一直是空壳。
- 待定：是把 `validate` 接进来，还是把这个方法从接口上摘掉——先定语义再动手。
- 层次：B1（表格驱动 + 负路径）。

### A3. 前端未打标记的遗留债（注释里写着「待收」，但没有 TODO 标记）

清理时发现三处以自然语言陈述的欠账，代码内陈述原样保留，在此登记以免继续沉没：

- `frontend/src/ui/eview/components/inputs.tsx:76` —— Spinner 桥缺 placeholder、
  步进器不可隐藏，注释写「记录，样式层再收」。
- `frontend/src/ui/eview/components/controls.tsx:41` —— Dropdown 走 textOf 文本化，
  label 内 `data-test` 锚点丢失，E2E 只能按文本选。属已知锚点债。
- `frontend/src/components/layout/Header.tsx:8` —— 注释称「新鲜度环随 tasks 11.3 在此
  挂载」，但 Header 里没有新鲜度环。**判断不了是已挪走还是仍未做，故原样未动**，
  需要知情人销项或补做。

另有两处「计划锚点疑似可销项」，改注释时已对齐代码，但台账那头要不要销需人工定：
`frontend/src/ui/eview/components/structure.tsx:313`（`expandable` 行为原称「tasks 5.1
已登记、行为暂缺」，实际 `flatten` 已实现 `defaultExpandAllRows`）。

## B. 注释清理途中发现的存量代码问题（本次刻意未改）

> 本次重构铁律是「只动注释」，以下问题只记录、不修。每条都需要单独判断是缺陷还是有意为之。

### B1. Manager.Stop 静默吞掉 CloseAll 错误

- 位置：`backend/pkg/yang-runtime/manager/manager.go:216`
- 形态：`if err := m.clientPool.CloseAll(); err != nil { }` —— 空块，原注释写的是
  「Log but continue shutdown」，但块内没有任何 log 调用。
- 判断：R08 要求异常必须降级处理，「继续关停」本身没问题，但错误完全不落地
  会让关停期的连接池故障不可观测。建议补 log。

### B2. file source 的 watcher 错误同样只 `_ = err`

- 位置：`backend/pkg/yang-runtime/source/file.go:86`
- 形态：原注释「Log error but continue」，代码只有 `_ = err`。
- 判断：同 B1，属可观测性缺口。

### B3. desired 为 nil 时不触发删除

- 位置：`backend/pkg/yang-runtime/reconcile/reconcile.go:91`
- 形态：`desired == nil` 直接返回成功，不做任何对齐动作。
- 判断：**需要确认设计意图**。若「desired 为 nil 应触发删除」才是本意，这就是逻辑
  缺口而非注释缺口。注意声明式通道刻意删不了、删除走独立 DELETE 命令通道
  （见 [[config-delete-semantics]]），所以现状很可能是有意为之——但原注释自相
  矛盾（一句说「应被删除」，下一句说「已经没了就不用动」），值得确认后把结论
  写死进注释。

### B4. 库代码里的裸 fmt.Printf 调试残留

- 位置：`backend/pkg/yang-runtime/client/netconf.go:298`（`Set` 失败路径）
- 形态：`fmt.Printf("Change failed: %v\n", ch.Error)` 直打 stdout。
- 判断：库代码不该直接写 stdout，建议改走 logger 或删除。

### B5. 生成器输出的注释里变量名笔误

- 位置：`backend/tools/yanggen/emit.go:330`（`emitEnumMaps` 的模板字符串内）
- 形态：模板输出 `// enumMaps 是全包枚举定义表…`，而它紧接着生成的变量名是
  `EnumMaps`（首字母大写）。
- 判断：改它会让 `internal/generated/*` 全量重生成、触发 regen-and-diff 门禁，
  必须作为生成器改动单独走 `make gen-yang` 流程（R04）。

## C. 注释里记录的措辞漂移（已就地改注释对齐现状，备案供复核）

以下是「注释停留在退役技术的现在时表述」，本次已改注释文字对齐代码实际行为，
台账锚点一律保留。若哪条改写口径不对，回退即可：

- `client/backend_core.go`、`client/netconf.go`、`client/rpc.go`：scrapligo 双路径
  时态（scrapligo 已于 2026-08-04 按 NC-01 物理删除，见 [[netconf-selfdev]]）
- `manager/manager.go`：Plugin management（脚手架已随 retire-idle-scaffolds 删除）、
  不存在的 `SchemeDir` 字段、ygot 运行期优先级、审计「flush to disk」
- `internal/controller/{vlan,ifm}/reconciler.go`：JSON/XML 判定次序与代码相反
- `backend/main.go`、`cmd/test-server/fixture.go`：gin 残留提法（已迁 beego，
  见 [[gin-to-beego-migration]]）
- `internal/staticweb/handler.go`：Vue history 路由（现栈 React）
- `internal/yangschema/ir_load.go`：checked-in IR blob（R18 已改为不入库、构建期生成）
- `frontend/src/stores/menu.ts`：zustand selector（已换自研 createStore）
- `frontend/src/utils/xmlFormat.ts`：v-for / v-html Vue 术语（现栈 React）

未改、留作历史 Why 锚点的（措辞提到退役技术，但那正是结论的来源）：
`frontend/src/utils/deviceForm.ts` 的 el-form 决策理由、`frontend/src/api/index.ts`
与 `utils/nodeSupport.ts` 的 axios 口径（inula-request 翻转波收尾时统一清）。
