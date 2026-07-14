## Context

`pkg/yang-runtime/xmlcodec` 是 Stack B 所有华为模块的通用编解码引擎（vlan/ifm/bgp/ni/tunnel-management/xpl/routing-policy 共用）。其 `encodeField`/`encodeLeaf` 对 ygot 枚举（derived int64，实现 `ygot.GoEnum`）此前直接 `fmt.Fprintf("%d", fv.Int())` 输出整数；`decodeField` 用 `strconv.ParseInt` 只收整数。YANG enumeration 的线上表示是**值域名**（RFC7950/RFC7951），真机据此校验。

此缺陷由 BGP 2b 波次④（acl）触发：acl `groups/group` 的 `type` 是 mandatory 枚举，`yang-config-test-design` 完备矩阵断言"编码 XML 含 `<type>basic</type>`"直接失败（引擎发 `<type>3</type>`）。追溯发现 vlan/ifm 亦潜伏此 bug——golden 固化整数、sim 宽容、2a af-type 未断言线上名。

## Goals / Non-Goals

**Goals:**
- 枚举 leaf/key 编码为 YANG 值域名（真机可接受），解码按名还原。
- 修复已交付 vlan/ifm 的真机枚举兼容 bug。
- 解耦子系统零改动：既有以 ygot 整数值断言的集成测试经 sim 侧名→int 解析无需改动。
- 向后兼容：解码兼容旧整数；未映射值编码退回整数不 panic。

**Non-Goals:**
- 改变非枚举 leaf 编解码（XC-01~07 不动）。
- 改 `generated/` 或 `gen.conf`。
- acl 配置面本身（波次④，本 change 只交付其依赖的枚举修复）。

## Decisions

**D1：编码用 `ygot.EnumName`（ΛMap），未映射退回整数。**
- `ygot.EnumName(GoEnum)` 由 `ΛMap[typeName][int]` 取值域名，是 ygot 官方 int→名映射（与 EmitJSON 一致）。
- 未映射值（`ΛMap` 无此 int——仅合成/异常，真机有效值必在 map）退回整数、不报错：upholds R08，且保持 golden 中合成 fixture 值（vlan type=1、access-type=1）的往返稳定。
- leaf 枚举（`encodeField`）与 key 枚举（`encodeKeysFirst`→`encodeLeaf`）统一走 `encodeLeaf` 的枚举分支，避免两处逻辑分叉。

**D2：解码用 ΛMap 反查名→int，回退整数。**
- 先按值域名反查（真机/本引擎 encode 均发名）；未命中回退 `ParseInt`（历史整数报文/数据兼容）；再不成则报错命名该 leaf（R08）。

**D3：模拟网元回读侧名→int（保集成测试不变），而非改断言。**
- `netconfsim/query.go` 新增 `enumInt(text, sampleEnum)` 经 `ΛMap` 反查，替换枚举 leaf 的 `toInt`。sim 因此需 import `huawei`/`ygot`（测试基建，非生产；无 import 环）。
- 备选（弃）：改 testdata 结构体为字符串 + 改 asserts + 改 17 集成测试值域名——churn 大 3×，且 ifm `type` 枚举 80+ 值不宜硬编码表。选 ΛMap 通用映射，零硬编码、零集成测试改动。

**D4：legacy shim 线上断言改名。**
- `TestBuild*` 经同名 shim 走通用引擎、断言线上 XML；引擎改后线上即值域名，断言相应更新（合成/未映射值如 vlan type=1 仍整数）。

## Risks / Trade-offs

- **[R1] 未映射枚举值静默发整数掩盖真实错误** → Mitigation：真机有效枚举值必在 `ΛMap`，未映射仅合成测试值；生产路径必得名。退回整数 upholds R08（不中断），且解码端能回读。
- **[R2] 解码歧义（名恰为数字串）** → Mitigation：YANG 枚举值域名非纯数字；先名后整数的顺序无歧义。
- **[R3] sim 耦合 huawei/ygot** → Mitigation：仅测试基建 query.go，无 import 环，无生产影响；换取零硬编码 + 集成测试零改动。
- **[R4] golden 更新掩盖非预期改动** → Mitigation：逐行 review golden diff 确认仅枚举 int→名（已核）；新增 `TestEncode_EnumEmitsYANGName` 定向锚点。

## Migration Plan

1. worktree 隔离（✅ `xmlcodec-enum-name` off main）。
2. 修 encode/decode → 重生 golden → 修 sim query.go → 修 shim 断言 → 加回归锚点。
3. `go test ./... -race` 全绿（34 包）→ review → 提交（≤500 行）。
4. sync XC-08 → 主 spec；archive；PR 合入。
5. 合入后波次④ acl rebase 于新 main（枚举编码即正确，acl 完备矩阵转绿）。

## Open Questions

- 无。既有子系统全绿、真机语义明确（值域名）、向后兼容（解码收整数）。
