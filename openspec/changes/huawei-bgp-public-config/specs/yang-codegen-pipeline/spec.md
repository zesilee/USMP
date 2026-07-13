## MODIFIED Requirements

### Requirement: CG-02 跨平台后处理（枚举标识符 + 确定性 schema 规范化）

生成管线 SHALL 使用 Go 实现的后处理器（`backend/tools/genfix`）对 ygot 生成物做两类幂等、机器无关的后处理，以满足 CG-01「同一仓库状态下重复执行输出字节一致」：

1. **枚举标识符修复**：修复含非法字符 `|` 的枚举标识符（如 `PortType_50|100GE` → `PortType_50_OR_100GE`），SHALL 在 Linux 与 macOS 上行为一致（不依赖平台 sed 方言）；SHALL NOT 改动枚举标识符之外的内容（含 YANG 原值字符串映射）。
2. **确定性 schema 规范化**：ygot 内嵌的 gzip schema blob（`var ySchema = []byte{…}`）序列化了 `yang.Entry` 的**无序集合数组**（首要为各节点的 `Augmented`——同一目标被多模块 augment 时，goyang 以非确定 map 迭代序应用，致数组元素顺序逐次不同、gzip 字节漂移）。后处理器 SHALL 解压该 blob、以稳定规则对**语义无序**的数组（`Augmented` 等，按元素规范化内容排序）与对象键做确定性重排、以固定参数（无时间戳、固定压缩级别）重新压缩并回填，使 blob 字节在重复生成间稳定。规范化 SHALL 保持 schema 语义等价：`ygot.GzipToSchema` 解出的 schema 与规范化前**结构与内容一致**（键集合、节点、类型、约束不变），SHALL NOT 重排语义有序的构造，SHALL NOT 改动数字/字符串字面量的值。

两类后处理 SHALL 幂等：对已处理或无匹配的输入执行为 no-op。

#### Scenario: 非法字符修复
- **WHEN** 生成物包含标识符 `HuaweiIfm_PortType_50|100GE`
- **THEN** 后处理后 SHALL 为 `HuaweiIfm_PortType_50_OR_100GE`，且对应 YANG 原值映射字符串（如 `"50|100GE"`）SHALL 保持原样

#### Scenario: 多-augment 闭包 schema 确定性
- **WHEN** 生成集含多模块 augment 同一目标节点的闭包（如 huawei-bgp 拉入 network-instance/ifm 被 tunnel-management/ethernet/bfd 等多方 augment），连续两次 `make gen-yang`
- **THEN** 两次 `all.gen.go`（含 `ySchema` blob）SHALL 字节一致（满足 CG-01 / CG-03 regen-and-diff）

#### Scenario: schema 规范化语义等价
- **WHEN** 对规范化前后的 `ySchema` 分别 `ygot.GzipToSchema`
- **THEN** 两者解出的 schema SHALL 结构与内容一致（节点键集合、类型、约束不变），仅无序集合数组的元素顺序被规整为稳定序

#### Scenario: 幂等 no-op
- **WHEN** 对同一文件执行后处理两次
- **THEN** 第二次执行 SHALL 不产生任何变更
