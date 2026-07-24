# schema-driven-test-harness — 设计

## Context

USMP 是模型驱动平台：60 个可用 YANG 模块共用**同一套**前端渲染代码，模块之间的差异纯粹是 schema 数据差异。但当前端到端验证的唯一入口是浏览器（`frontend/tests/staging-smoke.spec.ts`，19 条用例覆盖 vlan/ifm/ntp 三个模块），于是：

- 用最贵的夹具（docker 三镜像 + nginx + 真 Chromium）把同一段渲染代码测了 3 遍，而不是把 60 份数据喂进去；
- 剩余 57 个模块无任何自动化断言，只能人工点击；
- 该 spec 文件里大量注释在处理 teleport 残留、隐藏固定列副本、strict-mode 歧义——成本花在与浏览器搏斗，而非验证业务。

### 实测规模基线（本 change 探索阶段取自运行中 staging，2026-07-24）

| 指标 | 实测值 |
|------|--------|
| 左树叶 / 可用模块 | 65 / 60 |
| schema 节点总数 | 13 479 |
| list 总数 / 其中可写 | 902 / 587 |
| fixture 体积 | 约 3.1 MB（60 份 JSON） |
| 浏览器 E2E 覆盖模块 | 3 |

「587 个可写 list vs 3 个覆盖模块」是本设计的核心动机：这个缺口靠人工点击在数学上填不平。

### 关键约束

- **R05**：前端由 YANG 模型自动渲染，禁止手写固定表单 —— 派生逻辑是纯函数，这是本方案成立的前提。
- **R10**：禁止无关依赖 —— 全部复用既有 ygot/goyang 与 vitest，零新增依赖。
- **T05/T06**：测试设计先行、按层补齐 —— 本 change 本身即是补层动作。
- **T08**：覆盖率棘轮只升不降。
- **§5.6 分层**：新增能力落在 B1/B3（后端）与 F1（前端纯逻辑）。

## Goals / Non-Goals

**Goals:**

- 把「后端 schema 真相 ↔ 前端渲染行为」之间的共享物，从**一个跑起来的浏览器**换成**一份钉住的 fixture**。
- 模块级端到端断言覆盖 **3/60 → 60/60**，且新增模块**自动**进入覆盖（不靠人记得补）。
- 验证代价从「起 docker 全栈 + 装 chromium」降到**无浏览器、无 docker、毫秒级**。
- 产出的 fixture 同时成为后续「设备一致性矩阵」change 的输入，一份数据喂三层。
- 零运行期行为改动：本 change 不改任何用户可见行为，只是把既有行为钉住。

**Non-Goals:**

- **不删任何浏览器用例。** E2E 瘦身必须等新层真的接住后再做（后续 change），否则等于先拆防护网再修网。
- **不做设备一致性矩阵**（下发→回读→幂等→删除）—— 独立 change。
- **不做真机验证** —— 独立 change，需单独设计爆炸半径保险。
- **不做视觉/像素校验** —— 黄金证明的是派生结构，不证明「长得好看」。
- **不改后端 schema 契约、不改前端渲染逻辑** —— 只加导出与校验。

## Decisions

### D1 — fixture 由 Go 工具直调纯函数导出，不走 HTTP

**决定**：新增 `backend/tools/schemadump`，流程为 `yangschema.Load()` → 遍历模块 → `api.BuildYangSchemaNested(mod)` → 写 JSON。

**备选与否决理由**：

| 方案 | 否决理由 |
|------|----------|
| (a) curl 打运行中后端 | 生成依赖 docker 全栈，CI 里重新引入了我们正要卸掉的那条长链 |
| (b) `go test -update` 写 fixture | 生成动作藏在测试里，与仓库既有 `gen-yang`/`gen-crd`/`gen-contract` 三条管线形状不一致，`make` 入口不可见 |
| **(c) 独立 Go 工具 + Makefile 目标** ✅ | 与 `backend/tools/crdgen` 同形状；无 HTTP/无 docker；CI 可直接跑；`make gen-schema-fixtures` 入口显式 |

**代价**：`buildYangSchemaNested` 需从包私有改为导出（`BuildYangSchemaNested`）。这是本 change 对后端源码的**唯一**改动，纯可见性变更、零逻辑改动。

### D2 — 用等值测试保证 fixture 忠实于线上契约

