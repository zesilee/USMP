## 1. Worktree 与基线

- [ ] 1.1 创建 worktree `worktree-schema-driven-test-harness`（§6.1），确认已在 `.gitignore`
- [ ] 1.2 跑基线：`make compliance` 全绿、`cd frontend && npm run test` 全绿，记录当前覆盖率实测值（后端 `.coverage-baseline`、前端 vitest thresholds）作为棘轮起点

## 2. 导出器（B1 红绿循环，SF-01/SF-02）

- [ ] 2.1 **先写测试**：`backend/tools/schemadump` 的 B1 表格驱动测试——(a) 导出模块数等于 `yangschema.Load()` 已加载模块数；(b) 每份产出含 module/title/vendor 与非空 fields 树；(c) 模块集合来自动态发现，测试内不出现硬编码模块名单。此时应为红
- [ ] 2.2 把 `buildYangSchemaNested` 导出为 `BuildYangSchemaNested`（纯可见性变更，零逻辑改动），更新 `yang_handler.go:233` 调用点
- [ ] 2.3 实现 `backend/tools/schemadump`：`yangschema.Load()` → 遍历模块 → `api.BuildYangSchemaNested` → 写 JSON（2 空格缩进、末尾换行，D9/Open-Q2 口径）。形状对齐 `backend/tools/crdgen`。2.1 转绿
- [ ] 2.4 **先写测试**：SF-02 确定性测试——同进程连续导出两次，断言逐字节相同；再跑一次断言与首次一致。此时应为红或直接绿（依赖既有 `sortedDir` 定序），无论如何测试必须存在
- [ ] 2.5 确认 2.4 绿；若红则定位非确定源（优先查 map 遍历）并修复，禁止用排序掩盖真实语义顺序

## 3. 忠实性防线（B3，SF-03）

- [ ] 3.1 **先写测试**：`backend/internal/api` 下 B3 等值测试——对**每个**已加载模块，断言 `BuildYangSchemaNested(mod)` 序列化结果与 `GET /yang/schema/:module?form=nested` 响应 `data` 字段逐字节相等。禁止抽样。此时应为红（等值逻辑未接）
- [ ] 3.2 使 3.1 转绿；若发现 HTTP 路径与导出路径存在真实差异，**先停下说明差异原因**再决定对齐方向，禁止改断言迁就实现
- [ ] 3.3 **负路径测试**：构造导出路径与 HTTP 路径不一致的场景，断言等值测试确实失败（防止断言写成恒真）

## 4. fixture 入库与生成入口

- [ ] 4.1 执行导出，产出 `backend/testdata/schema-fixtures/<module>.json`（预期 60 份、约 3.1 MB）并入库
- [ ] 4.2 `Makefile` 新增 `gen-schema-fixtures` 目标，形状与文案对齐既有 `gen-yang` / `gen-crd` / `gen-contract`
- [ ] 4.3 人工抽查 3 份 fixture（vlan 小、ifm 中、network-instance 大）内容合理：字段树结构、readonly/required/pattern 等元数据齐全、label 为 raw YANG 节点名（D8）
- [ ] 4.4 提交（What/Why/How 三段式）——此时尚未开门禁，属纯新增

## 5. 前端黄金套件（F1 红绿循环，GD-01/GD-02/GD-03）

- [ ] 5.1 **先写测试**：`frontend/test/golden/` 套件骨架——遍历 fixture 目录动态发现模块（禁止硬编码名单），对每个模块调用既有派生纯函数并与 `__data__/<module>.json` 比对；黄金缺失即失败。此时应为红（无黄金数据）
- [ ] 5.2 实现派生结论收集器：tabs、每个 list tab 的 keyField/columns/filterable、字段级（path/widget/required/readonly/hasWhen/mustCount/units/presence/dynamicDefault）、tree 层级摘要。严格按 D5——**不含 schema 原文、不含 i18n 本地化标签**
- [ ] 5.3 **先写测试**：GD-02 反向断言——(a) 构造仅改 schema 非派生相关字段的 fixture，断言黄金不变；(b) 构造 i18n 资源变化场景，断言黄金不变；(c) 构造派生逻辑变化场景，断言黄金失败
- [ ] 5.4 **先写测试**：GD-03 定位性断言——一处派生变更后，未受影响模块的黄金文件逐字节不变
- [ ] 5.5 生成全部模块黄金到 `frontend/test/golden/__data__/<module>.json`，一模块一文件、键有序（D6/D7）。5.1/5.3/5.4 转绿
- [ ] 5.6 实测黄金总体积；若 `deriveSchemaTree` 完整层级导致体积失控，按 Open-Q3 降级为层级摘要并记录实测数据

## 6. 首次黄金人工审阅（R2 所述唯一一次语义确认）

- [ ] 6.1 按分层抽样审阅黄金内容（Open-Q1 待定稿口径）：EASY/EASY\* 各抽 2、PATTERN 7 个与 COND 2 个全审、规模 top3（network-instance/qos/ifm）全审
- [ ] 6.2 记录审阅结论到 change 内（发现的派生不合理项单独列出——**不在本 change 修**，转后续 issue/change，避免范围扩张）
- [ ] 6.3 提交黄金与套件（What/Why/How 三段式）

## 7. 门禁开启（SF-04；顺序不可提前）

> 前置硬门槛：第 2/3/5 组测试全绿，即 fixture 已被证明可信。否则会把错误的 fixture 钉成基线。

- [ ] 7.1 CI 新增 fixture 漂移校验（regen-and-diff）——并入 `contract-drift.yml` 或同形状新工作流，失败信息须指出差异模块
- [ ] 7.2 **验证门禁真的会拦**：故意改一处 schema 输出不刷 fixture，确认 CI 红；手工篡改一份 fixture，确认 CI 红。验证后还原
- [ ] 7.3 `pr-size.yml` 排除清单追加 `backend/testdata/schema-fixtures/**` 与 `frontend/test/golden/__data__/**`，口径对齐既有 `*.gen.ts` / `internal/generated/**`
- [ ] 7.4 前端黄金套件接入 `frontend-ci.yml`（F1 层，随 `npm run test` 一并执行，无需新 job）

## 8. 覆盖率棘轮与文档（T08）

- [ ] 8.1 跑 `npm run test:coverage` 与后端覆盖率，取实测新水平上调 `vitest.config.ts` thresholds 与 `backend/.coverage-baseline`
- [ ] 8.2 `frontend/TESTING.md` 补充黄金层定位：属 F1，说明它证明什么（派生确定性）与**不证明什么**（语义/视觉正确性），口径严格对齐 GD-04，禁止用覆盖模块数暗示更强保证
- [ ] 8.3 `CLAUDE.md §5.6` 改动类型→必补层表格追加一行：「新增/修改控制台派生逻辑 → 黄金套件（全模块）」

## 9. 完成分支

- [ ] 9.1 `make compliance` 全绿 + `npm run test` 全绿 + `npm run typecheck` 零错
- [ ] 9.2 跑一次 `make e2e-local` 确认既有浏览器 E2E **一条未破**（本 change 承诺零运行期改动，E2E 必须原样通过）
- [ ] 9.3 `go-code-review-check` 技能通过（T04）
- [ ] 9.4 §6.3 完成分支流程：push + PR，等 CI required checks 全绿
- [ ] 9.5 合入后 `/opsx:sync` 把两份 delta spec 并入主 spec，再 `/opsx:archive` 归档
