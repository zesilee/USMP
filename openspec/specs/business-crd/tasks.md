# business-crd — 差异 / 补全清单（反向还原）

## spec 与代码差异

- [ ] **D1 双 CRD 树抢注同 group**：`api/v1` 与 `api/biz/v1` 均注册 `biz.usmp.io/v1`，BusinessVlan schema 不兼容
- [ ] **D6 NativeDeviceConfig 下发 = TODO stub**：`applyNativeConfig` 仅 time.Sleep（`nativedeviceconfig_controller.go:223`）
- [ ] **生产控制器仍绑 legacy `api/v1`**：新树 `api/biz/v1` 仅被实验 reconciler 消费
- [ ] **NativeDeviceConfig 双语义**：core/v1 模型化 vs api/v1 透传，不兼容
- [ ] **旧树无 CRD YAML**：`api/v1` 未生成 `config/crd/bases`，未发布

## 改进建议

- [ ] 退役 `api/v1`，统一到 `api/biz/v1` + `api/core/v1`（唯一有生成 YAML）
- [ ] 生产入口切换绑定新树 CRD scheme
- [ ] 实现 NativeDeviceConfig 真实 NETCONF 下发
- [ ] 收敛 NativeDeviceConfig 为单一（模型化 core/v1）语义
