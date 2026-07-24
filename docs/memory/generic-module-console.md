---
name: generic-module-console
description: 通用模块控制台 change（Tab 由模块根派生/support-filter 搜索/operation-exclude 门禁/presence 开关）的交付状态与关键决定
metadata: 
  node_type: memory
  type: project
  originSessionId: 6b81ac50-6436-4309-9df5-740608ebd033
---

OpenSpec change `2026-07-08-generic-module-console`（分支 `generic-module-console`，创建于 dfc-sync-archive worktree，2026-07-08）。交付 UI 重构：左导航（/yang/modules 驱动）+ 右侧「面包屑+一级 Tab」通用控制台 `/module/:module`，零 per-module 前端代码。

**关键决定/事实（改这块前必读）：**
- 华为扩展存活于内嵌 gzip schema：`ext:support-filter`/`ext:operation-exclude` 在 `Entry.Exts`（*yang.Statement，Kind=前缀名/NName=参数），presence 与容器级 when/must 在 `Entry.Extra`。按扩展关键字**本名**（去前缀）匹配。
- FieldDef 新增 supportFilter/operationExclude/presence/isKey；改 FieldDef 后必须 `make gen-contract` 同步 api.gen.ts（漂移门禁）。
- 模块路由名 = 模块**根容器名**（ifm/vlan/system），非 huawei-ifm；运行时配置路径=逐段加根前缀（`configPathFor`，对齐 main.go Prefix 谓词 `/ifm:ifm/ifm:interfaces`）。
- 列派生分层：key→identity(operationExclude∋update)→when 条件叶→enum→其余，封顶 9（utils/moduleConsole.ts 纯函数）。
- presence 语义：开关关=键不存在不入 payload；presence 节点不存在时其 must 不适用（useConfigForm.mustViolations 过滤）。
- 「删除」按钮按模型门禁渲染但**禁用**：后端 /config 无 DELETE、POST 是合并语义，行删除无契约 → follow-up 债。
- ifm 回读解析已扩 class/parent-name/number/link-protocol/router-type（此前恒空）；netconfsim `DemoSeedConfig` 种子 5 接口（3 main/200GE/up + 2 sub/Vlanif/down），standalone 模拟器默认加载。
- DeviceConfigPage 退出路由（/config/interface、/config/vlan 已 redirect），组件暂存待后续 change 删除；SchemaTree 面板随之退出路由界面。
- 分支基于 52a5a36（= squash #119），PR 前须 rebase --onto origin/main 52a5a36（树已验证一致）。
- 覆盖率棘轮：前端 73/70/65/73，后端 57。

相关：[[yang-constraint-engine]]、[[vlan-config-stackb]]、[[reconcile-convergence-3rootcauses]]、[[test-governance-military-rules]]

**交付进度（2026-07-08）**：PR #121（后端扩展透出→main）+ 堆叠 PR #122（控制台→p1 分支）；e2e-local 10/10 绿；rebase 已完成。#121 合入后 GitHub 会自动把 #122 retarget 到 main（p1 分支随 merge 删除时）。

**Follow-up 债（risklog）**：
1. 行「删除」按钮按模型门禁渲染但禁用——后端 /config 无 DELETE、POST 合并语义无行删除契约；需另开 change 加 DELETE 端点后启用。
2. 表单 Tab（ifm global/damp/auto-recovery-times 等路径）GET/POST 后端无 codec/converter → 前端如实透出错误（§9 降级）；配置能力待后端 per-path codec 扩展。
3. ~~FieldDef 不透出 config-false/readonly~~ **已还清**（[[ext-ui-annotations]] PR #126，2026-07-08）：BR-09 readonly 透出 + FE-14 只读 Tab/禁用叶降级。
4. DeviceConfigPage.vue + SchemaTree 面板已退出路由（重定向），组件与其 F2 测试待后续 change 物理删除。
5. 回读 EmitJSON 已 SkipValidation（R08）：设备值不合本地 pattern 不再拖垮回读——但也意味着展示层不再校验设备数据合法性（有意为之）。
