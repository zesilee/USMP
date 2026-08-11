# 阶段3.3+4 大切换计划（S1–S4，2026-08-11 定稿）

> JSON 与 XML 通道的切换耦合在同一 `driver.Descriptor` 上，按此四步走保证每步可编译、可测试、可提交。上下文压缩后从此文档恢复。

## 现状事实（已实证）

- JSON 通道对拍全绿（3.1/3.2 ✓）：native 生成包（`internal/generated/native/{huawei,business}`）带 MarshalJSON/UnmarshalJSON；三条 ygot 行为已冻结（键无模块前缀/枚举键 UNSET 拒/空容器 {} 不发）。
- xmlcodec 的 Entry 用法闭集：根名 `e.Name`、子查找 `e.Dir[tag]`（encodeFields 350 行/leafdelete 49-50 行/resolve 82 行）、祖先链 `Parent`（wrappers 97 行，IR 树无 fakeroot：`for p:=n.Parent(); p!=nil` 语义等价）、Key 语句（keyNames 132/encodeKeysFirst 303）。
- Spec 构建点：`internal/drivers/huawei.go`（~6 处 `huawei.SchemaTree[...]`）、`huawei_modules.go`（registerPlain 表循环 1 处）、xmlcodec 约 10 个 `*_test.go`。
- decode/delete/canon 不直接用 Entry（经 resolve 的 r.list/r.schema）。
- intent/cr.go 的 `root.Validate()`（business 包）是 ygot 校验唯一消费点——S3 前须先落 IR 校验器（任务5.1/5.2）。

## S1 xmlcodec schema 源 IR 化（可独立提交）

- `Spec.Schema func() *yang.Entry` → `func() schema.Node`（xmlcodec import pkg/yang-runtime/schema，无环）。
- resolved.schema/.list 换 Node；助手：nodeChild（Container/List 断言后 Child）、nodeKeys（ListNode.Keys() 名序）、wrappers 用 Parent() 链（无 fakeroot 排除项）。
- drivers 构建点换 `yangschema.Load()` + `s.Path("/vlan/vlans")` 等（internal 可 import internal）；registerPlain 表行同（path=`/<root>`）。
- xmlcodec 测试统一加 helper `irSpecNode(t, path)`（内部 yangschema.Load），sed 替换各测试 Schema 闭包。
- 验收：xmlcodec 全套件 + B2 + golden 逐字节不变。

## S2 值接口双族兼容 + XML 对拍（可独立提交）

- 引擎内部 helper 双族断言：listKeyMap(v)（object.KeyedObject 优先，ygot.KeyHelperGoStruct 兜底）、enumName(v)（object.Enum / ygot.GoEnum）、marker 判定。
- 入口签名放宽 interface{}（Encode/Decode/EncodeDelete/EncodeLeafDelete/WrapListMap/ListMapType），nil/非法类型显式报错（R08）。
- 新增 XML 对拍测试：同值 ygot 结构体 vs native 结构体过同一引擎 → Encode/EncodeDelete 输出逐字节相等、Decode 结果语义相等（就是任务4.2 的对拍面）。
- 验收：既有 golden 不动、双族对拍绿。

## S3 大切换（多 commit，一起过 CI）

1. 任务5.1/5.2 先行：intent 现状 Validate 行为快照测试 → IR 校验器（pattern/range/length/enum，从 yangschema.Load 树取叶约束）→ intent/cr.go 切换。
2. native 包补导出：registry.go 增 `func Unmarshal(data []byte, dest object.Object) error`（json.Unmarshaler 断言分派；对齐 driver 现签名少 opts）。
3. `driver.Descriptor`：DecodeXML/NewStruct/Unmarshal/WrapXMLValue 换 object.Object；UnmarshalOpt 参数删除。
4. `internal/drivers/*`：NewStruct/Unmarshal/DecodeXML 全换 native 包；controllers（vlan/ifm/bgp/system/networkinstance/plainmodule）`&huawei.Device{}` 等换 native；intent/expand/cleanup、api/config_delete 的 huawei+ygot.String 换 native+object.String；api/config_codec+changeset_handler 的 `ygot.EmitJSON` 换 `parsed.(json.Marshaler).MarshalJSON()`；netconfsim query.go GoEnum→object.Enum。
5. 验收：全量 B1/B2/B3 -race + e2e smoke；对外 API 形状零变化。
- 涉及测试文件同步换 import（sed：internal/generated/huawei → internal/generated/native/huawei，类型名不变！）。

## S4 清理（阶段6 前奏）

- xmlcodec 删 ygot 分支与 import；schemagen 切直读 YANG 源（bridge 直建 IR，不再经 generated Schema()）；旧 generated/{huawei,business} 删除留待阶段6（对拍测试尚需 ygot 参照——对拍脚手架随旧包一起删，golden 留基线）。

## 风险备忘

- S3 第 4 步 sed 换 import 时**类型名两侧完全相同**（结构对拍保证），只有包路径变；`_ = ygot` 残留 import 用 goimports 清。
- client 包 fallback（XC-04 未注册类型走 xml.Marshal）不受影响（interface{} 面）。
- businessdemo/schemadump 留在 ygot 面（豁免），S4 之后随 6.2 拍板。
