## 1. 枚举编解码修复（XC-08）

- [x] 1.1 `encode.go`：`encodeLeaf` 枚举分支经 `ygot.EnumName` 输出值域名，UNSET 跳发，未映射退回整数（R08）
- [x] 1.2 `encode.go`：`encodeField` 枚举委托 `encodeLeaf`（leaf 与 key 统一路径）
- [x] 1.3 `decode.go`：`decodeField` 经 `ΛMap` 反查名→int，回退整数，未知报错命名 leaf
- [x] 1.4 回归锚点：`encode_test.go` `TestEncode_EnumEmitsYANGName`；`decode_test.go` "enum decodes by YANG name" + "unknown enum value errors"（T07）

## 2. golden 与子系统对齐

- [x] 2.1 重生 `hwfix/golden/{ifm_full,vlan_full}.canon.txt`（`-update-golden`），逐行 review 仅枚举 int→名
- [x] 2.2 `netconfsim/query.go`：`enumInt(text, sampleEnum)` 经 `ΛMap` 反查，替换枚举 leaf `toInt`（vlan+ifm 全部枚举字段）
- [x] 2.3 `client/netconf_{vlan,ifm}_test.go` `TestBuild*` 线上断言整数→值域名（未映射合成值仍整数）

## 3. 门禁

- [x] 3.1 `go test ./... -race` 全绿（34 包）；覆盖率不低于基线
- [x] 3.2 `go-code-review-check` 通过（T04）

## 4. 提交与合入

- [ ] 4.1 What/Why/How 三段式提交（≤500 行/commit，超限拆 codec / sim+tests）
- [ ] 4.2 `/opsx:sync`：XC-08 delta → 主 spec
- [ ] 4.3 `/opsx:archive`：归档 change
- [ ] 4.4 push + PR，CI 全绿后合入；合入后波次④ acl rebase
