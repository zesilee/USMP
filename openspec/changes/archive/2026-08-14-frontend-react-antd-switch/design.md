## Context

USMP 前端为 YANG 模型驱动的**运行时渲染**控制台（R05）：表格的列、表单的字段、校验的规则，全部由后端 YANG schema 在运行时派生，无任何手写固定表单。因此换库成败不取决于组件数量，而取决于新组件库能否支撑**运行时动态列**与**运行时动态校验规则**。

**关键前提：全新项目，无存量继承包袱。** 不需要向后兼容、不需要与 Vue 实现并行运行、不需要灰度切换与回滚窗口。这一前提大幅简化了方案——本设计据此按「原地重建」组织，而非「迁移」。

现有代码盘点（决定哪些沿用、哪些重建）：

| 分类 | 规模 | 处置 |
|------|------|------|
| `src/utils`（派生/diff/xpath/leafref…） | 15 文件 1368 行，**零框架依赖** | **沿用** |
| `src/types`（openapi 契约生成物） | 3 文件 1532 行 | **沿用** |
| `src/api`（axios） | 242 行 | **沿用** |
| `test/{utils,stores,composables}` | 33 文件 3220 行 | 大部分沿用 |
| Playwright E2E + `data-test` 命名 | 414 行 | **沿用** |
| i18n 词表 + 派生黄金 + schema fixture | — | **沿用** |
| `src/composables` | 7 文件 965 行 | 业务语义沿用，实现改写为 hooks |
| `src/stores`（Pinia） | 5 文件 485 行 | 结构参考，改 zustand |
| `src/components` + `src/views` | 36 文件 7017 行 | **重建** |
| `test/{components,views,browser}` | 51 文件 6304 行 | **重建** |

组件用量：Element Plus 38 种，承重三件为 `el-table`（46 处）、`el-form`（24 处）、`el-select`（19 处），其余 35 种为换皮级替换。

范式差异（实测计数，决定重建工作量）：作用域插槽 48（→ render 函数）、`v-model` 37（→ 受控）、`computed` 92 + `watch` 26、命令式 API 36（`ElMessage` 19 + `ElMessageBox` 17，其中 17 处为 `await confirm` 写法）、`reactive` 1（`useConfigForm.formData`，被全量表单编排依赖）。

约束：R05、R06/T01/T05/T06（测试先行、按层补齐、缺层禁合）、T08（覆盖率不下降）、R11/R12（视觉与图标规范）、TM04（PR ≤1000 行）、W01（禁在 main 开发）。

## Goals / Non-Goals

**Goals:**
- 前端运行于 React 19 + antd 6.6，产品形态与能力与现有控制台一致。
- 建立 UI 适配层，使组件库成为**单点可替换**依赖（当前 antd，后继可能 EviewUI）。
- 保住 R05：运行时动态列 / 运行时动态校验规则能力不因换库退化。
- 测试四层（F1–F4）在新栈上建齐，覆盖率不低于现有基线；派生黄金结论零漂移。
- 基础设施改动最小化——路径不变，只换工具链命令。

**Non-Goals:**
- **不做双轨并行、不做灰度切换、不保留回滚窗口**（全新项目，无线上存量）。
- 不做功能增删与交互改良——发现的改进点记入 follow-up，不在本 change 内动手。
- 不改后端、不改 HTTP 契约、不改部署形态。
- 不引入 EviewUI（本次仅为其预留适配层；真正接入另开 change）。
- 不做视觉重设计（antd 主题做到与现有观感接近即可，不追求像素级一致）。
- 不迁移 Storybook 既有故事内容（仅保证 CI 工作流不失效）。

## Decisions

### D1 原地重建，不做双轨

**决定**：在 `frontend/` 目录内直接以 React 实现取代 Vue 实现。先一个纯删除 PR 清场，随即脚手架 PR 建起 React 骨架，之后逐块重建。

**理由**：全新项目无存量用户与线上负担，双轨并行的唯一收益（可回退、可对比）在此不成立，而成本（两套依赖、两套 CI、双份维护）是实打实的。

**关键收益**：目录路径不变 ⇒ 6 个 CI 工作流、3 个 git 钩子、`Dockerfile`、`build-release.sh`、Makefile 目标**全部不需要改路径**，只需调整内部工具链命令（`vue-tsc`→`tsc`、Storybook 框架包、Vitest 插件）。相较「新目录并行」方案，基础设施改动量下降一个数量级。

**代价**：清场 PR 与脚手架 PR 之间存在短暂空窗（仓库无可运行前端），期间前端 CI 作业降级为 no-op。全新项目可接受，且窗口以「一个 PR」计。

