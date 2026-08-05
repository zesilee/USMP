# device-node-capability — 任务清单

> TDD 红绿循环（T01/T05/T06）：每组先写测试（红）再实现（绿）。单 PR ≤1000 行，按组拆 PR。

## 1. netconfsim 按路径拒绝注入（B2 防线地基）

- [x] 1.1 测试先行：sim 注入 unknown-element 路径后，get-config/get 命中该路径回华为 313 形态 rpc-error（bad-element/error-path），未命中路径不受影响
- [x] 1.2 实现 `SetUnknownElementPaths`（或等价注入）并通过 1.1

## 2. 后端学习集（CN-04，B1）

- [x] 2.1 测试先行：不支持集 增/查/清（重连清空）+ 并发 race + 归因匹配（bad-element 与请求路径段名匹配才入集、不匹配透传）
- [x] 2.2 实现连接层不支持集（毗邻 hello caps），netconfcore RPCReplyError 补 bad-element/error-tag 归因字段（如缺）
- [x] 2.3 B2 集成：真链路（sim 注入）读取→学习入集→重连清空（顺带修存量 bug：client.Get 从不检查业务 rpc-error，报错 XML 被当数据返回）

## 3. config-api 快速失败与 force 逃生（BR-12，B3+B2）

- [x] 3.1 测试先行：已标记路径 GET 不打设备且 reason=node-unsupported；首次学习当次即结构化；force_refresh 绕过+成功清标记+失败保留；写路径/变更集命中拒绝
- [x] 3.2 实现 GET/POST /config 与变更集提交的门禁及错误语义
- [x] 3.3 B2 集成：sim 注入下全链路（读→学习→关注入仍快速失败→force 恢复）

## 4. schema 透出（CN-05/CN-06，B3）

- [x] 4.1 测试先行：`?device=` 响应附 unsupported（相对模块根首段）、空集省略键、无参数契约不变、未注册 404
- [x] 4.2 实现 yang_handler 透出；重新生成前端契约（契约漂移门禁）
- [x] 4.3 测试先行 + 实现：`GET /devices/:ip/capabilities` hello 原文透出（在线原文数组/未注册 404/离线空列表+negotiated:false）

## 5. 前端占位降级态（FE-24，F1+F2）

- [ ] 5.1 测试先行（F1）：错误 reason 解析 util、schema unsupported 预标记派生
- [ ] 5.2 测试先行（F2）：预标记页签直接占位不取数；运行中 node-unsupported 即时转态不弹错；普通错误不误转；占位区无创建/编辑/下发入口；重试走 force_refresh 成功恢复
- [ ] 5.3 实现 ModuleListTab/ModuleFormTab 占位态 + Tab 头淡化标记 + api reason 解析
- [ ] 5.4 变更集提交链路命中 node-unsupported 的友好提示（F2）

## 6. 收尾

- [ ] 6.1 E2E smoke 校验既有流不回归（`make e2e-local`）
- [ ] 6.2 覆盖率棘轮核对/上调（T08）
- [ ] 6.3 sync：delta spec 合入主 spec（/opsx:sync）
- [ ] 6.4 archive：change 归档（/opsx:archive）
