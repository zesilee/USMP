# schema-driven-test-harness

## Why

USMP 端到端验证目前的唯一入口是浏览器，但配置面的真实规模是 **60 个可用模块 / 902 个 list（其中 587 个可写）/ 13479 个 schema 节点**，而 `staging-smoke.spec.ts` 实际覆盖 **3 个模块**（vlan/ifm/ntp）。差两个数量级的缺口只能靠人工点击填补，且每次前端派生逻辑改动都要对剩余 57 个模块重来一遍——这在数学上走不完。

根因是**后端 schema 真相与前端渲染行为之间唯一的共享物是「一个跑起来的浏览器」**：一根 `浏览器→nginx→SPA→API→Controller→NETCONF→sim` 的长链，被迫承担了本可由纯数据比对完成的验证。而 60 个模块共用同一套渲染代码，模块间差异是**数据差异**——数据差异不需要浏览器。

把共享物换成**钉住的 schema fixture**，验证即可从长链上卸下来：后端声明「我服务的 60 份 schema 就是这样」，前端声明「这 60 份 schema 派生成这样的控制台」，两侧各自用最便宜的夹具校验，漂移在门禁上暴露。

本 change 是四步替换的第 1 步（地基 + 最高杠杆层），不改动任何运行期行为。

## What Changes

- **新增 schema fixture 导出管线**：`make gen-schema-fixtures` 遍历 `yangschema.Load()` 已加载的全部模块，调用既有 `buildYangSchemaNested` 纯函数导出版本化 JSON fixture 入库（后端为唯一真源）。导出脱离 HTTP/Gin/docker，可在 `go test` 与 CI 中直接跑。
- **新增 fixture 漂移门禁**：CI regen-and-diff——重新生成后若与入库 fixture 有差异即 fail，形状对齐既有 `gen-yang` / `gen-crd` / `gen-contract` 三条管线与 `contract-drift.yml` 工作流。
- **新增前端控制台派生黄金**：对**全部** fixture 模块运行既有纯派生函数（`deriveTabs` / `deriveKeyField` / `deriveColumns` / `filterableFields` / `deriveSchemaTree`）与字段级派生（控件类型、必填、只读、约束、单位），产出结构化黄金快照入库。任一模块的控制台形态变化即门禁失败并指名到模块。
- **覆盖面**：模块级端到端断言 **3/60 → 60/60**，运行代价从「docker 三镜像 + 起栈 + npm ci + 装 chromium + 19 条浏览器用例」降到**无浏览器、无 docker、毫秒级**。
- **体积门禁**：fixture 与黄金目录加入 `pr-size.yml` 排除清单（生成物不计入手写代码体积，对齐既有 `*.gen.ts` / `internal/generated/**` 口径）。
- **覆盖率棘轮**：前端新增黄金套件后按实测新水平上调 `vitest.config.ts` thresholds（T08）。
- **浏览器 E2E 本次不动**：`staging-smoke.spec.ts` 一条不删。瘦身必须等新层真的接住之后，否则等于先拆防护网再修网——留给后续 change。

### 明确不在本次范围

- 设备一致性矩阵（下发→回读→幂等→删除，587 用例）—— 后续 change，spike 已验证自动生成命中率 55/57。
- 浏览器 E2E 瘦身 + 无头接触表 —— 后续 change，依赖本次与设备矩阵先落地。
- 真机 driver 与发布门 —— 后续 change，需单独设计爆炸半径保险。

## Capabilities

### New Capabilities

- `schema-fixture-pipeline`: 把后端已加载 YANG 模块的呈现 schema 导出为版本化 fixture 的生成管线与漂移门禁；fixture 是前端派生黄金与后续设备一致性矩阵的**共享输入**，且是后端 schema 契约的可审计快照。
- `console-derivation-golden`: 对全部 fixture 模块运行前端纯派生函数、把「模块 → 控制台形态」钉成黄金快照的验证契约；保证任一模块的 Tab/列/主键/必填/控件类型/约束派生发生非预期变化时门禁失败并定位到模块。

### Modified Capabilities

（无——后端 schema 契约与前端渲染行为**均不变**。本次只把既有行为钉住，不改任何运行期语义。）

## Impact

- **后端新增**：`backend/tools/schemadump/`（导出工具，形状对齐既有 `backend/tools/crdgen`）；`backend/internal/api/yang_handler.go` 导出 `buildYangSchemaNested` 供工具调用（唯一一处后端源码改动，纯可见性变更、零逻辑改动）。
- **入库生成物**：`backend/testdata/schema-fixtures/<module>.json`（实测 60 份、约 3.1 MB）。
- **前端新增**：`frontend/test/golden/` 黄金套件（F1 层，happy-dom 不必要——纯函数无 DOM）；`frontend/test/golden/__snapshots__/` 或结构化 JSON 黄金入库。
- **构建/门禁**：`Makefile` 新增 `gen-schema-fixtures`；CI 新增 fixture 漂移校验（并入 `contract-drift.yml` 或同形状新工作流）；`pr-size.yml` 排除清单追加两条。
- **依赖**：零新增第三方依赖（R10）。导出走既有 ygot/goyang 加载链，黄金走既有 vitest。
- **测试分层**（§5.6）：B1（导出工具确定性/全模块覆盖/降级）、B3（导出结果与 HTTP `GET /yang/schema/:module?form=nested` 等值——证明 fixture 忠实于线上契约）、F1（黄金派生 60/60 + 派生函数负路径）。
- **风险**：fixture 与黄金均为大体量生成物，前端渲染逻辑合法变更会导致 60 份黄金同时变动——需保证黄金结构化、键有序、diff 可读，否则评审会退化为盲签。design.md 中处理。
