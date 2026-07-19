# Tasks — ui-i18n

> spec 已先行（R17）。worktree 隔离。多 commit 按域拆（≤500 行/commit），预估 1-2 PR。

## 1. 基建（UI-01）

- [x] 1.1 依赖：vue-i18n@9；`make sync-snd-i18n`（cp snd/resources/i18n → frontend/src/assets/snd-i18n）并执行入库；pr-size/commit-msg 排除该目录（两处同步）
- [x] 1.2 F1 红灯：locale store（默认 zh-cn/切换/localStorage 持久化/非法值回退）
- [x] 1.3 实现：locale store + i18n 实例（legacy:false）+ App.vue ElConfigProvider 联动 + Header/Settings 切换入口 → 转绿

## 2. chrome 文案抽取（UI-02）

- [x] 2.1 locales/{zh-cn,en-us}.json 分域建 key（nav/layout/devices/logs/settings/console/business/dashboard/common）
- [x] 2.2 分域替换 t()（每域一 commit）：Sidebar+Header+左树（含 zh/en 字段切换）→ Devices → Logs+Settings → ModuleConsole → Business+Dashboard；F2 每域抽查双语断言
- [x] 2.3 清点门禁：`grep -rP '[一-龥]' frontend/src`（排除 locales/assets）= 0；`npm test` 全绿

## 3. YANG 字段标签（UI-03）

- [x] 3.1 F1 红灯：useFieldLabel（路径换算：rootContainer→sourceModule→res 键；命中/缺文件/缺键回退；locale 切换重查；用真实 vlan res 样本）
- [x] 3.2 实现：vite glob 懒加载 + 模块级缓存 + 左树映射消费 → 转绿；模块控制台/表格列/表单标签接入
- [x] 3.3 F2：vlan 控制台 zh 显「VLAN标识」/en 显英文/未覆盖字段回退节点名

## 4. 收官

- [x] 4.1 F4 smoke：语言切换冒烟（切 en 断言 2-3 锚点→切回 zh）；本地 `make e2e-local` 全绿
- [x] 4.2 typecheck + 前后端全量 + 覆盖率/阈值棘轮校验（前端 thresholds 如升同步上调）
- [x] 4.3 评审 → PR → CI 绿直接合（已授权）→ `/opsx:sync`+`/opsx:archive` → 更新 [[snd-integration-program]]（四期全部完成）
