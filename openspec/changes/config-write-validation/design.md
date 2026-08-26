# Design: config-write-validation

## D1 为什么必须先解循环依赖

`schema.Schema` 接口上的 `Validate(path, config)` 一直是空壳，不是因为没人想实现，
而是**实现不了**：

```
pkg/yang-runtime/validate  ──import──▶  pkg/yang-runtime/schema
        ▲                                        │
        └────────── 想调用（循环）◀───────────────┘
```

`validate` 用 `schema.Node` / `schema.LeafNode` / `schema.ListNode` 做走查，
必然 import `schema`；`schema.DefaultSchema.Validate` 想委托给它就成环，Go 直接编译不过。

三条出路，取第三条：

| 方案 | 做法 | 否决/采纳理由 |
|---|---|---|
| A. 依赖注入 | `DefaultSchema` 加 validator 字段，装配点注入 `validate.Object` | **否决**。没注入时静默退化为「只查路径」——又是一道看着存在、实际不设防的假防线，正是本次要消灭的东西 |
| B. 摘掉接口方法 | 从 `Schema` 接口删 `Validate`，调用方直接用 `validate.Object` | 可行且最小，但接口上少一个语义完整的能力；且调用方要各自拼「取节点 → 校验」两步 |
| C. 实现迁入 `schema` 包 | 校验实现搬进 `schema`，`validate` 包删除 | **采纳**。环从根上消失（不再有跨包边），`Validate` 缺省即生效、无注入空窗，调用方一步到位 |

迁入的合理性：校验逻辑本就是**纯 IR 走查**——只读 `schema.Node` 的约束元数据，
不依赖任何其他包。它待在 `schema` 包里是回到了它本来的位置，而不是硬塞。

## D2 迁移的语义冻结边界

`validate` 包的包注释携带一份**语义冻结契约**（YN-04 快照，
`internal/intent/validate_snapshot_test` 实证）：

- mandatory **不**校验（必填防线在 CRD OpenAPI required）
- min-elements **仅**对「存在但为空」的 list 生效（nil map 不触发，区分 nil 与空）
- must / when **不**做运行时求值（设备侧兜底）

**本次迁移一字不改这三条。** 迁移的验收标准是行为逐字节等价，不是「顺便改进」。
收紧任一项都属独立契约变更，须另行拍板——否则意图层的既有结论会跟着漂。

实现上：整段包注释随代码迁到新位置，`internal/intent` 的快照/双跑用例保持不变
（只改 import 路径），它们就是等价性的看门人。

## D3 校验插在写路径的哪一步

`SetConfig` 现有顺序：

```
解 JSON → convertConfig(类型转换) → validateConfig(VLAN 硬编码) → storeConfigMerged → 触发对账
```

新校验插在**类型转换之后、写 desired 之前**：

```
解 JSON → convertConfig → validateConfig(VLAN) → ★schema 约束校验★ → storeConfigMerged → 触发对账
```

三个位置理由：

1. **必须在 convertConfig 之后**：校验器走的是生成结构体的 `path` tag 反射，
   输入必须是已转换的类型化对象，不是裸 map。
2. **必须在 storeConfigMerged 之前**：拒绝要零副作用——不写 desired、不失效缓存、
   不触发对账、不触达设备。写完再校验就晚了。
3. **与 VLAN 硬编码校验并列而非替代**：VLAN ID 范围**没有编码进 YANG 模型**，
   通用校验器读不到这条约束。两者覆盖面不重叠，都留着。

## D4 校验的是提交增量，不是合并结果

`SetConfig` 的写语义是 merge（`storeConfigMerged` 把 incoming 并入已存 desired）。
校验对象取**本次提交的 incoming**，不是 merge 后的全量 desired。

理由：用户只该为自己这次提交的内容负责。若校验合并结果，一条存量的历史非法值
会让此后**所有**提交全部被拒，且错误信息指向用户根本没碰的字段——无从下手。
存量非法值的清理属另一议题（真要做得先有存量扫描与批量修复通道）。

## D5 错误呈现

拒绝返回 `code=400`，message 直接透出校验器原文——它已经带上了叶路径与被违反的
约束（如 `validate: /vlans/vlan/name: 长度 33 超出 [1, 32]`）。不做二次包装，
因为包装只会丢信息；前端表单的同源约束提示与之结论一致，用户不会看到互相矛盾的说法。

## D6 风险：校验过严误伤存量

这是本次唯一实质风险。校验器此前只在意图层跑过（覆盖面窄），接到原生写路径后
面对的是全部模块的真实配置，可能暴露它自身的缺陷（如某类 union / leafref 判定
过严），表现为「以前能提交的合法配置现在被拒」。

暴露手段三层，任何一层出现拒绝都必须逐条定性：

| 层 | 手段 | 判定 |
|---|---|---|
| B1/B3 | 全量 `go test -race ./...` | 既有用例被拒 = 大概率校验器过严 |
| B2 | 模拟网元端到端下发 | 种子配置被拒 = 校验器过严（种子是已知合法基线） |
| F4 | Playwright staging smoke | 前端正常流程被拒 = 前后端约束口径不一致 |

**判定原则**：被拒的输入若在真机上也会被拒或产生误配，则是校验器做对了，
按预期收下；若是合法配置被误拒，则修校验器而**不是**放宽接入范围——
放宽接入等于把防线又拆了。
