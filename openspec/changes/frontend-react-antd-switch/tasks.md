> **执行约定**：每个 `##` 任务组对应一个独立 PR（TM04 ≤1000 行，超出即再拆；纯删除 PR 走体积豁免）。每项按 §5.6「改动类型→必补层」补齐测试，缺层=未完成（T06）。测试先行（T05），Bug 先写回归（T07）。
> **全新项目前提**：无双轨并行、无灰度切换、无回滚窗口（design D1）。唯一硬闸门为第 7 组的 R05 能力验证。
> **窗口红灯声明**（清场 PR 评审确认）：`npm run build` 在窗口期为显式报错脚本 → `frontend/Dockerfile`、`scripts/e2e-smoke.sh`、`scripts/build-release.sh`、合并后 `e2e-staging` 工作流在脚手架恢复 build 前**预期红**；窗口期含 frontend 改动的推送使用 `USMP_SKIP_E2E=1` 一次性豁免（超出其"无 docker 机器"常规口径，依据即本声明与 design D1 空窗决策）。豁免分两段：build 断链段（4/6~6/6，Docker 构建即失败）；页面重建段（脚手架后 build 已恢复、但 staging-smoke 断言的页面尚未对等，豁免持续至 tasks 12.3 E2E 全绿即停用）。

## 1. 准备与隔离

- [x] 1.1 `EnterWorktree` 建立隔离环境（§6.1 W01），记录 worktree 与本 change 的对应关系
- [x] 1.2 执行基线测试（后端 `go test ./...` + 前端全量），记录基线覆盖率数值作为新栈阈值起点（T08）
- [x] 1.3 备份沿用资产清单并核对齐全：`src/utils`、`src/types`、`src/api`、`test/{utils,stores,composables}`、E2E 用例、i18n 词表、派生黄金与 schema fixture

## 2. 清场（PR-1，纯删除，体积豁免）

- [x] 2.1 删除 `src/{components,views}` 全部 Vue 实现（7017 行）与 `src/{router,i18n,stores,composables}` 的 Vue 化实现
- [x] 2.2 删除 `test/{components,views,browser}` 组件层测试（6304 行）
- [x] 2.3 移除 vue / vue-router / pinia / vue-i18n / element-plus / @vue/test-utils / vue-tsc / @vitejs/plugin-vue / @storybook/vue3-vite 依赖
- [x] 2.4 前端 CI 作业临时降级为 no-op（保留工作流骨架，design Open Question 4），git 钩子前端分支同步降级
- [x] 2.5 确认沿用资产（第 1.3 项清单）**未被误删**

## 3. 脚手架与 UI 适配层（PR-2）

- [x] 3.1 建 React 骨架：Vite + React 19 + TypeScript + antd 6.6；Node 引擎约束保持 `>=20.12.0`
- [x] 3.2 `src/ui/` 适配层：逐项显式声明导出面（覆盖 38 种在用控件的对应物），禁整包透传（FA-02）
- [x] 3.3 适配层 `feedback`：`toast()` 与 Promise 化 `confirm()`（FA-03，D7）— **F1**
- [x] 3.4 适配层图标与主题令牌收口：19 个在用图标映射 + 缺失图标规范占位（FA-04，R12）— **F1**
- [x] 3.5 门禁：守护测试硬拦业务代码直接 import 组件库与整包透传（FA-01/FA-02，pre-commit+CI 双跑）；ESLint 规则部分推迟至 lint 体系引入（项目现无 ESLint，守护测试已是硬拦截）— **守护测试**
- [x] 3.6 Vitest（happy-dom）+ `@testing-library/react` 配置就位；覆盖率阈值文件按 1.2 基线初始化
- [x] 3.7 `tsc --noEmit` typecheck 脚本就位；**前端 CI 作业恢复正常**（解除 2.4 降级）

## 4. 纯逻辑层就位（PR-3）

