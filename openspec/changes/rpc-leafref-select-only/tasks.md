# Tasks

## 1. Spec（先行，R17）

- [x] 1.1 delta spec：frontend FE-19 MODIFIED（leafref 输入禁自由文本，空列表=空下拉+占位提示）

## 2. 前端修复（TDD，F2）

- [x] 2.1 红灯：F2 回归测试——leafref 输入在目标列表拉取失败/为空时仍渲染 `leafref-select`（非 el-input），且带空态占位提示（T07）
- [x] 2.2 绿灯：`RpcExecuteTab.vue` resolvedInputs 对 leafref 字段始终注入 options（含空数组）；`FieldRenderer.vue` select 渲染条件适配空 options；空态占位文案 i18n（zh/en）
- [x] 2.3 存量测试适配：原「降级文本输入」断言改为「空下拉」断言；棘轮维持基线不降（T08，阈值不上调——本地实测被同机 staging 后端灌水，只锁 CI 可复现下界，见 vitest.config.ts 注释）
- [x] 2.4 F3 真浏览器防线：空/有 options 的 leafref 下拉键入任意文本+回车/失焦均不产生值（锁死无 allow-create 契约）

## 3. 收尾

- [ ] 3.1 code review + What/Why/How 提交
- [ ] 3.2 推送 + PR + CI 全绿
