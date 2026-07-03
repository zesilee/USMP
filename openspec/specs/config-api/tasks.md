# config-api - 补全清单

## spec 与代码差异

- [ ] **force_refresh未实现**：GET接口声明了force_refresh查询参数但代码中仅解析不使用（TODO注释）
- [ ] **GetConfig设备信息不完整**：pool.Get()只传IP，未传port/username/password，连接创建可能失败
- [ ] **无设备存在性校验**：GetConfig/SetConfig未校验设备IP是否已注册
- [ ] **响应格式不符合RESTful**：所有HTTP状态码为200，错误用JSON code区分
- [ ] **类型转换字段名大小写兼容**：convertToTypedStruct对字段名做了大量case兼容（ifName/Interface/vlans/Vlan），增加维护成本
- [ ] **VLAN结构体拼写错误**：代码中UnkownUnicastDiscard应为UnknownUnicastDiscard
- [ ] **无请求体大小限制**：SetConfig未限制请求body大小，可能造成内存问题
- [ ] **Reconcile失败无反馈**：SetConfig触发Reconcile后无后续状态追踪，客户端无法知道配置是否成功下发

## 后续改进建议

- [ ] 实现force_refresh缓存失效逻辑
- [ ] GetConfig传递完整设备连接信息
- [ ] 添加设备存在性校验
- [ ] 引入标准HTTP状态码
- [ ] 统一YANG字段命名约定，减少case兼容代码
- [ ] 修复UnkownUnicastDiscard拼写错误
- [ ] 添加请求体大小限制
- [ ] 提供Reconcile状态查询接口