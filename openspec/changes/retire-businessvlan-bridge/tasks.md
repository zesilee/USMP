# retire-businessvlan-bridge — 任务

> 三波 PR 交付（design.md D6），每波独立全绿按序合入；W2/W3 在前波合入 main 后 rebase。
> legacy→新架构迁移路径：并行（已完成，business-network-config）→ 切换（本 change W1 摘除旧桥接注册）→ 删除（W1-W3 物理删）。

## 1. W1 — 桥接摘除 + translator 退役（PR ①）

- [ ] 1.1 （移至 3.0，W1/W2 实测无需豁免）pr-size 门禁纯删除豁免——auto-mode 权限分类器拦截了该 TM04 契约变更的提交，需用户显式批准后落地，仅 W3 actor 删除依赖
- [x] 1.2 【测试先行 T05】driver 注册表 `VendorSupported` 表格驱动测试（B1）：已注册大小写无关命中 / 未注册 false / 空串 false / 与 Register 并发 race——红灯
- [x] 1.3 实现 `driver.Registry.VendorSupported` + 包级 facade（DR-04）——绿灯
- [x] 1.4 `device_handler.go` BR-04 厂商门禁切换到 `driver.VendorSupported`，删除 translator import；既有 B3 设备注册测试（缺省 huawei / 未知厂商 400）不改断言全绿证明行为等价
- [x] 1.5 `crdsource.StartCache` 原样平移为 `intent.StartCache`（含既有测试迁移），main.go 新意图 cache 调用点切换
- [x] 1.6 main.go 摘除 `RegisterIntentSources` 调用与「并行保留」注释，删除 `internal/crdsource` 整包
- [x] 1.7 删除 `pkg/translator` 整包（1.4 后零引用）
- [x] 1.8 全量验证：`go test ./... -race` 全绿、`go vet` 通过、覆盖率不低于基线（T08，纯删除通常抬升覆盖率，若基线可上调则同步上调 `.coverage-baseline`）
- [x] 1.9 W1 提交（What/Why/How、≤500 行/commit 按 1.1-1.7 原子拆分）、push、创建 PR ①（含本 change 全部 openspec 制品），CI 全绿后合入

## 2. W2 — 旧 CRD 类型与 Stack A 载体退役（PR ②）

- [x] 2.1 改为堆叠分支 retire-businessvlan-bridge-w2（W1 未合入期间继续推进；PR ② base=W1 分支，W1 合入后重定向 main）
- [x] 2.2 删除 `backend/api/biz/v1` 整包（四 Kind + groupversion + zz_generated deepcopy）
- [x] 2.3 删除退役 B0 e2e：`backend/test/e2e` 中引用 biz/v1 的测试（build-tag 隔离件），（bases 目录仍存 NDC 清单，apply 行保留仍语义成立；e2e 脚手架目录级退役另立 follow-up）
- [x] 2.4 删除 `backend/config/crd/bases/biz.usmp.io_business{vlans,interfaces,routes,switches}.yaml` 与 `config/samples/biz_v1_*.yaml`（NativeDeviceConfig 清单不动，属 roadmap D1 范围）
- [x] 2.5 删除 `backend/hack/crd-injector`（唯一调用方是 backend/Makefile `manifests` target）并裁剪该 target；裁剪前 grep CI workflows/scripts 确认无其他引用
- [x] 2.6 裁剪 `backend/deploy/scripts/{one-click-deploy,verify-deployment}.sh` 中 businessvlans/businessinterfaces/businessroutes/businessswitches CRD 等待/校验条目
- [x] 2.7 全量验证：`go test ./... -race` 全绿；`grep -r "api/biz/v1\|bizv1" backend` 零命中
- [x] 2.8 W2 提交、push、PR ②，CI 全绿后合入

## 3. W3 — actor 包物理删除 + spec 收尾（PR ③）

- [ ] 3.0 【需用户批准】pr-size（CI）与 commit-msg（本地钩子）两处同口径增设纯删除豁免：insertions ≤ 50 上限 6000（design D5，TM04 契约变更；auto-mode 分类器已拦截自动提交，须由用户确认后落地——actor 4718 行整包删除的唯一可行通道）
- [ ] 3.1 rebase 到 W2 合入后的 main
- [ ] 3.2 删除 `pkg/yang-runtime/actor` 整包（14 文件 4718 行，一次性删保编译，走 1.1 纯删除豁免）
- [ ] 3.3 验证零残留：`grep -r "yang-runtime/actor" backend` 零命中、`go build ./...` + `go test ./... -race` 全绿
- [ ] 3.4 更新 LEGACY spec Purpose 横幅：`actor-transaction`（代码载体已物理删除）、`business-crd`（api/biz/v1 类型已物理删除）、`translation-engine`（能力退役，见 REMOVED delta）
- [ ] 3.5 W3 提交、push、PR ③，CI 全绿后合入

## 4. 收尾

- [ ] 4.1 `/opsx:sync`：delta 合入主 spec（device-driver-registry +DR-04、translation-engine REMOVED、system-architecture SC-01）
- [ ] 4.2 `/opsx:archive` 归档 change；`openspec/tasks/retire-businessvlan-bridge.md` 置 completed 并 `/task archive`
- [ ] 4.3 更新记忆：arch-optimization-roadmap（actor 物理删除已勾销）、k8s-paas-deployment-constraints（旧桥接退役 follow-up 完成）
- [ ] 4.4 清理 worktree（§6.3 完成分支流程，B 选项推送+PR 后视合入情况走 A 清理）