- [x] 4.1 `src/types` 与 `gen:api` 契约生成链路跑通（openapi-typescript 不变）
- [x] 4.2 `src/utils` 15 文件就位——**逐字节沿用，禁止顺手重构**（D4）
- [x] 4.3 `src/api` axios 层就位
- [x] 4.4 `test/utils` 18 文件跑通 — **F1**
- [x] 4.5 派生黄金与 schema fixture 在新栈运行并确认**零漂移**（GD-01，本组硬门槛）

## 5. 状态与 i18n（PR-4）

- [x] 5.1 zustand 重建 5 个 store（action/state 命名沿用，便于与既有测试对齐）
- [x] 5.2 `test/stores` 7 文件跑通 — **F1**
- [x] 5.3 i18n 薄层接 React 响应式（清场 PR 已建同形 API 基座）：locale 词表**键值原样沿用不重命名**（D4，UI-02）
- [x] 5.4 语言切换入口 + localStorage 持久化 + 适配层 locale 联动（UI-01）— **F1 + F2**
- [x] 5.5 中文硬编码扫描脚本跑通，确认零残留（UI-02 判据）
- [x] 5.6 `useFieldLabels`（snd res 懒加载，UI-03）就位；`make sync-snd-i18n` 目标验证 — **F1**

## 6. 表单编排 hooks（PR-5）

- [x] 6.1 **先写红灯测试**：FE-27 三场景（presence 关闭 / choice 切分支 / dynamicDefault 留空）断言键**真正消失**（键枚举判定，非取值判定）— **F1，T05 测试先行**
- [x] 6.2 `useConstraintEngine` 重建（when 显隐 / must 校验 / xpath 求值）— **F1**
- [x] 6.3 `useConfigForm` 重建为 hooks：不可变更新，删键一律解构（D6）— **F1**
- [x] 6.4 ESLint 规则拦截 `{...prev, [k]: undefined}` 伪删键写法（D6 护栏）— **守护测试**
- [x] 6.5 `test/composables` 8 文件跑通 — **F1**
- [x] 6.6 payload 口径（`visiblePayload` / `changedPayload` / `clearedKeys`）与既有语义逐项核对 — **F1**

## 7. 垂直切片：模块控制台列表页（硬闸门，PR-6）

- [x] 7.1 `FieldRenderer` 重建：递归渲染 + 类型→控件分派（8 种类型全覆盖），全部控件经适配层（FE-01）— **F2（含 add/edit/remove 与校验错误态）**
- [x] 7.2 列表 Tab 打通 **Table 运行时动态列**：列设置显隐、排序、enum/boolean 表头筛选、多选、自定义单元格（状态点/enum 标签/when 不可见占位）— **F2**
- [x] 7.3 表单打通 **运行时动态 rules**：pattern / range / required / must 由 YANG 元数据现场生成 — **F2**
- [x] 7.4 以 68 模块 schema fixture 驱动跑通 7.2/7.3（非玩具示例）— **F2**
- [x] 7.5 **闸门评审**：R05 能力确认通过方可继续；未通过则暂停重建、回到 design 重新评估（D3）

## 8. 控制台其余部分（PR-7、PR-8）

- [x] 8.1 `ItemDetailPane` 详情同屏编辑区 + 二级 Tab 派生（FE-21）；顺带拍板 range 违例是否入 blocked 门禁（rangeViolations 已备，评审 #9）与行内 touched/违例计数 — **F2**
- [x] 8.2 `ModuleFormTab` 表单 Tab（FE-10）— **F2**
- [x] 8.3 `RpcExecuteTab` + 高危 rpc 二次确认（FE-19/FE-20）— **F2**
- [x] 8.4 只读 state 子树降级（FE-14）+ 节点不支持占位态 + 单行状态读合并 — **F2**
- [x] 8.5 列表双模式分页（阈值自适应服务端分页，FE-25）— **F2**
- [x] 8.6 行删除入变更集（FE-16）+ 归属徽标（FE-18）— **F2**

## 9. 布局与导航（PR-9）

