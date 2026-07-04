# actor-transaction — 差异 / 补全清单（反向还原）

> ⚠️ 本能力整体与 R01（禁 Actor 模型）冲突，标注 legacy；退役前需先把事务/版本能力迁入 Stack B。

## spec 与代码差异

- [ ] **模块路由 stub**：`extractModuleFromPath` 恒返回 "default"→回退首模块（`device_actor.go:264,159`）
- [ ] **`ReflectTranslator.ToPayload` 未实现**：stub（`translator.go:86`）
- [ ] **框架泄漏设备模型特判**：`fetchActualFromDevice` 硬编码 Huawei 结构（`model_actor.go:456,495`）
- [ ] **`internal/controller/vlan/actor_reconciler.go` 的 k8sConfigStore 为 stub**：Get「not fully implemented」，Set no-op（`:243`）
- [ ] **整体违反 R01**：Actor 模型被红线禁止，但被生产控制器 `backend/controllers/*` 依赖

## 改进建议（迁移前置）

- [ ] 评估 2PC/版本回滚能力是否迁入 Stack B 的 Reconciler/plugin，避免退役丢功能
- [ ] 实现真实模块路由（按 YANG 路径解析模块名）
- [ ] 消除框架内的厂商特判，下沉到 translator/ygot 层
- [ ] 完成后按 §5.3 双路径验证 → 切换生产入口 → 删除 actor 包
