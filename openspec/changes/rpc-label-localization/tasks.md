# Tasks — rpc-label-localization

## 1. spec-first（R17）
- [x] 1.1 delta：MODIFIED UI-03 覆盖 rpc 标签 + input 叶本地化，含 rpc 场景与回退场景

## 2. F1 本地化逻辑（TDD 红→绿）
- [x] 2.1 `useFieldLabels.test.ts` 加 `localizeRpcs` 用例：rpc 标签命中中文、input 叶命中中文、缺键回退原名、缺 res 整树回退、en-us 英文（先红）
- [x] 2.2 实现 `localizeRpcs`（`useFieldLabels.ts`）→ 绿

## 3. F2 控制台接线
- [x] 3.1 `ModuleConsolePage.vue`：保留 `rawRpcs`，`relabelFields` 并行本地化 rpc，守卫防竞态
- [x] 3.2 组件测：mock schema 含 rpc，断言 rpc Tab 标签本地化（非原始节点名）

## 4. 收口
- [x] 4.1 前端全量单测 + 覆盖率不下降（T08）
- [x] 4.2 go-code-review 自审 / 前端对应自审
- [x] 4.3 What/Why/How 提交
