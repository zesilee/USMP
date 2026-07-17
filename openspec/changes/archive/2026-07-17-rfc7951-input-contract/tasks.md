# rfc7951-input-contract — 任务

> 三 PR：①立项制品 → ②实现（TDD）→ ③纯删除 + sync/archive。

## 1. PR① — 立项

- [x] 1.1 change 四件制品 + `openspec validate` 通过（spike 结论沉淀进 design）
- [x] 1.2 提交、push、PR ①，CI 全绿合入

## 2. PR② — 单一解码路径实现（TDD）

- [x] 2.1 【测试先行】B1 包裹/锚点矩阵红灯：path=锚点零包裹 / 子路径单层与多层包裹 / 非前缀 400 / 段含 `[` 谓词 400 / 未注册路径 400 / Unmarshal 失败 400 透出原因 / 旧形状（复数键、camelCase）400
- [x] 2.2 driver.Descriptor + `EncodeAnchor` 字段（DR-05）+ 注册表单测；internal/drivers 六模块登记锚点
- [x] 2.3 config_codec：`convertConfig` 改单一路径（wrap + encodeToYgot，删 legacy 回退调用）——绿灯
- [x] 2.4 存量测试形状对齐：~20 处 `{"vlans":…}`/camelCase 改 RFC7951 真名形状（顺带覆盖率核对）
- [x] 2.5 B2：system form-tab 形状（`/system:system/system:system-info` 扁平载荷）经 netconfsim 端到端下发→回读
- [x] 2.6 `go test ./... -race` 全绿 + code review + 提交 push PR ②，CI 全绿合入

## 3. PR③ — 纯删除 + 收尾

- [x] 3.1 （提前并入 PR②——死代码拖覆盖率跌破棘轮）删 `convertToTypedStruct`+`convertMapToHuawei*`+助手（config_handler.go ~685 行，task 3.6 收口）
- [x] 3.2 （提前并入 PR②）删 yang-api alias 假 schema switch（task 2.5 收口；BR-04 最小降级行为保持）
- [x] 3.3 删 `DeviceConfigPage.vue`、`types/yang-schema.ts`、router 暂存注释；前端单测/typecheck/e2e smoke 全绿
- [x] 3.4 sync：config-api BR-05/BR-06、DR-05、FE-03 合入主 spec；archive change
- [ ] 3.5 记忆更新（arch-optimization-roadmap：P1 遗留 2.5/3.6/4.3 全收口）+ 任务归档 + worktree 清理
