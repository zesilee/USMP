# design — ext-ui-annotations（收割存量呈现元数据）

## Context

呈现扩展管线已由 generic-module-console 验证成熟：内嵌 gzip schema（`huawei.Schema()`，编译进二进制、运行期零 `.yang` 依赖）→ goyang `Entry.Exts/Extra/原生字段` → `schema/entry.go` 前缀无关提取 → LeafNode/Node → `internal/api/field_gen.go` → FieldDef JSON → `make gen-contract` 漂移门禁 → 前端 moduleConsole/FieldRenderer 消费。

2026-07-08 探针实证（对内嵌 schema 实测，非推断）：

| 元数据 | 存活性 | ifm 实测 |
|--------|--------|----------|
| `config false`（原生） | ✅ 存活，`Entry.Config`=false 于子树根，`e.ReadOnly()` 父链推导可用（unzip 后 Parent 指针完好） | 121 个只读节点（ipv4-interface-count、remote-interfaces…） |
| `ext:dynamic-default`（数据节点级 Exts） | ✅ 存活 | 10 处（admin-status/mtu/link-protocol…） |
| `units`（原生 `Type.Units`/`Entry.Units`） | ✅ 存活 | 1 处（dynamic/bandwidth "bit/s"） |
| `ext:task-name`（**模块级**） | ❌ **不存活**——全 SchemaTree 扫描=0，模块级语句在 ygot 生成时被丢弃 | — |

## Goals / Non-Goals

**Goals:**
- S1 `config false` → `FieldDef.Readonly`（契约字段已存在、前端 `moduleConsole.ts` 已过滤 `!f.readonly`，仅后端未填）；前端 state 子树降级只读呈现。
- S2 `ext:dynamic-default` → `FieldDef.DynamicDefault`；前端「系统自动分配」占位语义。
- S3 `units` → `FieldDef.Units`；输入框单位后缀。
- S4 模块级 `ext:task-name` → 构建期 codegen 映射表 → `/yang/modules` 增 `category`；左导航分组。
- 决策入档：自造 `ext:ui-*` 词汇推迟。

**Non-Goals:**
- 不自造任何 YANG extension（`ext:ui-widget/ui-order/ui-view` 等）——无消费场景（segmented 可由 enum 基数派生、Tab 已由 container 派生、顺序=schema 序），且需 patch 厂商模型源文件构成 fork 维护债。将来出现真实场景时以 **augment/deviation 模块**承载，不改厂商源文件。
- 不改 Reconciler / NETCONF 写链路 / 下发 payload 编码（readonly 字段本就不该进 payload，前端裁剪即可）。
- 不做 `ext:can-be-deleted` / `generated-by`（三服务模块零使用，语义归 P4 删除语义 change）。
- 不动 DeviceConfigPage（已退役）。

## Decisions

### D1 readonly 计算：构树时下推，不依赖运行期父链
`entry.go` 递归构树时把「继承只读」作为参数下推（`config false` 一经出现整棵子树只读），而非每次调 `e.ReadOnly()` 走父链。理由：语义等价（YANG config 继承性），但与现有 entryToNode 递归结构同构、零父链遍历开销；探针已证 Parent 指针可用，故两者皆可，选更贴合现有代码的。**容器级只读也要落到 Node**（Tab 派生需要），不只叶级。

### D2 dynamic-default 只透出布尔，不透出子句
华为 `ext:dynamic-default` 可带 `ext:default-value` 子句（XPath 表达式+条件）。本期只透出 `DynamicDefault bool`——UI 语义仅需「该字段由系统动态缺省」即可（占位符+非必填）；表达式求值属过度设计（ifm 10 处全为无子句形态，探针实证）。

### D3 S4 走构建期 codegen（用户拍板，2026-07-08）
模块级扩展不存活于运行期 schema → 仿 ygot 的 go:generate：小生成器用 goyang 解析 `yang-models/network-router/8.20.10/ne40e-x8x16` 下与 ygot 生成集**同一模块清单**（huawei-vlan/ifm/system），提取模块级 `ext:task-name`，生成 `taskname.gen.go`（`map[模块根容器名]category`，与前端路由键对齐——路由名=根容器名 ifm/vlan/system，非 huawei-ifm）。生成物**提交入库**（同 all.gen.go），运行期零 submodule 依赖。备选「正则抠 .yang」被否：goyang 已是依赖，AST 解析不脆弱。

### D4 前端只读降级分两层
- **Tab/子树级**：`moduleConsole.ts` 派生 Tab 时，整棵 readonly 子树标记只读 Tab → 渲染只读视图（描述列表/禁用表单），非直接隐藏（呈现噪音债#3 的修复是「降级」不是「消失」，state 数据仍有查看价值）。
- **叶级**：混合容器内的 readonly 叶 → FieldRenderer 禁用态 + 不入 diff/payload/校验。

### D5 契约同步
FieldDef 增 `DynamicDefault`/`Units`，YangModuleInfo 增 `Category`（均 `omitempty`）→ `make gen-contract` 再生成 api.gen.ts，漂移门禁 CI 卡未同步。

## Risks / Trade-offs

- [ifm units 仅 1 处且位于只读子树，S3 本期几乎无可见收益] → 成本极低（一个字段透传），价值在契约完备性与后续模块；接受。
- [readonly 子树可能包含 list（如 remote-interfaces），只读列表呈现路径与可编辑列表不同] → 只读 Tab 统一走简化只读视图，F2 覆盖 list/leaf 两种形态。
- [S4 生成器绑定 8.20.10/ne40e-x8x16 单版本] → 与 ygot 生成集同源同版本，升级模型时一并重跑 go:generate；生成器入参路径与 huawei.go 的 go:generate 保持相邻声明，降低漂移。
- [category 键与模块路由键不一致风险（huawei-ifm vs ifm）] → D3 明确用根容器名做键，B1 测试断言与 `/yang/modules` 返回的 name 一致。
- [覆盖率棘轮：后端 57 / 前端 73/70/65/73] → 新增代码全部带层内测试（B1/B3/F1/F2），补测后按需上调基线。

## Migration Plan

纯增量透出 + 前端呈现，无数据迁移。`omitempty` 保证旧前端对新字段零感知；先合后端透出、前端消费同 PR 内跟进（≤1000 行可容纳）。回滚 = revert PR。

## Open Questions

（无 — S4 机制已拍板，其余无悬而未决项。）
