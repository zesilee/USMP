# Tasks — adopt-snd-baseline

> 顺序即依赖。spec delta 已先行（R17）。worktree 隔离（§6）。按 D6 拆 2 个 PR。

## PR-1：snd 入库 + 切源 regen + 测试适配

- [ ] 1.1 `cp -r /home/leezesi/snd snd/` 入库（原样目录结构）；`.github/workflows/pr-size.yml` 排除清单加 `:(exclude)snd/**`（vendored 上游制品，与 generated 同理；commit 说明明示此门禁变更）
- [ ] 1.2 `backend/internal/generated/huawei/gen.conf` `yang_path` 改 `snd/ce6866p-yang`；`backend/internal/yangschema/load.go` go:generate `-path` 同步；跑 `make gen-yang VENDOR=huawei` + `go generate ./internal/yangschema/`
- [ ] 1.3 红灯收集：`go build ./... && go test ./... -race` 全量跑一遍，记录编译错/断言挂的完整清单（预期集中在 hwfix、vlan_hardening、ifm 集成、netconf_vlan_test、netconfsim query/asserts、golden）
- [ ] 1.4 测试面适配（D2）：漂移字段断言与 netconfsim 数据树节点一并删除；hwfix 夹具去 `NetworkLayerStatus` 等；golden 重生成后人工 diff 抽查 vlan/ifm 仅预期字段消失
- [ ] 1.5 taskname.gen.go diff 检查：category 键集如有变化，同步 api/前端测试断言
- [ ] 1.6 门禁：`go test ./... -race` 全绿 + `make gen-yang` 零漂移 + 前端 `npm test` 全绿（如 category 变化波及）
- [ ] 1.7 评审（`go-code-review-check`）→ What/Why/How 提交 → push + PR-1 → CI 全绿合入

## PR-2：删除 yang-models submodule 及 14 触点

- [ ] 2.1 `git submodule deinit yang-models && git rm yang-models && rm .gitmodules`（或仅删该条目）
- [ ] 2.2 `scripts/gen-yang.sh`：目录缺失提示语改「入库目录应随仓库存在」（CG-01 新 Scenario，先补 shell 层断言/手测负路径）
- [ ] 2.3 `compliance.yml`：path 触发器 `yang-models`→`snd/ce6866p-yang/**`、变更检测 grep 同步、删 submodule init 步骤
- [ ] 2.4 `.githooks/pre-commit`：提示语去 submodule；触发路径加 `snd/ce6866p-yang`
- [ ] 2.5 文档清理：`generated/huawei/doc.go`、`generated/README.md`、`docs/DEPLOY-WSL-CN.md`、CLAUDE.md/TEAM_HANDBOOK 如有提及
- [ ] 2.6 全量验证：干净 checkout 模拟（`git stash -u` 后 `make gen-yang`）+ `go test ./... -race` 全绿
- [ ] 2.7 覆盖率棘轮校验（T08，留 0.1 CI 抖动余量）→ 评审 → 提交 → PR-2 → CI 全绿合入

## 收官

- [ ] 3.1 `/opsx:sync`（CG-01/CG-03/BR-11 delta 合主 spec + 新建 snd-package 主 spec）→ `/opsx:archive`
- [ ] 3.2 更新记忆 [[snd-integration-program]]：①期完成状态、②期入口
