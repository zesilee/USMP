# snd-xml-codec — 设计

## Context

P5-1（#134）交付了驱动描述符注册表：manager 路由与 `/config` 编解码已查 `pkg/yang-runtime/driver` 注册表，但描述符的 `DecodeXML` 闭包仍指向 `client.ParseHuaweiXxxXML` 手写解析器，而下发方向 `client.marshalChange`/`marshalDeleteChange` 根本不经注册表——每模型手写 build（字段逐行 Sprintf）+ Parse（`xml:` tag 中间结构）+ type-switch 分发 + camelCase→kebab 替换器，共约 800 行/两模块。存量债两处：`buildHuaweiVlanVlansXML` 重复发送 `<suppression>`（netconf.go:1387、:1436）；IFM build 35 字段 vs parse 10 字段不对称 → 超出 10 字段的配置永久漂移。

已核实的数据基础（全部在 ygot 生成物内，无需新 codegen）：
- `path:"vlan-id"` / `module:"huawei-vlan"` struct tag —— XML 元素名；
- `SchemaTree map[string]*yang.Entry`（`types.go:42`，内嵌 gzip schema）——`Entry.Namespace().Name` 给出 XML namespace；
- `ΛListKeyMap()` —— list 条目主键（键式删除编码）；
- enum 类型底层为 int64，零值 = UNSET（既有 builder「enum != 0 才发」语义与之吻合）。

## Goals / Non-Goals

**Goals:**
- 一个通用引擎处理任意 ygot GoStruct 的 NETCONF XML encode / decode / 键式 delete；加新模块 = 注册描述符（含编解码数据字段），零 XML 代码。
- `pkg/yang-runtime/client` 不再 import `internal/generated/huawei`（框架层与厂商生成物解耦）。
- encode 与既有输出严格等价（golden 对拍；唯一例外 suppression 重复修复）；decode 扩展为与 encode 字段对称。

**Non-Goals:**
- 不动 `buildOpenConfigInterfacesXML`（Stack A 邻接 `controller/interfaces` 在用，保留原样与 openconfig 类型分支）。
- 不收敛 `mergeConfig`/`config_delete` 的 API 层 type-switch（独立 change）。
- 不做 ygot 生成管线参数化（P5-3）、不做非 Huawei 驱动（P5-4）。
- 不改 diff 引擎、不改 Reconciler 语义、不改 NETCONF 会话层（opMu/自愈已在 #131 定型）。

## Decisions

### D1 通用引擎放 `pkg/yang-runtime/xmlcodec`，描述符只带数据
备选：把手写 builder 搬进描述符闭包（每厂商包仍手写 XML 代码，仅换位置）。否决：那只是代码搬家，模块边际成本不降。选定：`driver.Descriptor` 新增纯数据字段——

```go
// driver.Descriptor 新增（P5-1 注释已预留"在此 struct 上扩展"）：
XML *XMLCodecSpec // nil = 该模块不走通用引擎（如 system 无回读）
type XMLCodecSpec struct {
    Namespace  string                  // yang.Entry.Namespace().Name，接线时从 SchemaTree 取
    RootElem   string                  // 外层容器元素名，如 "vlans"
    RootSchema func() *yang.Entry      // SchemaTree 入口（懒取，避免 init 顺序耦合）
}
```
引擎函数签名 `xmlcodec.Encode(spec, GoStruct)` / `Decode(spec, raw, GoStruct)` / `EncodeDelete(spec, GoStruct)`。依赖方向：`xmlcodec` → ygot/goyang（零业务依赖）；`client` → `driver` + `xmlcodec`；`internal/drivers` → 全部。无环。

### D2 Encode 用 reflect + struct tag，不用 schema 树驱动遍历
元素名与层级已完整编码在生成 struct 的 `path:` tag 里，reflect 遍历 struct 即可；SchemaTree 仅用于 namespace 与（decode 侧）list key 识别。备选「按 yang.Entry 树遍历、从 struct 反查字段」需要维护 entry↔field 双向映射，复杂度高一档且收益为零。跳发语义精确复刻既有 builder：nil 指针跳过、enum==0 跳过、嵌套 container nil 跳过、list map 为空发自闭合根元素。

### D3 golden 先行，「严格等价」按规范化形式比对
先写 golden 测试冻结**现有** builder 对代表性 fixture（VLAN 全字段+member-ports、IFM 全字段+嵌套 damp/error-down/control-flap、单字段最小体、空容器、特殊字符）的输出。规范化（`xmlcodec.Canonicalize`）：元素/属性名解析为 namespace URI（前缀不敏感）、属性排序、**同级元素全排序**（YANG leaf/list 同级序在语义上不重要；builder 迭代 Go map 条目序非确定，不排序则 golden 天然 flaky）、**排序后相邻完全相同的同级元素去重**（NETCONF merge 幂等语义下重复同内容元素为 no-op——顺带把 suppression 重复 bug 规范化掉，golden 无需豁免机制）。因全排序抹掉了元素序，「key 叶为首元素」（DP-07）与根元素 namespace 由**定向单测在原始输出上单独断言**。golden 文件存规范化形式。通用引擎必须逐 fixture 对拍通过后才允许切换调用点。

