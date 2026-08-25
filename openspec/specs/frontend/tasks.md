# frontend — 差异 / 补全清单

> 初版（`main@b1cfbae` 反向还原）所列 4 项差异与 4 条改进建议已全部闭环，台账如下（2026-08-25 审计刷新）。

- [x] **D9 两代动态表单并存** → 旧 `components/yang/*`/`useDeviceConfig`/`api/crd.ts` 已随 device-native-lowcode-config 与前端重建物理删除
- [x] **`BUSINESS_CRDS` 硬编码** / **K8sClient 依赖外部 proxy** / **CRD 配置态不入 store** → 前端 CRD 消费链 2026-07 整链退役（#143），问题连同载体消失；现状为 Stack B 直连 + 自研 store
- [x] 旧 yang/ 静态表单子系统清理 → 已删（React 重建后仓库 .vue 文件数为 0）
- [x] R11/R12 视觉复核 → 已在 NCE 配置台改版与 EviewUI 切换中系统处理（PR#238-241、#342-#404）

当前活跃前端工作见 `openspec/tasks/frontend-nce-parity-r2.md`。