D1 绕开了 HTTP，就必须证明「工具导出的」等于「用户真实拿到的」，否则 fixture 会悄悄与线上契约脱钩，前端黄金全绿而实际渲染是错的——那比没有测试更危险。

**决定**：新增 B3 层等值测试——对每个模块，断言 `BuildYangSchemaNested(mod)` 的序列化结果与 `GET /yang/schema/:module?form=nested` 响应的 `data` 字段**逐字节相等**。这条测试是整个方案的信任锚点，必须覆盖全部模块而非抽样。

### D3 — 确定性已验证，但仍建确定性回归防线

regen-and-diff 门禁的前提是导出确定。**已实测验证**：

- `schema/entry.go:65` `sortedDir()` 把 goyang `Entry.Dir`（map）的键取出后立刻 `sort.Strings`，注释明写 "for deterministic output"；
- `defaultContainer/defaultList/defaultCase.Children()` 返回建树时定序的 `[]Node` slice，查找用独立 `childrenMap`，遍历不碰 map；
- `buildYangSchemaNested` 是对 `Children()` 的纯递归，无 map 遍历；
- 运行中 staging 上对 vlan/ifm/qos/network-instance/bgp/acl 六个模块连拉两次，字节一致。

**决定**：仍新增 B1 确定性测试（同进程连生成两次比对 + 跨进程由 CI 的 regen-and-diff 天然覆盖）。理由：上游 goyang 升级或 schema 包重构可能引入 map 遍历，这条测试是把「确定性」从当前实现细节升格为受保护契约。门禁因此 fail 是**正确行为**，不是误报。

### D4 — 模块集合动态发现，禁止硬编码

**决定**：导出器遍历 `yangschema.Load()` 实际加载的全部模块，fixture 目录内容即模块全集；黄金套件遍历 fixture 目录，不维护模块名单。

**理由**：硬编码 60 会让**新接入的模块默认逃逸出覆盖**——而「新接模块要人工点一遍」正是本 change 要消灭的痛点 A。动态发现让新模块在接入当天自动获得 fixture + 黄金，无需任何人记得补。

### D5 — 黄金只记「派生结论」，不记输入副本

**决定**：黄金内容是派生**结果**，不含 schema 原文：

```
per module:
  tabs         : [{ name, kind }]                     ← deriveTabs
  per list tab : keyField                             ← deriveKeyField
                 columns    [{ name, widget }]        ← deriveColumns（含 cap=9 截断结论）
                 filterable [name]                    ← filterableFields
  fields       : [{ path, widget, required, readonly,
                    hasWhen, mustCount, units,
                    presence, dynamicDefault }]
  tree         : 层级摘要                              ← deriveSchemaTree
```

**理由**：若把 schema 原文塞进黄金，黄金就退化为 fixture 的副本——schema 一动黄金必动，但那个变化 fixture 门禁已经抓了，黄金只是噪声。只记结论时，黄金变动**当且仅当**前端派生逻辑或 schema 的语义影响发生变化，diff 才承载信息。

### D6 — 一模块一黄金文件

**决定**：`frontend/test/golden/__data__/<module>.json`，60 个独立文件，而非单个巨型快照。

**理由**：直接缓解 D-R1（盲签风险）。前端派生逻辑合法变更时，`git diff --stat` 立刻显示「哪 12 个模块受影响」而非「一个 4 万行文件变了」，评审可按模块抽查而非整体盲签。

### D7 — 黄金用结构化 JSON，不用 vitest 内联 snapshot

**决定**：黄金以有序 JSON 落盘、由测试显式读写比对，不使用 `toMatchSnapshot()`。

**理由**：（1）JSON 可被后续「设备一致性矩阵」change 复用为用例清单来源；（2）内联 snapshot 的 `-u` 全量刷新过于顺手，正是盲签的温床；（3）JSON diff 在 PR 界面可读。

### D8 — 黄金不含 i18n 本地化结果

**决定**：fixture 与黄金中的 `label` 一律为后端给出的 raw YANG 节点名；SND i18n 本地化（`useFieldLabels` / snd res）不进黄金。

**理由**：本地化是渲染管线的独立一层，若进黄金，每次 i18n 资源更新都会震动 60 份黄金，制造与派生逻辑无关的噪声。已核对 fixture 实际内容确认 `label` 为 raw（如 `"label": "default-instance"`）。

### D9 — fixture 全量入库 + 体积门禁排除

