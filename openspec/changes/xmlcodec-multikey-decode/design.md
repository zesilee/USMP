## Context

`xmlcodec.Decode` 的 `entryKey`（`decode.go:294`）对 `ΛListKeyMap()` 返回多键的列表直接报 `multi-key lists unsupported`，错误沿 `Decode → decodeRunningConfig` 上抛，整树降级原始透传，前端零行。生成代码中共 360 个多键列表（68 模块），真机已实测踩中 devm `physical-entitys`、fib `route-statistics` 两处。

ygot 生成物已备齐所需数据：

- 多键列表的 map key 类型是生成的复合 key struct（如 `HuaweiDevm_Devm_PhysicalEntitys_PhysicalEntity_Key{Class, Position, SerialNumber}`），**每个字段带 `path:` tag**（`path:"class"` 等）；
- `ΛListKeyMap()` 返回 `map[string]interface{}`，键为 YANG 叶名（与 key struct 的 path tag 一致），值为**已解引用**的具体类型（与 key struct 字段类型一致）；任一键叶为 nil 时整体返回 error；
- key struct 字段名与条目 struct 的键字段名相同（`Class`/`Position`/`SerialNumber`），可用于缺键回退复制。

## Goals / Non-Goals

**Goals:**

- `entryKey` 支持复合 key struct：`len(km) > 1` 时按 path tag 填充 key struct 作为 map key。
- 根级列表（`Decode`）与嵌套列表（`decodeField` Map 分支）共用同一 `entryKey`，一处修改两处生效。
- 宽容语义对齐单键：`ΛListKeyMap()` 报错（键叶缺失）时不丢条目——多键场景从条目自身键字段复制构造 key struct。
- 单键列表行为零回归（既有 golden/往返测试全绿）。

**Non-Goals:**

- **删除通道多键支持**（`delete.go` XC-03）：维持明确不支持错误。多键列表绝大多数是 config-false 状态表，无删除需求；真需要时另开 change（需多键谓词编码，体量独立）。
- 前端改动：无。解码成功后走既有 `EmitJSON RFC7951 → peelToPath`，多键列表 emit 为普通数组，前端渲染链路不感知键数。
- `convertConfig` 写路径：ygot `Unmarshal` 自带多键支持，与本缺口无关。

## Decisions

**D1 — key struct 填充走 path tag 映射，不走字段序**

`keyType.Kind() == reflect.Struct` 时：遍历 key struct 字段，取 `path` tag 在 `km` 中查值，`ConvertibleTo` 校验后 `Convert` 赋值。理由：tag 是生成物的显式契约（与 `ΛListKeyMap` 键名同源），按字段顺序对位则依赖生成器的隐式排序，脆弱。不可转换 → 命名 list 的明确错误（R08），与现有单键 `key type not convertible` 负路径同风格。

**D2 — 判定分支挂在 keyType 形态上，而非 km 长度上**

以 `keyType.Kind() == reflect.Struct`（且非 `time.Time` 类特殊标量——生成物中 key struct 均为本地类型，无此干扰）为多键分支入口。理由：`km` 长度只在 `ΛListKeyMap` 成功时可知，缺键回退路径拿不到 `km`；keyType 形态是编译期确定的稳定信号，两条路径（正常/回退）共用同一判定。单键列表 keyType 恒为标量，既有路径原样保留。

**D3 — 多键缺键回退：从条目键字段反射复制，缺失字段零值**

`ΛListKeyMap` 报错时（任一键叶 nil 即整体 error，见生成物），按 key struct 字段名在条目 struct 上取同名字段：指针非 nil 则解引用赋值，nil/缺失保零值。理由：与单键「合成 key 保留条目」的宽容语义对齐（回读展示不丢行优先）；多个条目键完全缺失时 key struct 相同会互相覆盖——接受此退化（真机状态表键叶必回传，此路径仅防御畸形回读；比整表报错降级好）。备选「拼 idx 进某个 string 字段」被否：污染键值语义，前端会把合成值当真值展示。

**D4 — 不动 `Decode`/`decodeField` 调用点签名**

`entryKey` 签名不变（`entry, keyType, elemTag, idx`），仅内部扩展。两个调用点零改动，嵌套列表自动获益。

## Risks / Trade-offs

- [多键 key struct 含 enum 字段（如 devm 的 `Class`），若回读枚举文本超出本地模型被叶级容错跳过，则该键字段为 UNSET] → D3 语义天然覆盖（UNSET 参与 key struct，条目保留）；此形态在既有单键 enum-key 列表同样存在，非新增风险。
- [键完全缺失的多条目互相覆盖（D3 退化）] → 仅畸形回读触发；日志已有叶级容错留痕，接受。
- [360 个列表逐一验证不现实] → 机制型修复 + 代表性用例（devm 三键含 enum、嵌套多键、缺键、负路径）+ 既有全量往返测试兜底；B2 集成用 devm 真实回读形态。

## Migration Plan

纯后端引擎扩展，无部署/数据迁移。回滚 = revert 单 PR。

## Open Questions

（无）
