# rpc-leafref-select-only

## Why

用户反馈：「接口管理 → huawei-ifm → 清除振荡抑制次数」的接口输入应该只支持从下拉选择，不允许人工手动输入。排查发现这是全模型共性问题：`RpcExecuteTab` 对 leafref 输入只有成功拉到目标列表才渲染下拉，拉取失败或列表为空时**降级为普通文本框**，用户可随意手输（拼错的接口名会直接下发到设备，靠设备回 rpc-error 才被发现）。全部模块 40+ 处 leafref rpc 输入（ifm 接口名、devm 单板/端口位置、cfg 配置文件名/检查点、arp 等）共用此路径。

## What Changes

- **frontend / FE-19（MODIFIED）**：带 leafref 目标的 rpc 输入 SHALL 始终渲染为下拉（仅可从设备实际存在的目标值中选择），SHALL NOT 降级为自由文本输入。目标列表拉取失败或为空时，下拉 SHALL 呈空且给出明确占位提示（如「无可选项，请确认设备在线」）；mandatory 的 leafref 输入无值时执行按钮维持校验拦截（既有行为）。
- 实现集中在 `RpcExecuteTab.vue`（resolvedInputs 降级分支）与 `FieldRenderer.vue`（select 渲染条件），一处修复全模型统一生效，无逐模块改动。

## Impact

- 受影响 spec：`frontend`（FE-19）
- 受影响代码：`frontend/src/components/config/RpcExecuteTab.vue`、`frontend/src/components/config/FieldRenderer.vue` 及对应 F2 测试
- 行为取舍（有意为之）：设备离线/列表拉不到时，用户将无法执行需要 leafref 输入的 rpc（此前可手输强行执行）。这正是本需求要的效果——杜绝手输错误值下发设备。