### D3b 引擎不做 config-false 过滤；namespace 为显式描述符数据（实测校准）
原设想「Encode 按 `yang.Entry.ReadOnly()` 跳过 config-false」被 schema 实测推翻：华为模型把 member-ports **整棵子树**（含 interface-name/tag-mode/access-type）、is-l2-switch、error-down、encapsulation-type 等**既有 builder 照发的字段**都标了 config-false（运行态镜像建模惯例），按 schema 过滤会直接破坏严格等价（golden 立红）。故引擎**不过滤**：populated 即发（nil/enum 零值跳过不变）。风险面小：desired 只含 UI 提交字段，CommonStatistics 等纯运行态容器天然不会被填充。Decode 侧全字段填充（呈现叶 class/parent-name 等继续透出，既有 parser 已如此）；不变量为 decode ⊇ encode（可下发字段必可回读）。
另一实测：内嵌 gzip schema 的 `Entry.Namespace()` 返回空——**namespace 无法从 SchemaTree 派生**，作为描述符显式数据字段登记（`HuaweiVlanNS`/`HuaweiIfmNS` 常量迁至 drivers 注册处成为数据）。

### D3c enum 编码保持内部数值（%d），不改字符串名
既有 builder 以 ygot enum 内部 int 值 `%d` 上写，回读亦按数值解析，真机/模拟器链路已按此交付（[[vlan-config-stackb]]）。enum→YANG 字符串名（ΛMap）是潜在正确性改进但属行为变更，本期严格等价不动，留待有真机反证时独立 change。

### D4 Decode 字段全覆盖是刻意的行为改善，不是等价破坏
build/parse 不对称是已定位的漂移债（同 [[reconcile-convergence-3rootcauses]]「回读解析恒空」谱系的字段级残留）。decode 侧以 encode 同一 reflect 路径反向填充，天然对称；新增回读字段只会让 diff 更准确（消除假漂移），由存量 B2 收敛断言 + 新增「全字段配置→回读→diff 为空」用例验证。

### D5 键式删除经 ΛListKeyMap，key 元素序对齐 RFC 惯例
`EncodeDelete` 对 list map 每条目取 `ΛListKeyMap()` 输出 key 叶（现两模块均单 key：vlan→id、ifm→name），key 为首个（唯一）子元素，`nc:operation="delete"` 属性带 NETCONF base namespace——DP-07 全部既有断言逐条保持。未知/无 XML spec 的模型返回明确错误（R08），不发裸 delete。

### D6 marshalChange 分发顺序：先注册表，后既有 fallback
`client.marshalChange` 改为：string/[]byte 直通 → 查 `driver.EncoderXMLFor(value 类型)`（新查找：按 GoStruct 具体类型匹配描述符，非路径匹配——Change.NewValue 在 diff 引擎产出时路径信息不可靠，这正是当年 IFM map 型分支 bug 的根因）→ openconfig 遗留类型分支原样保留 → 最后 xml.Marshal 兜底原样保留。类型匹配用 `reflect.TypeOf` 对描述符 `NewStruct()` 的类型，含「裸内层 map」形态（diff 引擎会发 `map[uint16]*Vlan` 这种内层 map，现有两个专用 type assertion 分支即为此而生——描述符需同时登记容器类型与内层 map 类型）。

## Risks / Trade-offs

- [编码语义偏差 = 全网漂移/下发失败（对账主链路）] → golden 逐 fixture 对拍 + 存量 B2（下发→回读→收敛）全绿硬门禁 + netconfsim RFC edit-config 已接线（P4）能捕获非法 XML。
- [reflect 引擎遗漏 ygot 类型形态（Binary/YANGEmpty/leaf-list slice/union）] → 引擎按现存两模块实际出现的形态实现并对未支持形态返回明确错误（R08 不静默丢字段）；fixture 覆盖全部现役字段类型。
- [decode 全覆盖引入新回读字段，若 netconfsim 种子数据缺字段可能翻出隐藏断言] → 先跑存量 B2 基线，失败即为真回归信号（这正是 golden-first 的意义）。
- [suppression 修复属字节级行为变更] → 真机/模拟器均为「后值覆盖」宽容语义，B2 收敛断言兜底；在 delta spec 中显式记录。
- [PR 体积] → 引擎 ~+500、golden fixture ~+300、退役 ~-800，净额可控；若超 1000 行，decode 对称扩展拆为同 change 第二 commit（原子性仍单 PR）。

## Migration Plan

存量并行→切换→删除（§5.3）：1) golden 冻结现状（红）；2) xmlcodec 引擎 + 单测对拍 golden（绿）；3) 描述符扩展 + huawei 接线，`marshalChange`/`marshalDeleteChange`/reconciler Parse 调用点切换到注册表；4) 双路径验证（B2 全绿）后删除手写三件套与 client 的 huawei import。回滚 = revert PR（无数据迁移、无接口变更，前端/API 面零感知）。

## Open Questions

（无——插拔性已在 P5-1 拍板纯 Go 编译期；注解载体已在 P3' 拍板 YANG extension；本期无待用户决策分叉。）
