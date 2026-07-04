# frontend — 差异 / 补全清单（反向还原）

## spec 与代码差异

- [ ] **D9 两代动态表单并存**：旧 `components/yang/*` + `components/DynamicForm.vue` + `useDeviceConfig` + `api/crd.ts`(SSE) 未接活跃路由（死代码）
- [ ] **`BUSINESS_CRDS` 硬编码**：模块→group 映射写死（`useConfigPage.ts:7`）
- [ ] **K8sClient 依赖外部 proxy**：dev kubectl proxy / prod `/api/k8s`（`useK8sCRD.ts:140`）
- [ ] **CRD 配置态不入 store**：每页态在 composable 局部 ref，非 Pinia

## 改进建议

- [ ] 清理旧 yang/ 静态表单子系统（确认无引用后删除）
- [ ] `BUSINESS_CRDS` 改为从后端/CRD 发现动态构建
- [ ] 明确 K8s 访问代理契约（dev/prod 一致化）
- [ ] 按 `web-design-engineer` 复核 R11/R12（反 AI 陈词滥调、图标规范）
