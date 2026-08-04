# Tasks: crd-to-yang-codegen

> TDD（T01/T05）：每组先写红灯测试再实现。测试分层（§5.6）：工具核心=B1 表格驱动；往返对账=B2 风格集成（Short 跳过）；前端=派生黄金（GD-01）。单 PR ≤1000 行（TM04），生成物与手写代码分 commit。

## 1. 示例 CRD 与工具骨架（B1 红灯先行）

- [x] 1.1 编写示例 CRD `testdata/businessvlannets.crd.yaml`（Kind=BusinessVlanNet，约束逐条对齐 huawei-vlan/huawei-ifm：vlan-id 1..4094、name 1..31、description 1..80、devices list map-keys=[ip]、mode enum、leaf-list、嵌套 container、mandatory；带 D3 全套注解）
- [x] 1.2 `tools/crd2yang/` 骨架：`Options`/`Generate` 签名 + main.go flag 解析；先写 B1 表格驱动测试（C2Y-01/02/03 全分支 + 负路径：缺 revision、多 served 版本、无 key 对象数组、oneOf fail-fast）——红灯
- [x] 1.3 实现 manifest 解析 + 注解/缺省派生（C2Y-02）+ 结构映射（C2Y-03：container/list/leaf-list/integer 宽度推导/pattern/enum/mandatory 剥离 key）+ YANG 文本渲染（含 task-name 扩展、generated-from 溯源、description）——绿灯
- [x] 1.4 golden 测试：`testdata/businessvlannets.golden.yang` 入库，goyang 解析零错误断言（C2Y-01）

## 2. 往返对账（C2Y-04）

- [x] 2.1 先写红灯：集成测试 CRD→crd2yang→YANG→（os/exec `go run ../crdgen`）→CRD′，归一化比对器（剥 x-kubernetes-*、key 并回 required）+ 深度相等断言，`testing.Short()` 跳过
- [x] 2.2 修通往返零失真（宽度推导/required 剥离/pattern 序列化等失真点在此收敛）——绿灯

## 3. 管线接入（C2Y-05）

- [x] 3.1 生成 `backend/internal/yang/models/usmp-business-vlan-net.yang`（工具产出，单独 commit）；新建 `internal/generated/businessdemo/gen.conf` 并 `make gen-yang VENDOR=businessdemo` 出 ygot 包（生成物 commit）
- [x] 3.2 先写红灯：schema 树注册测试（`AddYgotSchemaWithVendor` businessdemo，断言 container/list/leaf 层级与 range/pattern/mandatory 元数据在位）——随后绿灯
- [x] 3.3 `tools/schemadump/main.go` 追加加载 businessdemo 包 → `make gen-schema-fixtures` 出 `business-vlan-net.json`（模块名=顶层容器名，AddYgotSchemaWithVendor 命名口径）；fixture 语义变化注记（README/头注释）
- [x] 3.4 前端派生黄金：跑黄金生成，`usmp-business-vlan-net` 黄金入库，deriveTabs/deriveColumns/deriveKeyField 全绿（GD-01）；受影响模块人工核对（SF-04）

## 4. Makefile 与门禁（C2Y-06）

- [x] 4.1 Makefile 增 `gen-crd2yang` target（+.PHONY）；本地验证幂等（连跑两次零 diff）
- [x] 4.2 `compliance.yml` regen-and-diff 块追加 `make gen-crd2yang`（置于 gen-yang 之前）+ CHANGED grep 白名单；`.githooks/pre-commit` 同口径（CG-03）；手改 `.yang` 漂移被拦验证

## 5. 收尾

- [x] 5.1 覆盖率棘轮：后端新增包补测后核对 `.coverage-baseline` 不降、按需上调（T08）
- [x] 5.2 `go-code-review-check` 通过；What/Why/How 三段式 commit 整理；记忆更新（`docs: 记忆更新` 单独 commit，MEM04）
- [x] 5.3 完成分支流程（§6.3 选项 B：push + PR #268），CI 全绿后自助 merge（合入授权口径）——2026-08-04 收官
