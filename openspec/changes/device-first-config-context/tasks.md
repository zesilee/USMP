# Tasks: device-first-config-context

> TDD（T05/T07）：每项先写红灯测试再实现。改动类型→必补层：store→F1、组件/页面→F2、端到端流→F4（§5.6）。

## 1. 全局设备上下文（store，F1）

- [x] 1.1 F1 红灯：`test/stores/device.*` 新增 `selectedDeviceIp` 选中/清除/初始空用例
- [x] 1.2 实现：device store `selectedDevice: Device|null` → `selectedDeviceIp: ref<string>`（IP 口径），`selectDevice(ip)`/`clearSelection()` 同步收敛；F1 绿

## 2. 模块控制台消费全局上下文（F2）

- [x] 2.1 F2 红灯：跨模块保持——同一 store 下先在 ifm 选中设备，重挂载（模拟切 `/module/vlan`）断言选中沿用
- [x] 2.2 F2 红灯：`?device=` 深链写入 store（非仅局部）；无 query 时沿用 store 现值
- [x] 2.3 F2 红灯：未选设备渲染引导空态（`data-test="select-device-empty"`），选中后恢复 Tab 渲染
- [x] 2.4 实现：`ModuleConsolePage` 局部 ref → store 双向绑定 + query 初始化优先级（D2）+ 空态分支（D3）；既有 ownership/deviceQuery 测试同步迁移；F2 全绿

## 3. 设备管理入口写上下文（F2）

- [x] 3.1 F2 红灯：「查看配置」点击后 store `selectedDeviceIp` = 该行 IP（保留跳转断言）
- [x] 3.2 实现：`goToConfig` 双写 store + query；F2 绿

## 4. 端到端（F4）

- [x] 4.1 staging-smoke 新增：查看配置进入 ifm → 左树切 vlan → 断言设备仍选中、配置按该设备加载
- [x] 4.2 staging-smoke 新增：直开 `/module/ifm`（无 query、无上下文）→ 断言引导空态
- [x] 4.3 `make e2e-local` 全绿

## 5. 收口

- [x] 5.1 全量前端单测 + typecheck + 覆盖率棘轮不降（T08，必要时上调阈值）
- [ ] 5.2 code review + What/Why/How 提交 + PR + CI 全绿
- [ ] 5.3 `/opsx:sync` delta → 主 spec；`/opsx:archive` 归档
