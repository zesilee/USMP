## ADDED Requirements

### Requirement: FE-24 设备不支持节点占位降级态

模块控制台 SHALL 对「当前设备不支持」的页签呈现占位降级态而非错误弹窗：内容区显示占位提示（含「当前设备不支持此功能」文案与重试入口），SHALL NOT 提供创建/编辑/删除/下发入口，Tab 头 SHALL 有淡化视觉标记但 SHALL NOT 隐藏页签（诚实透出，与 blacklist 注解同口径）。判定 SHALL 以响应体结构化 `reason:"node-unsupported"` 为准，SHALL NOT 依赖错误文案字符串匹配。进入控制台时 SHALL 消费 schema 响应的 `unsupported` 预标记（CN-05）直接呈现占位态、不发取数请求；未预标记页签照常取数，收到 `node-unsupported` 错误 SHALL 即时转占位态。占位区重试 SHALL 走 `force_refresh` 通道，成功即恢复正常渲染。

#### Scenario: 预标记页签直接占位
- **WHEN** schema 响应含 `unsupported:["cards"]`，用户打开 cards 页签
- **THEN** SHALL 直接显示占位态且不发取数请求

#### Scenario: 运行中学习即时转态
- **WHEN** 未预标记页签取数返回 `reason:"node-unsupported"`
- **THEN** 该页签 SHALL 即时转占位态，SHALL NOT 弹裸错误信息

#### Scenario: 重试恢复
- **WHEN** 占位态页签点击重试且设备本次成功返回数据
- **THEN** SHALL 恢复正常列表/表单渲染

#### Scenario: 普通错误不误转（负路径）
- **WHEN** 页签取数返回不含 `node-unsupported` reason 的普通错误（如设备离线）
- **THEN** SHALL 沿用既有错误呈现，SHALL NOT 显示「设备不支持」占位
