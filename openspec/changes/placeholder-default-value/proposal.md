# Proposal: placeholder-default-value

## Why

NCE waterMark 分析（2026-08-06）遗留小项：我们的约束合成占位（FE-22）只有范围/长度，缺「默认值」一截——NCE 完整形态是"整数 合法范围：[10..600] 默认值：300"。后端契约 `FieldDef.Default` 现成，纯前端合成缺口。

（同批评估的「危险配置字段标记」经探索**不立项**：snd 模型全部厂商扩展中无危险标记数据源，词表猜测误伤严重、自造注解无领域知识支撑且此前已拍板推迟——留待真机/业务方给出危险字段清单再议。）

## What Changes

- FE-22 占位合成扩展：字段携带 `default` 时占位追加「，默认值: <值>」；仅有 default（无 range/length）的字段给「默认值: <值>」占位；enum 下拉空值同样展示默认值占位。优先级不变（显式 placeholder > dynamicDefault > 合成）。
- 双语 i18n；boolean/leaf-list 不涉及（无占位语义）。

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `frontend`：FE-22 约束合成占位规则扩展（追加默认值段）。

## Impact

- 前端：`FieldRenderer.vue` 占位合成、i18n 词条；F2 组件测试。
- 后端零改动（Default 已在契约）；派生逻辑零改动（黄金不受影响）。