**决定**：3.1 MB fixture 全量入库，并把 fixture 与黄金目录加入 `pr-size.yml` 排除清单。

**理由**：精简版（剔除 label/units 等呈现字段）会让 fixture 不再等于线上契约，直接摧毁 D2 的忠实性保障。3.1 MB 相对仓库既有 `snd/` 与 `internal/generated/` 属可接受量级。排除清单口径对齐既有 `*.gen.ts` / `internal/generated/**`。

## Risks / Trade-offs

**[R1] 前端派生逻辑合法变更导致 60 份黄金同时变动，评审退化为盲签**
→ 缓解：D5（只记结论，减少变动面）+ D6（一模块一文件，diff 可定位）+ D7（不用 `-u` 顺手刷新）。补充：黄金刷新的 commit 必须在 What/Why/How 的 How 段说明**预期影响哪些模块、为什么**，与实际 diff 对不上即评审打回。

**[R2] 黄金全绿 ≠ 用户看到的界面是对的**
→ 这是**真实边界，不掩饰**。黄金证明的是「schema → 控制台形态的派生是确定且未意外改变的」，不证明「派生结果对用户合理」「控件好用」「视觉正确」。首次生成黄金时的人工审阅是唯一一次语义确认，此后只保回归。语义层与视觉层由后续 change 的无头接触表补，门禁文案必须明写这条边界，禁止用「60/60 覆盖」暗示比实际更强的保证。

**[R3] fixture 陈旧于后端真实 schema**
→ 缓解：CI regen-and-diff 门禁（后端 schema 一动、fixture 不刷即 fail）+ D2 等值测试（fixture 一旦与 HTTP 响应不符即 fail）。两道防线互补：前者防「忘了刷」，后者防「刷错了」。

**[R4] 上游 goyang 升级改变遍历顺序，导致 fixture 大规模变动**
→ 这是**期望行为而非故障**：顺序变化会真实影响前端字段渲染次序，本就该在门禁上暴露、由人确认。D3 的确定性测试保证它不会以「随机偶发」形式出现，而是以「升级即一次性全量变更」形式出现，可一次性评审。

**[R5] 仓库体积增长 3.1 MB，且每次 schema 变更产生大 diff**
→ 接受。相对 fixture 带来的 3/60 → 60/60 覆盖跃迁，以及它同时成为后续两个 change 的共享输入，这个代价明确划算。若未来体积失控，可评估只对 fixture 目录启用 git 属性抑制 diff 渲染，但不改变入库策略（D9 理由不变）。

**[Trade-off] 本 change 不减少任何现有浏览器用例，因此短期内总测试时间是**增加**的**
→ 明确接受。安全的替换顺序只能是「先建新层 → 验证接住 → 再拆旧层」。收益（人工点击成本消除、覆盖 20 倍跃迁）在本 change 即兑现；浏览器瘦身的时间收益在后续 change 兑现。这一点必须对预期讲清楚，不能承诺本次就变快。

## Migration Plan

无运行期迁移——本 change 零运行期行为改动，不涉及部署、无需回滚预案。仅有门禁引入顺序：

1. 导出器 + fixture 入库（此时无门禁，纯新增）；
2. D2 等值测试 + D3 确定性测试转绿（证明 fixture 可信）；
3. 前端黄金套件 + 首次人工审阅黄金内容（R2 所述的唯一一次语义确认）；
4. 开启 CI 漂移门禁与体积门禁排除（此时门禁才有意义）；
5. 上调覆盖率棘轮至实测新水平。

顺序不可颠倒：门禁必须在 fixture 被证明可信**之后**开启，否则会把错误的 fixture 钉成基线。

## Open Questions

- **黄金首次审阅的抽样策略**：60 份黄金全量人工审阅不现实。倾向按 D-1 分类分层抽样——EASY/EASY\* 各抽 2、PATTERN/COND 全审（9 个）、规模 top3（network-instance/qos/ifm）全审。待 apply 阶段确认。
- **fixture JSON 缩进格式**：可读优先（2 空格，约 3.1 MB）vs 体积优先（紧凑，约 1/3 体积但 diff 不可读）。倾向可读优先——本方案的价值恰在 diff 可审。
- **黄金是否纳入 `deriveSchemaTree` 完整层级**：完整树会显著放大黄金体积且与 `tabs` 信息重叠。倾向只记层级摘要（深度、每层节点数、分组名），待实测黄金体积后定。
