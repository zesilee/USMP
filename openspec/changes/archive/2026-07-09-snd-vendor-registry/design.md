# design — snd-vendor-registry（SND 声明式化第一步）

## Context

审计实证（详见 proposal Why 与总跟踪任务 P5 节）：vendor 无数据通路、translator 注册表虚设、路径分发三处硬编码。用户拍板纯 Go 编译期（③→①）。本 change 切在「所有分支的共同上游」：注册表 + vendor 贯穿，行为等价，为后续把手写 XML builder 换成声明式描述腾出接入点。

## Goals / Non-Goals

**Goals:**
- Vendor 成为一等数据：注册入口 → device.Store → 消费点全通路。
- translator 真注册表化（init 自注册），调用点按设备解析而非写死常量。
- 路径分发（manager 路由 + config 编解码）收敛到单一驱动描述符注册表。
- 行为严格等价：存量 B2 集成套件不改一行全绿。

**Non-Goals:**
- 不实现任何非 Huawei 驱动（枚举放开但仅 huawei 有注册项；注册未知厂商 400）。
- 不碰 netconf.go XML builder/parser（后续 change，golden test 先行）。
- 不做运行时插件/热加载（用户已拍板否）。
- 不改 Stack A Actor 路径（translator 输入契约仍吃 bizv1 Spec——解耦输入契约属①终态，本期只改「怎么选驱动」不改「驱动吃什么」）。

## Decisions

### D1 Vendor 是 string 枚举贯穿，缺省 huawei，未知注册即拒
`DeviceConnectionInfo.Vendor string`（复用 `translator.VendorType` 语义但连接层用 string 防包环依赖）；注册 API 可选 `vendor` 字段缺省 `"huawei"`（存量调用零破坏）；传未知厂商（无注册驱动）→ 400 明确错误（早失败优于下发时爆炸）。R08：消费点查不到 Vendor（老数据）一律按 huawei 降级。

### D2 translator init() 自注册，GetTranslator 语义不变
`huawei.go` 增 `func init() { RegisterTranslator(VendorHuawei, NewHuaweiTranslator()) }`；`factory.go` 删 `once.Do` 硬注册；`RegisterTranslator` 加锁保并发注册安全（R09）。crdsource 调用点：`GetTranslator(vendorOf(deviceIP))`——`vendorOf` 查 device.Store，miss→huawei（D1 降级）。翻译方法签名与实现零改动。

### D3 驱动描述符注册表：新包 pkg/yang-runtime/driver，编译期 init 注册
```go
type Descriptor struct {
    Vendor       string   // "huawei" / "openconfig"
    ModulePrefix []string // 路径匹配前缀，如 "vlan:", "ifm:"
    Controller   string   // manager 路由目标 controller name
    Codec        Codec    // config_codec 的 encode/decode 闭包对
}
Register(d Descriptor); Lookup(vendor, path) (Descriptor, bool)
```
现有三模块（system/vlan/ifm）各注册一条（vendor=huawei）。`manager.go:325-342` 与 `config_codec.go:28/71` 的 `strings.Contains` 链改为 `driver.Lookup`；查不到时保留现有 fallback 行为（R08，与今天未匹配路径的行为一致）。descriptor 字段刻意最小——①终态的模板/路径描述符将来扩这个 struct，不另起炉灶。

### D4 注册表并发与初始化顺序
注册全部发生在 `init()`（单 goroutine 阶段），运行期只读——map 无锁读安全；仍加 RWMutex 防将来运行期注册误用（R09，race 测试锁定）。包依赖方向：driver 包零依赖（被 manager/api/translator 消费），防环。

### D5 测试策略：行为等价用存量 B2 兜底，新逻辑 B1/B3 全覆盖
新增逻辑（Vendor 贯穿、注册表、查表路由）全部 B1 表格驱动+race；注册 API 契约 B3（带/缺省/非法 vendor）；**不写新 B2**——本 change 行为等价，存量 netconfsim 集成套件（vlan/ifm 下发-回读-收敛、删除链路）原样全绿即是等价性证明，比新写用例更有说服力。

## Risks / Trade-offs

- [manager 路由查表改造破坏对账触发] → 查表结果与原 Contains 链逐一对拍（B1 表格用例枚举现有全部路径），存量 B2 兜底。
- [translator 仍吃 bizv1 Spec，Vendor 解析加了 device.Store 依赖] → crdsource 本就持有 store 句柄（YR-03 建连同源）；输入契约解耦刻意推迟（Non-Goal），避免一次动两层。
- [driver 包与 controller name 字符串耦合] → controller name 本就是 manager 注册键（现状），描述符只是把散落的字符串收拢；①终态再模型化。
- [swagger 加字段引发前端契约漂移] → `make gen-contract` 再生成 api.gen.ts 一并提交，CI 漂移门禁兜底。

## Migration Plan

单 PR 渐进：S1→S2→S3 各自独立 commit、每步全量测试；回滚=revert（Vendor 字段缺省行为等价，可安全回退）。无数据迁移（device.Store 是内存注册表，重启即新 schema）。

## Open Questions

（无——插拔性已拍板；XML builder 声明式化的 descriptor 扩展形态留到下一 change 的 explore）
