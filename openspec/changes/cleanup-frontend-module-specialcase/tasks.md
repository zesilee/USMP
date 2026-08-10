# 任务：前端遗留特判与死代码清理

> TDD（T05）：先改/迁测试（红），再改实现（绿）。改动类型=前端 util/组件逻辑 → F1+F2（§5.6）；含 `frontend/` 改动 → 推前 `make e2e-local`（§6.2）。

## 1. 测试先行（红）

- [x] 1.1 `test/utils/logRows.test.ts`：重写 opLabelOf 用例——已知模块（传 titles 映射）显示标题、未知模块回退段名、剥命名空间前缀、空路径回退通用标签；deriveLogRows 透传 titles
- [x] 1.2 新建 `test/utils/extractRows.test.ts`：迁移 useDeviceConfig.test.ts 中 extractRows 的 2 个用例（兼容三形态 + R08 降级），import 指向新位置
- [x] 1.3 `test/components/Header.test.ts`：面包屑用例改用现役路由名 `devices`（断言「设备管理」），移除 `vlan` 桩路由

## 2. 实现（绿）

- [x] 2.1 `utils/logRows.ts`：opLabelOf/deriveLogRows 改模型驱动派生（titles 参数）；`views/Logs.vue` 传入 menu store 标题映射
- [x] 2.2 新建 `utils/extractRows.ts`；`RpcExecuteTab.vue` 改 import；删除 `composables/useDeviceConfig.ts` + `test/composables/useDeviceConfig.test.ts`
- [x] 2.3 `Header.vue`：删 CRUMB_KEYS 四条死映射
- [x] 2.4 locale 清理：删 `logs.opVlan/opInterface/opSystem/opRoute`、`header.crumb.interface/vlan/route/native`（zh-cn + en-us）
- [x] 2.5 删除 `types/vlan.ts`、`types/interfaces.ts`

## 3. 门禁与收尾

- [x] 3.1 前端单测全绿 + typecheck 干净 + 覆盖率不低于棘轮阈值（T08）
- [x] 3.2 `make e2e-local` staging smoke 全绿（§6.2）
- [x] 3.3 code review + What/Why/How 提交，推 PR
