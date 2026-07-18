# Tasks — retire-openconfig-models

> 顺序即依赖：spec 已先行（R17），先测试改造（红/绿），再物理删除，最后 regen + 棘轮校验。全程 worktree 隔离（§6）。

## 1. 测试改造：解除对 generated/openconfig 的测试载体依赖（D2）

- [x] 1.1 `pkg/yang-runtime/client/netconf_registry_dispatch_test.go`：「注册表未命中降级」用例改用测试本地未注册 GoStruct 伪类型（实现 `ygot.GoStruct` 的哑结构体），断言走 `xml.Marshal` 兜底链、不报未注册硬错误（对齐 delta XC-04 负路径 Scenario）
- [x] 1.2 `pkg/yang-runtime/client/netconf_delete_marshal_test.go`：同法替换 openconfig 类型载体
- [x] 1.3 `pkg/yang-runtime/driver/registry_xml_test.go` 等其余引用 openconfig 的测试逐一改造（先 `grep -rl "generated/openconfig" backend --include="*_test.go"` 全量盘点）
- [x] 1.4 B1 门禁：`go test ./pkg/yang-runtime/... -race` 全绿

## 2. 运行期 schema 收敛（yang-api BR-11）

- [x] 2.1 先写红灯：`internal/yangschema` 或 `internal/api` 测试断言 `Load()` 产出模块 vendor 仅 `huawei`/`usmp`、不含 `interfaces`/`vlans`（openconfig 根容器）
- [x] 2.2 `internal/yangschema/load.go`：删 openconfig schema 加载与 import，更新包注释
- [x] 2.3 B3 门禁：`go test ./internal/api/... ./internal/yangschema/... -race` 全绿

## 3. 物理删除 openconfig 载体

- [x] 3.1 删 `client/netconf.go` openconfig 特判分支：`OpenconfigVlan_Vlans` 块、`OpenconfigInterfaces_Interfaces` 块、`buildOpenConfigInterfacesXML`、`OpenConfigInterfacesNS`；逐符号查引用后清理仅剩死引用的辅助符号（`IanaIfTypeNS`、`xmlEscape` 若无他用）与 map 兜底 replacer 中的 `OpenconfigVlan_Vlans_Vlan` 死 token；删 `generated/openconfig` import
- [x] 3.2 删 `backend/internal/controller/interfaces/` 整目录（含测试；main.go 本就未注册，零接线改动）
- [x] 3.3 删 `backend/internal/generated/openconfig/` 整目录（all.gen.go + gen.conf）
- [x] 3.4 删 `backend/internal/yang/models/{openconfig-vlan,openconfig-interfaces,openconfig-types,openconfig-extensions}.yang`（保留 `usmp-business-vlan.yang`）
- [x] 3.5 删 `spec/openconfig-vlan-controller.md`；清 `controller/vlan/reconciler.go` 等处残留 openconfig 注释（`grep -rn openconfig backend --include="*.go"` 收敛到仅 `github.com/openconfig/*` 库 import）
- [x] 3.6 `go build ./... && go test ./... -race` 全绿（backend 全量）

## 4. 生成管线与前端收尾

- [x] 4.1 `make gen-yang` 全量 regen-and-diff 零漂移（huawei/business 两包不受影响，openconfig 包消失后脚本自然跳过）
- [x] 4.2 前端 `test/stores/menu.native.test.ts`、`test/components/Sidebar.test.ts`：合成数据 `vendor:'openconfig'` 改中性标签（D4），断言不变；`npm test` 全绿
- [x] 4.3 覆盖率棘轮校验（T08）：重算后端覆盖率，不低于 `backend/.coverage-baseline`；升了则同步上调基线

## 5. 收官

- [x] 5.1 `go-code-review-check` 通过（T04）
- [x] 5.2 What/Why/How 三段式提交（净删除预估 ≤1000 行，TM04）
- [x] 5.3 推送 + PR，CI 全绿后合入；随后 `/opsx:sync` 合 delta spec、`/opsx:archive` 归档
