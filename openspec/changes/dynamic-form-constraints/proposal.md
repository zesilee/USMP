# Change: dynamic-form-constraints

## Why

USMP 的 YANG 驱动配置前端（Stack B 主链路 `DeviceConfigPage` + `useDeviceConfig` + `FieldRenderer`）目前只消费 schema 的**扁平类型**（string/number/boolean/enum/group/list），把 YANG 里承载真实业务约束的四类元数据**全部丢弃**：

- `choice`/`case` —— ygot 生成时被拍平，前端无从渲染「互斥配置块切换」；
- `when` —— 字段级显隐条件（如 `../class='sub-interface'` 才显示 `parent-name`）完全不生效；
- `must` —— 跨字段约束（如 `suppress>reuse>ceiling`、`interval mod 10 = 0`）无校验；
- `pattern`/`range`/`default` —— DTO 已声明字段但后端从不填充，前端 `FieldRenderer` 也未接 `pattern`。

后果：模型驱动是**半截的**——控件类型对了，但业务语义（何时显示、何时合法、互斥选择）仍需人工判断或硬编码，违背 R05「YANG 自动渲染」的初衷。

本变更把引擎补成**100% 元数据驱动**：后端从 goyang schema 树把 `when/must/pattern/range/default` 透出、并从原始 `.yang` 恢复 `choice/case` 分组（ygot 拍平的部分），前端新增通用**约束引擎**（XPath 子集求值器）驱动显隐与校验、并把 `choice→Tabs/RadioGroup`、`leaf-list`、`pattern` 补进渲染层。**引擎不含任何厂商/模型硬编码**；以 **Huawei IFM**（`huawei-ifm.yang`）作为高复杂度演示数据（含嵌套 choice、复合 when、跨字段 must）验证通用性。

## What Changes

- **后端 `yang-api`（schema 端点）**
  - `FieldDef` DTO 扩展：新增 `when`（可见性表达式）、`must`（`[{expr,message}]`）、`nodeKind`（leaf/container/list/leaf-list/choice/case）、`cases`（choice 的分支分组）；填充既有但空置的 `pattern`/`minimum`/`maximum`。
  - `schema/entry.go` 从 `yang.Entry` 采集 `Extra["when"]/["must"]`、`Type.Pattern`、`Type.Range/Length`，经 `schema.LeafNode` 接口透出。
  - 从原始 `.yang`（goyang `Loader`）恢复 `choice/case` 分组，作为**呈现层元数据**注入 nested schema——**子叶保持真实扁平 path**，NETCONF 写入链路零改动。
  - 修正 `yang-api` 主 spec 陈旧契约（BR-03/BR-04 描述「预定义/硬编码 schema」与现状 `buildYangSchema` 动态生成不符），并迁移到 OpenSpec CLI 标准格式。
- **前端 `frontend`（约束引擎 + 渲染层）**
  - 新增 `useConstraintEngine`（`utils/xpathEval.ts` + composable）：YANG XPath 子集求值器（`../leaf`、`=`、`!=`、`and`/`or`/`not()`、`>`/`<`、`mod`、字面量），响应式计算每字段 `visible` 与 `must` 校验结果；语法错误**降级**为「字段可见 + 记录告警」（R08），不崩。
  - `FieldRenderer`：`choice→el-tabs`/`el-radio-group`（按 `cases` 渲染、切换 case 时清空非激活分支）、`leaf-list→可增删标签/输入行`、接 `pattern` 正则校验。
  - `DeviceConfigPage`：订阅表单数据变化→实时刷新 `visible`/`must` 错误态；提交前 `must` + `pattern` + `required`/`range` 全通过才 `POST`。
- **测试**：后端 B1/B3、前端 F1（求值器表驱动）/F2（choice/leaf-list/pattern 组件）/F3（el-tabs teleport 真浏览器）/F4（staging-smoke：IFM 切 `class` 显隐 `parent-name`、`must` 违例拦截提交）。

## Impact

- **Affected specs**: `yang-api`（MODIFIED BR-03/BR-04 + ADDED BR-05/BR-06 + 迁移标准格式）、`frontend`（ADDED FE-07/FE-08/FE-09）。
- **Affected code（后端）**: `internal/api/yang_handler.go`(FieldDef)、`internal/api/field_gen.go`、`pkg/yang-runtime/schema/{entry.go,types.go,schema.go,loader.go}`、`internal/generated/huawei` 原始 `.yang` 加载。
- **Affected code（前端）**: 新增 `utils/xpathEval.ts`、`composables/useConstraintEngine.ts`；改 `components/config/FieldRenderer.vue`、`views/DeviceConfigPage.vue`、`utils/crdSchemaParser.ts`(Field 类型扩展)。
- **写入链路（NETCONF/config codec）**: **不变**——choice/case 仅作呈现分组，子叶 path 保持扁平；`encodeToYgot`/`convertMapToHuaweiIfm` 消费的 map key 不变。
- **R 合规**: R04(ygot/后端为源)、R05(YANG 自动渲染)、R08(表达式降级不崩)、R09(约束引擎无共享可变态)、R10(不引入 safe-eval 等新依赖，自研轻量求值器)、R17(本 change 即 spec-first)。
- **约束**: 每 commit ≤500 行、PR ≤1000 行（TM04）；分阶段交付见 `tasks.md`。
