# Tasks — nce-batch-commit（二期：攒批提交工作流）

> 六段对应 design Migration Plan，每段独立 PR（≤1000 行，TM04），TDD 先测后码（T01/T05）。
> 不涉及派生函数变更，GD-01 黄金不刷新。含 frontend 改动的提交前 `make e2e-local` 全绿（§6.2）。
> worktree 前端环境按既有踩坑记忆执行（node_modules 勿 symlink 入 docker、勿 npx vitest、
> Playwright 版本、e2e 断言限定 `.el-table__row` 作用域）。后端改动 `go test ./... -race` 全绿。

## 1. 后端纯函数层（PR-1）

- [x] 1.1 B1 红灯：edit-config 编码分派导出函数测试——从 `client.marshalChange` 提取（注册表分派 + `xmlcodec.Encode/EncodeDelete`），覆盖 add/modify/delete、无 XML 通道模块返回显式不支持错误、与既有 marshalChange 输出字节一致（回归锚点）
- [x] 1.2 实现编码分派导出纯函数（`pkg/yang-runtime/client` 或新 preview 包），`NETCONFClient.marshalChange` 改为其薄封装，绿灯
- [x] 1.3 B1 红灯：回滚反算测试（CS-02）——ADD↔条目删除、DELETE↔按基线重建、MODIFY 旧值重编码、混合变更集、空 diff 边界
- [x] 1.4 实现回滚反算纯函数，绿灯
- [x] 1.5 B1 红灯：xmlcodec 叶级删除测试（CS-05）——主键定位+目标叶 `nc:operation="delete"`、多叶、嵌套条目、幂等、不误删条目
- [x] 1.6 实现 `xmlcodec` 叶级删除编码，绿灯；race 全绿，commit（What/Why/How）+ PR-1

## 2. 后端接口层（PR-2）

- [x] 2.1 B3 红灯：`POST /config/changeset/preview` 契约测试（CS-01/02/03）——正向/回滚 XML + diff 树 + 基线来源标注、desired 优先→cache→实时 GET 基线链、无副作用（desired/缓存/审计零变化）、幂等、400 负路径
- [x] 2.2 实现 preview handler + 路由注册，绿灯
- [x] 2.3 B1 红灯：`intent.Fragment` 加 `Op` 字段与 prepare 删除映射测试——delete→DeleteChange/叶级删除、缺省 merge 零行为变化（BIO-03 既有测试全绿回归）
- [x] 2.4 实现 Fragment Op 扩展，绿灯
- [x] 2.5 B3+B2 红灯：`POST /config/changeset/commit` 契约与 netconfsim 集成测试（CS-04）——跨模块原子提交、中途失败 discard 整体回退（scenario.ErrorOnRPC 注入）、删除条目端到端、desired 后写时序、审计逐条目、意图链路零回归
- [x] 2.6 实现 commit handler（TxCoordinator 复用、设备锁、desired/缓存/审计/对账时序），绿灯
- [ ] 2.7 后端覆盖率棘轮核对（T08），commit + PR-2

## 3. 前端纯逻辑层（PR-3）

- [x] 3.1 F1 红灯：changeset store 测试——add/update/delete 条目、同条目合并（baseline 保首次快照）、删除待创建=直接移除、按设备隔离、清空、计数派生
- [x] 3.2 实现 `stores/changeset.ts`，绿灯
- [x] 3.3 F1 红灯：`configDiff` 删除表达测试——`DiffEntry.op`、基线有值清除→remove、基线无值清除仍忽略、既有 add/modify 语义字节不变
- [x] 3.4 实现 configDiff 扩展 + API client（preview/commit 两函数），绿灯
- [ ] 3.5 覆盖率棘轮核对，commit + PR-3

## 4. 前端工具栏与弹窗（PR-4）

- [x] 4.1 F2 红灯：攒批工具栏（FE-23）——四按钮渲染/禁用态、徽标计数、提示条出现与关闭、切设备隔离
- [x] 4.2 实现工具栏组件接入 `ModuleConsolePage` 页头，绿灯；i18n `console.batch` 域双语补齐（zh/en 行号对齐）
- [x] 4.3 F2 红灯：变更内容弹窗——树形三列、增/改/删着色与图例计数、空态
- [x] 4.4 实现变更内容弹窗（纯前端渲染），绿灯
- [x] 4.5 F2 红灯：试运行弹窗——双 Tab、正向/回滚双栏 XML、降级说明、基线标注、失败如实报错
- [x] 4.6 实现试运行弹窗（调 preview 接口），绿灯
- [x] 4.7 F3 真浏览器：工具栏弹窗开合、变更内容树展开、试运行 Tab 切换
- [x] 4.8 覆盖率棘轮核对，commit + PR-4

## 5. 前端链路切换（PR-5，BREAKING）

- [ ] 5.1 F2 红灯：`ItemDetailPane` 确定入集（FE-21）——不发写请求、同条目合并回填、创建态入集与标记行、未入集草稿切行确认
- [ ] 5.2 F2 红灯：行删除/批量删除入集（FE-16/FE-11）——待删除标记、取消删除、删除待创建、批量门禁跳过提示、标记合成视图
- [ ] 5.3 F2 红灯：字段级清除删除语义（FE-22）——基线有值→删除意图入集、基线无值→仅置空、必填拦截、tooltip 新文案
- [ ] 5.4 实现三路「确定」入集 + 删除/批量入集 + 清除升级；`useConfigSubmit` 单条即时编排退役，改为变更集提交编排（commit→回读→轮询对账→清空）+「重置」，绿灯
- [ ] 5.5 F2 红灯→实现：提交/重置编排——确认弹窗、成功清空刷新、失败保留变更集、对账进度复用、路由离开确认
- [ ] 5.6 F3 真浏览器：完整攒批闭环（编辑入集→变更内容→试运行→提交/重置）、批量删除菜单
- [ ] 5.7 F4：staging-smoke 即时下发断言全部改写为攒批闭环，`make e2e-local` 全绿
- [ ] 5.8 覆盖率棘轮上调核对，commit + PR-5

## 6. 收尾（PR-6）

- [ ] 6.1 D9 string length 元数据：B1 红灯（field_gen 透出 length）→实现→契约生成同步（`make gen-contract`）→FE-22 占位自动生效核验
- [ ] 6.2 `/opsx:sync`：两份 delta（config-changeset/frontend）合入主 spec
- [ ] 6.3 对照目标截图走查（变更内容/试运行/工具栏），差异清单确认或立后续项
- [ ] 6.4 记忆归档（docs/memory：二期交付与后续期待办——导出/配置项/跨设备变更集）+ `/opsx:archive`
