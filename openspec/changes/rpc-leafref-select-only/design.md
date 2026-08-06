# Design

## 现状与根因

`RpcExecuteTab.vue` 的 `resolvedInputs`：leafref 字段仅当 `leafrefOptions` 拉取成功且非空时注入 `options` → `FieldRenderer` 按 `field.type === 'string' && field.options?.length` 渲染 `el-select`（filterable、无 allow-create，本身不可手输）。拉取失败/空列表时不注入 options → 落到普通 `el-input` 分支，可自由手输。

## 方案

契约改为「leafref ⇒ 永远下拉」：

1. `RpcExecuteTab.resolvedInputs`：只要 `f.leafRef` 非空即注入 `options: opts ?? []`（空数组也是"下拉"信号），不再以拉取结果决定控件形态。
2. `FieldRenderer`：string 分支渲染 select 的条件由 `options?.length` 改为 `options != null`（数组存在即 select，空数组=空下拉）。枚举等既有 options 消费方均为非空数组，行为不变。
3. 空下拉占位提示：options 为空时 placeholder 用 i18n 文案「无可选项，请确认设备在线」（zh/en 双语），并利用 el-select 自带 no-data 空态。
4. mandatory 校验拦截（submittable）不变：空下拉选不出值 → 执行按钮禁用，天然闭环。

## 替代方案（弃）

- el-select 加 `allow-create` 白名单式放开：与需求相反，弃。
- 拉取失败时禁用整个执行面板：过重，非 leafref 的其它输入不该连坐，弃。

## 测试

F2（happy-dom）：RpcExecuteTab——拉取失败/空列表两种情况断言渲染 `[data-test="leafref-select"]` 而非 el-input；FieldRenderer——空 options 数组渲染 select、undefined options 渲染 input（非 leafref 字段回归不破）。