**备选与否决**：
- *新目录 `frontend-react/` 并行、末期切换*：为不存在的回退需求付出双份成本，并使全部基础设施路径需要改两次。否决。
- *同 app 内 Vue/React 桥接共存*：引入桥接层，双响应式系统互调，且桥接层本身用完即废。否决。

### D2 UI 适配层：业务代码禁止直接 import 组件库

**决定**：新增 `src/ui/`，对外导出项目实际使用的控件面；业务代码 SHALL 只从 `@/ui` 导入，SHALL NOT 出现 `from 'antd'`，由 ESLint 规则 + 守护测试双重拦截。

**理由**：用户已明确「将来可能换 EviewUI」。适配层把「换库」的爆炸半径从数千行业务代码收敛到一个目录，额外成本约 5%，一次即回本。

**边界**：适配层是**薄转发层**，只做三件事——① 控件重导出；② 抹平已知 API 差异（如命令式 API 的 Promise 化）；③ 收口图标与主题令牌。**禁止塞业务逻辑**，否则换库时它自己就成了负债。

### D3 承重能力先验：垂直切片作为闸门

**决定**：脚手架就绪、纯逻辑层与表单编排到位后，**第一个业务任务是打通「模块控制台列表页」单页**（同时用到 Table 动态列 + Form 动态 rules + Select），作为 R05 能力验证探针。以真实的 68 模块 schema fixture 驱动，非玩具示例。**此闸门不通过即暂停后续重建并重新评估方案。**

**理由**：antd Table 的列是 `columns` 配置数组（比 Element Plus 的子组件声明式更利于运行时生成），Form 的 `rules` 亦可运行时生成——纸面可行，但必须实测。风险前置，避免建到第 20 个页面才发现撞墙。

### D4 沿用框架无关资产（这不是继承包袱）

**决定**：`src/utils` 派生纯函数、`src/types` 契约生成物、`src/api`、i18n 词表、派生黄金与 fixture、E2E 用例与 `data-test` 命名——**全部沿用，且派生纯函数逐字节搬运、禁止顺手重构**。

**理由**：「全新项目」指的是不背向后兼容与并行运行的包袱，**不是「凡旧必弃」**。这些资产与前端框架完全无关，是已被测试覆盖的产品逻辑本身；重写它们只会引入新缺陷、丢失既有测试覆盖。派生黄金（GD-01）在新栈上零漂移，是「换库没换掉产品行为」最强的客观证据。

**边界**：沿用的是**框架无关的逻辑与数据**，不沿用任何 Vue 组件结构、组件命名或 Vue 化的状态写法。

### D5 派生链不做 memo，纯函数每次重算

**决定**：`visibleFields → flatFields → editableFlat → diff → submittable` 这条派生链在 React 侧**不套 `useMemo`**，每次渲染直接重算。

**理由**：均为纯函数，输入规模是「单表单几十个字段」，重算成本可忽略；而手写依赖数组漏项会产生**陈旧值 bug**，不报错且极难定位。性能真出问题时再针对性加 memo，届时有 profile 数据支撑。

**备选**：jotai/signals 类自动派生方案——引入额外依赖与心智模型，收益不足。记为 open question。

### D6 表单状态：不可变更新，且「删键」必须真删

**决定**：`useConfigForm` 业务语义全部保留，`formData` 由 `reactive` 改为不可变 state。**凡表达「该键不存在」的场景，SHALL 使用解构删键，SHALL NOT 置为 `undefined`**：

```ts
// 正确：键真正消失
setForm(prev => { const { [k]: _drop, ...rest } = prev; return rest })
// 错误：键仍在，payload 语义被破坏
setForm(prev => ({ ...prev, [k]: undefined }))
```

**理由**：项目用「键不存在 = 节点不存在」表达 presence 容器关闭、choice 非激活分支清空、字段不入 payload（FE-12 / FE-08 / FE-15 语义）。置 `undefined` 时 `in` 判断仍为真、`Object.keys` 仍含该键，会导致**下发多余字段且不报错**——真机可能因 `unknown-element` 拒绝整次配置。须专门写回归测试钉死（新增 FE-27）。

**备选**：不引入 react-hook-form——校验规则由 YANG 运行时生成、字段面本身动态，第三方表单库的静态字段注册模型反而是阻碍。

### D7 命令式 API：Promise 适配器收口在 `src/ui`

**决定**：轻提示与确认框统一封装为 `src/ui/feedback` 的 `toast()` 与 `confirm(): Promise<boolean>`，内部基于 antd `App.useApp()` + 顶层容器实现。

