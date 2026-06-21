# yang-api - 补全清单

## spec 与代码差异

- [ ] **Schema硬编码**：GetSchema的IFM/VLAN Schema全部硬编码在handler中，未从YANG模型文件动态生成
- [ ] **模块列表硬编码回退**：ListModules在无模块时返回2个固定示例，非真实数据
- [ ] **vendor固定为huawei**：代码中vendor字段写死为"huawei"，不支持其他厂商
- [ ] **type字段为数字字符串**：YangModuleInfo.type是Root.Type()的数字字符串表示，语义不明确
- [ ] **模块名别名映射**："Interfaces"映射到huawei-ifm，"VLANs"映射到huawei-vlan，映射关系硬编码
- [ ] **未知模块无错误**：请求不存在的模块返回通用Schema而非404
- [ ] **FieldDef与ygot结构体不同步**：Schema中的字段定义与generated/huawei结构体手动维护，容易不一致

## 后续改进建议

- [ ] 从YANG模型文件动态生成Schema（ygot→FieldDef自动映射）
- [ ] 移除硬编码示例模块，无模块时返回空列表
- [ ] 支持多厂商vendor字段
- [ ] 未知模块返回404而非通用Schema
- [ ] FieldDef自动从ygot结构体注解生成，保证同步