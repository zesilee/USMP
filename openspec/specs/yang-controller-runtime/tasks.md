# yang-controller-runtime — 差异 / 补全清单（反向还原）

> as-built 与目标的差异 + 待办，非实施步骤。

## spec 与代码差异

- [x] **plugin 从不被调用** → 整包物理删除（retire-idle-scaffolds），扩展点由真实需求驱动再设计
- [x] **schema 层运行时为空** → `internal/yangschema.Load` 构建 schema 树 + manager `WithSchema` 挂载（device-native-lowcode-config）
- [x] **ConfigStore.List/ListDevices = stub** → 基于 `cache.Keys()` 枚举设备/路径（device-native-lowcode-config）
- [x] **两套 Reconciler 形态并存** → Actor 栈 2026-07-17 物理删除（retire-stacka-residue），GenericReconciler 为唯一形态
- [ ] **Source 接口不统一**：初版观察项（`Stop()` 签名），闭环状态未单独复核，处理前以当前代码为准
- [ ] **`DoneWaitGroup` 非线程安全**：初版观察项（R09 风险），同上
- [ ] **worker Get() goroutine 可能泄漏**：初版观察项，同上
- [ ] **`New` 死分支**：初版观察项，同上

## 改进建议

- [x] plugin / schema / ConfigStore 枚举三项 → 已随上表闭环
- [ ] 统一 Source.Stop() 签名；用 `sync.WaitGroup` 替换 `DoneWaitGroup`（若复核仍存在）
- [ ] 使 `queue.Get()` 支持 context，移除 worker 的 goroutine hack（若复核仍存在）
