---
id: code-todo-backlog
title: 代码内 TODO 台账（注释清理重构收割）
status: pending
priority: low
branch: (A 段未开工；B 段已于 worktree-fix-observability-gaps 收口)
worktree: (无活跃)
change: (按项另立)
updated: 2026-08-26
source: comment-cleanup 全仓注释清理重构（8 批子代理，256 个非测试源文件）
b_done: 2026-08-26 —— B1/B2/B4 三处静默吞错误已补日志（各带回归用例），
  B3 经 spec BIO-05 定性为契约、加锁定用例防误改，B5 生成器笔误已修并重生成
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

## B. 注释清理途中发现的存量代码问题 —— ✅ 全部收口（2026-08-26）

> 原记录是「只记不修」，因为注释清理那一轮的铁律是不动代码。本段已单独立项修完，
> 每条按 T07 先写复现回归用例（红）再修（绿）。

### B1. Manager.Stop 静默吞掉 CloseAll 错误 —— ✅ 已修

- 位置：`backend/pkg/yang-runtime/manager/manager.go`
- 病灶：`if err := m.clientPool.CloseAll(); err != nil { }` 是空块，注释写着
  「Log but continue shutdown」却一行 log 都没有。连接池关不干净会在设备侧留下
  悬挂会话，而这个失败完全不可观测。
- 处置：补 `log.Printf("manager: client pool close during shutdown failed …")`。
  「继续关停」的降级语义（R08）保持不变，只是错误不再蒸发。
- 回归：`manager_stop_logging_test.go` —— 失败必落日志 + 成功不刷噪声（双向锁定）。

### B2. file source 的 watcher 错误同样只 `_ = err` —— ✅ 已修

- 位置：`backend/pkg/yang-runtime/source/file.go`
- 病灶：同 B1。监听退化（inotify 上限、挂载点消失）后事件源静默失聪，
  现象是「改了文件没反应」且无任何线索。
- 处置：补 `log.Printf("file-source: watcher error on %s (still watching) …")`。
- 回归：`file_watcher_error_test.go` —— 错误落日志 + 报错后监听协程仍存活。
  日志写在独立协程，用例用带锁缓冲接收，`-race` 下干净（R09）。

### B3. desired 为 nil 时不触发删除 —— ✅ 定性为契约，非缺陷

- 位置：`backend/pkg/yang-runtime/reconcile/reconcile.go`
- 结论：**有意为之**。`openspec/specs/business-intent-orchestration/spec.md` BIO-05
  写死「删除请求 SHALL 触发展开为 DELETE 命令通道调用（声明式通道不承载删除）」。
  同见 [[config-delete-semantics]]。
- 为什么容易误判：diff 引擎里确实有 `desiredNil && !actualNil → DeleteChange`
  分支（`diff/diff.go`），单看那段很容易得出「对账应该能删」，进而把这里的提前
  返回当逻辑缺口"修"掉。真那么改，**任何一次 desired 过期或读取落空都会被翻译
  成删真机配置**。
- 处置：把结论连同 BIO-05 锚点写死进注释；补锁定用例
  `reconcile_desired_absent_test.go`（断言不回读设备、不进 diff，另设对照组
  确认 desired 存在时正常走完整链路）。

### B4. 库代码里的裸 fmt.Printf 调试残留 —— ✅ 已修

- 位置：`backend/pkg/yang-runtime/client/netconf.go`（`Set` 失败路径）
- 病灶：`fmt.Printf("Change failed: %v\n", ch.Error)` 直占 stdout——绕过进程日志
  配置（无时间戳、不可重定向分级），且与全仓一律 `log.Printf` 的约定不一致。
- 处置：改为 `log.Printf("netconf: %s change %s failed: …")`，并补上设备 IP 与
  变更路径——聚合错误只说「有变更失败」，哪条路径因何而废原先只能靠猜。
- 回归：`netconf_set_failure_logging_test.go` —— 失败明细带路径落日志 +
  全成功时不刷噪声。

### B5. 生成器输出的注释里变量名笔误 —— ✅ 已修

- 位置：`backend/tools/yanggen/emit.go`（`emitEnumMaps` 的模板字符串内）
- 病灶：模板输出 `// enumMaps 是全包枚举定义表…`，实际生成的变量名是 `EnumMaps`。
- 处置：改模板首字母大写，`make gen-yang` 重生成。漂移精确命中 2 个生成文件各 1 行
  （`native/business/all.gen.go`、`native/huawei/enum_map.go`），无无关漂移。
  `yanggen` 的 golden 用例如期拦截，已 `UPDATE_GENTEST=1` 刷新并核对 diff。

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