**理由**：antd 5+ 要求通过 hook 取得实例，不能在普通函数里裸调。收口后业务写法保持 `if (await confirm(...))` 的直观形态，且换库时只改这一个文件。

### D8 测试四层建齐

| 层 | 实现 |
|----|------|
| F1 纯逻辑 | Vitest + happy-dom（沿用现有用例） |
| F2 组件 | `@testing-library/react`（重建） |
| F3 真浏览器 | Vitest Browser Mode（配置换 React 插件） |
| F4 E2E | Playwright（沿用用例，依赖 `data-test` 命名保留） |

覆盖率阈值以现有基线为起点，SHALL NOT 下调（T08）。

### D9 视觉：以现有界面为参照，不追求像素级一致

**决定**：antd 主题令牌对齐现有 CSS 变量的主色 / 中性色 / 圆角 / 间距四组即可；页面布局形态以现有界面为参照实现。

**理由**：antd 与 Element Plus 的默认排版存在系统性差异，硬追像素级一致成本高、收益低。R11（禁 AI 陈词滥调视觉）与 R12（禁 emoji 图标）不变。

## Risks / Trade-offs

| 风险 | 缓解 |
|------|------|
| **antd Table 撑不住运行时动态列 / Form 撑不住运行时动态 rules（触及 R05 命门）** | D3 垂直切片前置为闸门，以 68 模块 fixture 实测；未通过即暂停并重新评估 |
| **`reactive`→不可变更新丢失「键不存在」语义，静默下发多余字段** | D6 明确写法 + FE-27 三场景回归测试先行（红灯）+ ESLint 拦截伪删键写法 |
| `useMemo` 依赖数组漏项导致陈旧值 | D5 不做 memo，从根上消除 |
| 清场后至脚手架就位前的 CI 空窗 | 空窗期前端作业降级为 no-op，脚手架 PR 立即恢复；窗口以一个 PR 计 |
| 沿用的派生纯函数被「顺手重构」污染黄金证据链 | D4 明确逐字节搬运；黄金零漂移作为该 PR 的硬门槛 |
| 界面观感与现有形态偏离过大 | D9 四组令牌对齐 + 以现有页面截图为参照逐页比对 |
| PR 体积超 TM04 上限 | 任务按页面/模块切分，每项预估 ≤1000 行；纯删除 PR 走体积豁免 |
| Storybook 故事作废 | 列为 Non-Goal，仅保 CI 工作流不失效 |

## 实施顺序

1. **清场**：删除 Vue 实现与其依赖（纯删除 PR，体积豁免），前端 CI 作业临时降级。
2. **脚手架 + 适配层**：Vite + React 19 + antd 6.6 + TS；`src/ui/` 与其门禁先行；CI 恢复。
3. **纯逻辑层就位**：`utils` / `types` / `api` 与其单测；派生黄金零漂移。
4. **状态与 i18n**：zustand + 自研 i18n 薄层（清场 PR 已铺路：与旧 vue-i18n 同形的 `i18n.global.t` API + `{name}` 插值，词表键值原样沿用；React 侧语言切换在其上加响应式externals，少一个运行时依赖）。
5. **表单编排**：`useConstraintEngine` / `useConfigForm` 改写为 hooks；FE-27 红灯先行。
6. **垂直切片闸门**：模块控制台列表页打通，实测 Table 动态列 + Form 动态 rules。**不通过则暂停。**
7. **逐块重建**：控制台其余 Tab → 布局与导航 → 其余页面 → 变更集与批量链路。
8. **测试补齐**：F3 真浏览器 + F4 E2E 全绿 + 覆盖率达标。
9. **工具链收尾**：typecheck / Storybook / Dockerfile / `build-release.sh` / Makefile / git 钩子的命令调整。
10. **文档与归档**：`CLAUDE.md` §3 技术栈、`frontend/TESTING.md`、spec sync、change archive、记忆更新。

## Open Questions

1. **antd 主题令牌对齐粒度**——四组（主色/中性色/圆角/间距）是否足够？待垂直切片出图后由用户目视确认。
2. **Storybook 是否重建故事**——当前列为 Non-Goal；若团队依赖度高则另开 change。
3. **EviewUI 接入时机与其 React 版本约束**——若 EviewUI 仅支持 React 18，本次锁 React 19 将在接入时产生降级成本；待其 `package.json` / d.ts 到手后确认（不阻塞本次）。
4. **清场空窗期的 CI 处理方式**——前端作业「临时 no-op」与「整体跳过」二选一，倾向前者（保留工作流骨架，减少恢复时的改动面）。
