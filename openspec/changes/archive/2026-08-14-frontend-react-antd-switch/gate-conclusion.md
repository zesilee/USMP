# 垂直切片闸门结论（tasks 7.5，design D3）

**结论：通过。** antd 6.6 撑得住 R05 的两项命门能力，重建继续。

## 实测证据（2026-08-14，PR 11/N）

| 命门 | 验证方式 | 结果 |
|------|----------|------|
| Table 运行时动态列 | 68 个真实模块 fixture（backend/testdata/schema-fixtures，与派生黄金同源）全量驱动：每个 list Tab 的列由 `deriveColumns` 现场派生 → antd `columns` 配置数组承接（排序/enum·boolean 表头筛选/多选/列设置显隐/自定义单元格分派全部按列元数据生成） | 全部模块列集合与黄金同源纯函数结论一致，渲染零崩 |
| Form 运行时动态校验 | 同批 fixture 全字段校验探针（`fieldValidation` 空表单/示例值两态 + when/must 全表达式解析扫描）+ F2 交互用例（pattern 即时红/must 跨字段/range/required 门禁/dynamicDefault 豁免/when 显隐） | 全字段求值零异常；FE-02 全场景行内提示与提交拦截成立 |

配套：单元格分派（when 行级显隐/状态点/enum·boolean Tag）以按列类型合成的数据行在 68 模块口径下真实执行 render 路径；FieldRenderer 8 类型在真实 schema 全形态下渲染不崩（R08）；`data-test` 选择器沿用。独立评审判定：通过（有保留），三条保留项（force 失败保列表 §9 语义、gate 定时器收口、列断言精确化）已随本 PR 修复。

## 架构决定（如实记录）

**未采用 antd Form store 的 rules 引擎**（`Form.Item name` 绑定模式）：该模式要求 Form store 作为字段值数据源，与 `useConfigForm` 单一数据源冲突（双源同步是 bug 温床）。改用 `validateStatus/help` 受控展示 + `form/antdRules.fieldValidation` 现场求值，校验权威保持在 `form/configForm.isBlocked`（§9 双防线）。这是架构选择而非 antd 能力缺陷——能力面（运行时生成 required/pattern/range/must、失败不提交、行内提示）完整满足 FE-02。

## 已知边界

- 客户端排序/筛选口径；服务端分页双模式（FE-25）随 tasks 8.5 扩展。
- required 缺失不做行内红（无 touched 语义会满屏红），由提交门禁（blocked）拦截；行内 touched 语义与违例计数提示随详情区（tasks 8.1）细化。range 违例目前靠 InputNumber 钳制+行内红，是否入 blocked 门禁随 8.1 拍板（rangeViolations 已备）。
