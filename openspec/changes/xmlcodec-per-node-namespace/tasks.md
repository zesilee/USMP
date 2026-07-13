## 1. 基线与红灯

- [x] 1.1 worktree 内基线 `go test ./pkg/yang-runtime/xmlcodec/... ./internal/drivers/...` 全绿，记录 golden 现状
- [x] 1.2 红灯先行（TDD）：写「ni 根下 bgp 子树 `<bgp>` 须带 huawei-bgp namespace」encode 真值断言 + 「单模块树零新增 xmlns（golden 逐字节不变）」断言——先失败

## 2. 引擎实现（XC-06）

- [x] 2.1 `xmlcodec.Spec` 新增 `Namespaces map[string]string`（module→URI，可选）
- [x] 2.2 加 `moduleTag(f)` helper（与 pathTag 并置）
- [x] 2.3 编码器下降跟踪 parentNS（根=Spec.Namespace）：字段 module 解析 namespace，≠parentNS 且非空则发 xmlns 并作为子树 parentNS；一致适用于嵌套子容器与 list 条目（复用字段级机制，不写平行逻辑）
- [x] 2.4 单模块树零回归：Namespaces 缺省或值同根 → 不发新 xmlns

## 3. 驱动登记 + 断言

- [x] 3.1 `internal/drivers/huawei.go`：ni 描述符 Spec 填 `Namespaces{huawei-network-instance, huawei-bgp}`
- [x] 3.2 encode 真值断言用例：ni 根下 bgp 子树带 `xmlns="urn:huawei:yang:huawei-bgp"`、`<name>` 不另发（XC-06 场景1）
- [x] 3.3 混合 namespace 往返用例：解码鲁棒（XC-06 场景4）

## 4. 零回归门禁

- [x] 4.1 VLAN/IFM/BGP/system/ni golden 逐字节一致、往返恒等保持（XC-01/02/05 全绿，硬门禁）
- [x] 4.2 `go test -race ./pkg/yang-runtime/xmlcodec/... ./internal/drivers/... ./internal/controller/...` 全绿
- [x] 4.3 覆盖率不下降（T08），达标上调基线

## 5. 评审与提交

- [ ] 5.1 `go-code-review-check` 通过
- [ ] 5.2 `git-what-why-how-commit` 三段式提交，≤500 行/commit
- [ ] 5.3 push + PR（CI required checks 全绿，PR ≤1000 行）

## 6. sync + archive（合入后）

- [ ] 6.1 `/opsx:sync`：XC-06 合入主 spec `openspec/specs/yang-xml-codec`
- [ ] 6.2 `/opsx:archive`
- [ ] 6.3 更新记忆：per-node namespace 已交付，解锁 2a peers
