# translation-engine — 差异 / 补全清单（反向还原）

## spec 与代码差异

- [ ] **D8 多厂商仅 Huawei**：Cisco/H3C/Juniper 仅枚举占位，无 translator 文件（`factory.go:21`）
- [ ] **Route 翻译未完成**：返回 `map[string]interface{}` 而非 ygot 结构（`huawei.go:54`）
- [ ] **System 翻译不支持**：基类返回 `NewUnsupportedError`，Huawei 未覆盖
- [ ] **接口用 interface{} 传参/结果**：spec/结果均 `interface{}`，R04 张力

## 改进建议

- [ ] 补全 Cisco/H3C/Juniper 翻译器或明确单厂商范围
- [ ] 实现 Route 翻译为 ygot 结构（huawei-ip 模型）
- [ ] 实现 System 翻译
- [ ] 用泛型约束替代 `interface{}` 提升类型安全（R04）