- [x] 9.1 左树：模块级展开、搜索过滤、rpc 入树、双语名（FE-13）— **F2**
- [x] 9.2 Header / 侧边栏 / 面包屑 / 语言切换入口 — **F2**
- [x] 9.3 react-router 路由表与重定向（对齐现有路由契约）— **F1 + F2**

## 10. 其余页面（PR-10、PR-11）

- [x] 10.1 设备页（列表、表单、离线态）— **F2**
- [x] 10.2 Dashboard（图表库届时再引入——echarts 已随清场移除死依赖）；**必须**在 `src/composables/useFleetOverview.ts` 同路径重建 hook 并恢复 `deriveOverview` 导出（沿用套件 `test/composables/useFleetOverview.test.ts` 复活，vitest include 同步加回）— **F1 + F2**
- [x] 10.3 日志页（操作类型模型驱动派生，FE-26）— **F2**
- [x] 10.4 设置页 — **F2**
- [x] 10.5 业务网络配置控制台页（FE-17）— **F2**

## 11. 变更集与批量链路（PR-12）

- [ ] 11.1 变更集 store + 批量工具栏（FE-23）— **F2**
- [ ] 11.2 `DiffPreview` / 提交对话框 / 试运行对话框 / 变更内容对话框（FE-03）— **F2**
- [ ] 11.3 对账进度与新鲜度展示；**含读路径新鲜度埋点恢复**（getConfig 响应 cache_age/ttl/source → useFreshnessStore.record，评审 #6 提醒旧 requestRows 有此埋点）— **F2**

## 12. 测试补齐与门禁达标（PR-13）

- [ ] 12.1 F3 真浏览器测试建齐：select 弹层/teleport、嵌套 list 真实增删改 — **F3**
- [ ] 12.2 `data-test` 属性逐项核对，确认 E2E 选择器零遗漏
- [ ] 12.3 Playwright E2E 全绿（414 行沿用）— **F4**
- [ ] 12.4 覆盖率四项阈值达到或高于 1.2 记录的基线（T08），达标后同步上调阈值文件
- [ ] 12.5 `make e2e-local` 全栈（docker 编排）冒烟全绿

## 13. 工具链与基础设施收尾（PR-14）

- [ ] 13.1 6 个 CI 工作流内部命令调整（路径不变）：`frontend-ci`、`frontend-browser-tests`、`frontend-storybook`、`contract-drift`、`e2e-staging`、`pr-size`
- [ ] 13.2 3 个 git 钩子命令调整：`pre-commit`（前端单测）、`pre-push`（e2e smoke）、`commit-msg`/pr-size 排除清单两处同步
- [ ] 13.3 `frontend/Dockerfile` 构建命令调整
- [ ] 13.4 `scripts/build-release.sh` 静态站打包验证（产物形态不变）
- [ ] 13.5 Makefile `gen-contract` / `sync-snd-i18n` / `staging-up` 验证
- [ ] 13.6 Storybook 框架包切换，保证工作流不失效（故事内容不迁移，Non-Goal）

## 14. 文档与归档（PR-15）

- [ ] 14.1 更新 `CLAUDE.md` §3 技术栈表（前端行）与 §7.2 前端技能表述
- [ ] 14.2 更新 `frontend/TESTING.md` 四层描述为新栈实现
- [ ] 14.3 `/opsx:sync`：delta spec 合并入主 spec（`frontend`、`ui-i18n`、新增 `frontend-ui-adapter`）
- [ ] 14.4 `/opsx:archive`：归档本 change
- [ ] 14.5 记忆归档（§13 MEM01/MEM04，单独 commit）：新增 React 栈记忆条目，更新受影响的既有条目（`frontend-contract-gen`、`nce-console-redesign`、`test-governance-military-rules` 等）
- [ ] 14.6 follow-up 债登记：Storybook 故事重建、主题令牌对齐粒度、EviewUI 接入时机与 React 版本约束（design Open Questions 1–4）
