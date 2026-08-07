# Design: placeholder-default-value

## Context

FE-22 已交付 range/length 合成占位（FieldRenderer.vue `rangePlaceholder`/`lengthPlaceholder`）；`Field.default` 由后端契约携带（field_gen.go `f.Default`）。NCE waterMark 把默认值并入同一条占位提示。

## Goals / Non-Goals

**Goals:** 占位提示补齐默认值段，与 NCE waterMark 信息量对齐；双语。

**Non-Goals:** 不做危险字段标记（无数据源，见 proposal）；不动占位优先级；不动 boolean/leaf-list/嵌套控件；不动后端。

## Decisions

- **D1 合成规则**：`合成占位 = [范围/长度段][，默认值: <值>]`；两段都无 → 无合成占位（现状）；仅有 default → `默认值: <值>`。enum 下拉 placeholder 同规（仅 default 段，enum 无 range）。
- **D2 值展示**：default 原样字符串化（枚举名/数值/布尔字面量），不翻译值本身——设备语义值不本地化（与列表单元格口径一致）。
- **D3 dynamicDefault 与 default 并存时**：dynamicDefault 优先（「系统自动分配”语义强于静态默认值），维持 FE-15/FE-22 既有优先级链。

## Risks / Trade-offs

- [占位过长被截断] → el-input placeholder 原生省略号，信息前重后轻（范围在前默认值在后），可接受。
