---
name: schema-driven-test-harness
description: 改前端派生逻辑/加YANG模块/碰schema契约前必读——fixture+黄金测试地基（PR#223）
metadata:
  type: project
---

**schema 驱动测试地基已交付**（PR #223，2026-07-24 合入 main `de995a5`）。目的：把端到端验证从「起 docker 全栈 + 浏览器人工点」换成钉住的 schema fixture，模块级断言 **3/68 → 68/68**，无浏览器、毫秒级。四步替换计划的第 1 步，**零运行时改动**。

**两个新能力**（主 spec：`openspec/specs/schema-fixture-pipeline` SF-01~04、`console-derivation-golden` GD-01~04）：
- **fixture 导出**：`backend/tools/schemadump`（`make gen-schema-fixtures`）遍历 `yangschema.Load()` 全部 68 模块，经 `api.BuildYangSchemaNested` 导出 `backend/testdata/schema-fixtures/<module>.json`（7.8M，git 压缩后~0.4M）。动态发现（新模块自动纳入，不硬编码名单）。
- **派生黄金**：`frontend/test/golden/` 对全部 fixture 跑派生纯函数（`deriveTabs`/`deriveKeyField`/`deriveColumns`/`filterableFields`/`deriveSchemaTree`）钉黄金，一模块一份 `__data__/<module>.json`（3.3M）。

**关键接缝**：前端 `useDeviceConfig` 把后端 `data.fields`（FieldDef JSON）**零转换**直接当 `Field[]` 用 → `fixture.fields → deriveXxx` 就是黄金钉的派生。

**三个必须记住的设计约束**：
1. **信任锚点 SF-03**：导出绕开 HTTP，故有一条全 68 模块逐字节等值测试（`schema_fixture_equivalence_test.go`）证明「工具导出 == GET /yang/schema?form=nested」。少了它 fixture 会与线上契约悄悄脱钩、黄金全绿而渲染是错的。**改 field_gen.go/schema 派生时这条会红——那是对的，重跑 `make gen-schema-fixtures` 提交即可**。
2. **诚实边界 GD-04**：黄金只证明「派生确定、未非预期变化」，**不证明**派生对用户合理/视觉正确。别用「68/68」暗示更强保证。
3. **黄金只记派生结论**（GD-02），不含 schema 原文、不含 i18n 本地化 label（`__basic__` tab 的 label 走 i18n 故只记 name+kind）。改派生逻辑→黄金震动，用 `UPDATE_GOLDEN=1 npx vitest run test/golden` 刷新后人工核对受影响模块（**刻意不用 vitest -u**）。

**门禁**：fixture 漂移 = compliance.yml regen-and-diff（无条件跑，SF-04）；黄金漂移 = frontend-ci 的比对测试（trigger 含 `backend/testdata/schema-fixtures/**`，防纯后端 PR 改 fixture 漏刷黄金）。二者已加入 pr-size/commit-msg 体积排除清单（两处同步）。

**确定性**：goyang `schema/entry.go:65 sortedDir` 已对 map 键 `sort.Strings`，`Children()` 返回定序 slice，导出跨进程逐字节一致（实测）——regen-and-diff 不误报。

**follow-up（不在本 change 修）**：F1 = `deriveColumns` cap=9 对宽 list（vlan/snmp/arp/ifm）截断，enum 优先分层可能把 name/description 挤出前 9，纯 UX 议题，见归档 `golden-review.md`。

**未做的后续三步**（各自独立 change）：设备一致性矩阵（下发→回读→幂等→删除，587 可写 list，spike 实测自动生成命中率 55/57）、浏览器 E2E 瘦身 + 无头接触表、真机 driver + 发布门。详见 [[test-governance-military-rules]]。
