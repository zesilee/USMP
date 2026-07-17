# ownership-hard-lock — 任务

> 单 PR 交付（预估 <600 行）。TDD：每步先测试后实现（T05/T01）。

## 1. 后端硬锁 + force

- [x] 1.1 【测试先行】B3 handler 测试红灯：POST 命中认领路径无 force → 信封 409 + data.intents + 不下发不记审计；force=true → 放行 + ownershipWarning + 审计 Forced；DELETE 同锁；未认领路径照常；兄弟路径不受锁（负路径）
- [x] 1.2 `ErrorWithData` 响应助手（信封同构 + data）+ 单测
- [x] 1.3 SetConfig/DeleteConfig 入口门禁：解析 force → 查 `intent.DefaultOwnership.Owners` → 无 force 命中即 409 早拒（编解码前）——绿灯
- [x] 1.4 audit.Record + Forced/ForcedOwners 字段：memory/CRD store 透传单测先行（含 CRDStore spec 映射、/logs DTO 透出）、`deploy/crds/auditrecords.core.usmp.io.yaml` +可选属性
- [x] 1.5 swagger 注解补 force query 参数与 409 响应，`make`（或对应 target）再生前端契约生成物（1a 漂移门禁）

## 2. 前端阻断确认流

- [x] 2.1 【测试先行】F1：`confirmOwnershipOverride` 助手单测（409+intents 识别 / 非 409 透传 / 确认 true / 取消 false）
- [x] 2.2 实现助手（ElMessageBox.confirm，列意图 + 覆盖警示，确认按钮「强制下发」）
- [x] 2.3 F2：useConfigSubmit 与 ModuleFormTab 的 409 分支组件测试（mock 确认 → force 重发；取消 → 不置 error）
- [x] 2.4 setConfig/deleteConfig API 追加可选 force 参数；两处调用点接入确认流——绿灯
- [x] 2.5 覆盖率阈值核对：80.69/76.15/74.7/81.4 全过现阈值（80/75/73/80）；按 frontend-ci-gotchas 教训（CI 比本地低约 1 点、贴边即 flaky）不冒进上调

## 3. 验证与交付

- [x] 3.1 后端 `go test ./... -race` 全绿；前端单测 + typecheck 全绿；`openspec validate ownership-hard-lock` 通过
- [x] 3.2 含 frontend/ 改动：本地 `make e2e-local` Playwright staging smoke 全绿（无 docker 机器才许 USMP_SKIP_E2E=1）
- [x] 3.3 code review（go-code-review-check）通过 → What/Why/How 原子提交（≤500 行/commit）
- [ ] 3.4 push + PR（CI 全绿后合入，用户已授权 merge-on-green）

## 4. 收尾

- [ ] 4.1 `/opsx:sync` delta 合入主 spec（config-api BR-11 / operation-audit OA-01 / frontend FE-18）
- [ ] 4.2 `/opsx:archive` 归档 change；`openspec/tasks/ownership-hard-lock.md` 置 completed 并归档
- [ ] 4.3 更新记忆（k8s-paas-deployment-constraints：硬锁二期完成，follow-up 清零）
- [ ] 4.4 清理 worktree（§6.3）
