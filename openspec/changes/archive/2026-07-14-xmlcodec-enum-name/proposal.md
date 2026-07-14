## Why

通用 XML 编解码引擎（`pkg/yang-runtime/xmlcodec`）此前把 YANG `enumeration` leaf 编解码为**整数值**（`<admin-status>2</admin-status>`），而非 YANG **值域名**（`<admin-status>up</admin-status>`）。真实华为 NETCONF 设备按值域名校验，整数会被拒——即已交付的 **VLAN/IFM 配置面在真机上对枚举字段静默失效**（潜伏 bug）。

该缺陷长期未暴露：golden 用例固化了整数形态、模拟网元对格式宽容、BGP 2a 的 `af-type` 枚举 key 也只经宽容 sim 往返、从未断言线上值域名。BGP 2b 波次④（acl）的 `groups/group/type`（**mandatory** 枚举）在 `yang-config-test-design` 完备矩阵中首次断言"编码须为值域名"，一击命中此缺口——正是 T02b 矩阵"断言 encode 真值而非仅 sim 往返"的设计意图。

按用户决策（2026-07-14）：此为**平台级编解码正确性修复**，影响已合入的 vlan/ifm，拆为独立 change 先行合入，再供 acl（波次④）等后续 change 复用。

## What Changes

- **枚举 leaf 编码为 YANG 值域名（XC-08）**：`encodeLeaf`/`encodeField` 经 `ygot.EnumName`（由 ygot `ΛMap` 映射 int→名）输出值域名；`UNSET(0)` 跳发；**映射不到的值**（仅合成/异常值，真机有效枚举必在 `ΛMap`）退回整数、不报错（R08：不因单个异常枚举中断整树编码）。leaf 枚举与 list key 枚举走同一路径。
- **枚举 leaf 解码按 YANG 值域名**：`decodeField` 经 `ΛMap` 反查 名→int；兼容历史整数形态（回退 `ParseInt`）以不破坏旧报文/旧数据。
- **golden 基线更新**：`internal/testutil/hwfix/golden/{ifm_full,vlan_full}.canon.txt` 由整数枚举重生为值域名（`admin-status 1→down/2→up`、`link-protocol 1→ethernet`、`router-type 1→PtoP`、`type 1→Ethernet`、`class 1→main-interface` 等）。**捕获的是被修复的旧 bug 形态，非行为回退**——未映射的合成 fixture 值（如 `vlan type=1`、`access-type=1`）仍为整数（退回路径）。
- **模拟网元回读解析枚举名**：`simulator/netconfsim/query.go` 新增 `enumInt(text, sampleEnum)` 经 `ΛMap` 反查名→int，替换枚举 leaf 的 `toInt`；使既有以 ygot 整数值断言的集成测试**无需改动**继续通过。
- **legacy shim 线上断言更新**：`pkg/yang-runtime/client/netconf_{vlan,ifm}_test.go` 的 `TestBuild*`（经同名 shim 走通用引擎）线上 XML 断言由整数改为值域名（等价性回归锚点保持）。
- **回归锚点**：`encode_test.go` 新增 `TestEncode_EnumEmitsYANGName`、`decode_test.go` 新增"enum decodes by YANG name"+"unknown enum value errors"（T07）。

## Capabilities

### Modified Capabilities
- `yang-xml-codec`: **ADD XC-08**——YANG enumeration leaf/key 编解码按值域名（`ygot.EnumName`/`ΛMap`），替换旧的整数形态；未映射值退回整数（R08）；解码兼容旧整数。XC-01~07 行为不变（容器根/per-node ns/empty 等）。

## Impact

- **代码**：`pkg/yang-runtime/xmlcodec/{encode,decode}.go`（枚举名编解码）、`simulator/netconfsim/query.go`（回读枚举名解析，新增 `huawei`/`ygot` import）、goldens 重生、`pkg/yang-runtime/client/netconf_{vlan,ifm}_test.go` + `xmlcodec/{encode,decode}_test.go` 用例更新。**不动 `generated/`、不动 `gen.conf`**。
- **真机正确性修复**：VLAN/IFM 下发的枚举字段此前发整数（真机拒），本 change 后发值域名——**修复已交付模块的真机兼容 bug**（下游波次④ acl 依赖此修复的枚举编码）。
- **兼容性**：解码兼容旧整数形态（回退 ParseInt）；未映射枚举值编码退回整数、不 panic（R08）。
- **前端**：无影响（YANG 自动渲染）。
- **合规**：R04、R08（未映射不中断）、T07（回归锚点）、≤500 行/commit、worktree 隔离。
