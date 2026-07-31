# Tasks — nce-console-redesign（一期：界面与交互对齐）

> 四段对应 design Migration Plan，每段独立 PR（≤1000 行，TM04），TDD 先测后码（T01/T05）。
> 派生逻辑变更触发 GD-01 全模块黄金刷新 + 人工核对（SF-04）。含 frontend 改动的提交前
> `make e2e-local` 全绿（§6.2）。worktree 前端环境按既有踩坑记忆执行（node_modules 勿
> symlink、勿 npx vitest、Playwright 版本）。

## 1. 派生纯逻辑层（PR-1）

- [x] 1.1 F1 红灯：`deriveDetailTabs(listField)` 测试——标量叶→主 Tab、嵌套 group→子表单 Tab、嵌套 list→子表格 Tab、无嵌套退化单 Tab、空 fields 边界
- [x] 1.2 实现 `deriveDetailTabs`（utils/moduleConsole.ts），绿灯
- [x] 1.3 F1 红灯：`deriveColumns` 双维度测试——默认集=现分层前 9（语义不变）、全集=全部标量叶、去重与顺序
- [x] 1.4 实现列派生「默认集+全集」双维度（保持现有默认集结论字节不变），绿灯
- [x] 1.5 F1：左树搜索过滤纯函数测试（zh/en/name 三口径、祖先链保留、不可用叶保态、无命中空态）+ 实现
- [x] 1.6 刷新 GD-01 全模块派生黄金（纳入 deriveDetailTabs 与列全集维度），逐模块人工核对 diff 并记录核对结论
- [x] 1.7 覆盖率棘轮核对（T08），commit（What/Why/How）+ PR-1

## 2. master-detail 重构（PR-2）

- [x] 2.1 F2 红灯：`ItemDetailPane` 组件测试——点行高亮+展开、面包屑主键、二级 Tab 渲染、关闭收起、创建态空表单、未提交草稿切行确认、只读 Tab 无编辑区
- [x] 2.2 实现 `ItemDetailPane`（提交编排 useConfigForm/useConfigSubmit/DiffPreview/ReconcileSteps 迁入；即时下发语义不变）
- [x] 2.3 ModuleListTab 接入详情区并**删除 el-drawer 编辑路径**；「创建」入详情区创建态
- [x] 2.4 F2 红灯→实现：列表工具区——刷新按钮、多选框列、查询时间戳+总记录数、分页加「前往」跳页
- [x] 2.5 F2 红灯→实现：列设置（全集勾选显隐、默认 9 列、横向滚动）、全列客户端排序、enum/boolean 列头筛选
- [x] 2.6 F2 红灯→实现：行操作「获取数据源」（force_refresh 回读、时间戳/新鲜度更新、失败如实报错保持原状）
- [x] 2.7 F3 真浏览器：master-detail 点行/切行/关闭、列头筛选弹层、列设置弹层、详情区嵌套 list 增删改
- [x] 2.8 F4：staging-smoke 抽屉断言全部改写为 master-detail 交互，`make e2e-local` 全绿
- [x] 2.9 覆盖率棘轮上调核对，commit + PR-2

## 3. 表单控件规范（PR-3）

- [x] 3.1 F2 红灯：FieldRenderer boolean→打开/关闭 radio（true/false 值语义、可选不选不入 payload、禁用态）
- [x] 3.2 F2 红灯：约束合成 placeholder（range/length 文案、dynamicDefault 优先级不被覆盖、无约束无占位）
- [x] 3.3 F2 红灯:key 叶钥匙图标+编辑态只读（创建态可填）；字段级清除（置空→不入 diff/payload、必填清除触发校验、tooltip 文案）
- [x] 3.4 实现 FieldRenderer/表单层上述规范，绿灯；i18n 新文案双语补齐（UI-02 无残留硬编码）
- [x] 3.5 三列栅格布局（窄视口降列、choice/leaf-list/嵌套表格整行、when 隐藏不占位）+ F2 断言
- [x] 3.6 ModuleFormTab 嵌套 group→二级 Tab（FE-02 新形态），既有折叠断言改写
- [x] 3.7 F3 评估：radio/清除/占位均为 happy-dom 可真实覆盖的交互（无 teleport/弹层），F2 已覆盖；既有 F3 套件 13 例回归全绿，`make e2e-local` 全绿（pre-push 实跑）
- [x] 3.8 覆盖率棘轮核对，commit + PR-3

## 4. 左树搜索与展开收起（PR-4）

- [x] 4.1 F2 红灯：左树搜索框（命中展开祖先链、清空恢复、不可用叶保禁用、无命中空态）与展开/收起全部按钮
- [x] 4.2 实现 LeftTreeMenu/Sidebar 搜索与展开收起（复用 1.5 过滤纯函数；LT 生成物与接口零变更）
- [x] 4.3 E2E 左树既有断言回归 + 新增搜索冒烟；`make e2e-local` 全绿
- [x] 4.4 覆盖率棘轮核对，commit + PR-4

## 5. 收尾

- [ ] 5.1 `/opsx:sync`：三份 delta（frontend/left-tree-navigation/console-derivation-golden）合入主 spec
- [ ] 5.2 对照 7 张目标截图逐项走查（左树/工具区/master-detail/三列表单/控件规范），差异清单确认或立后续项
- [ ] 5.3 记忆归档（docs/memory：改版决策与二期攒批提交待办）+ `/opsx:archive`
