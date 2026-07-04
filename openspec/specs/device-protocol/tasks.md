# device-protocol — 差异 / 补全清单（反向还原）

## spec 与代码差异

- [ ] **gNMI Get/Set 空壳**：Get 发空 `GetRequest{}`（`gnmi.go:97`），Set 发空 Path/Val（`gnmi.go:154`）→ gNMI 实际不可用
- [ ] **NETCONF Subscribe 未实现**：返回 error（`netconf.go:258`）
- [ ] **无重试退避**：Get/Set/Discard 仅单次 lazy reconnect，无 backoff/重试计数（`netconf.go:86`）
- [ ] **`Release` no-op**：连接不主动释放，长期驻留（`pool.go:87`）
- [ ] **`CloseAll` 吞错**：关闭异常不可见（R08 瑕疵）
- [ ] **AUTO 恒落 NETCONF**：gNMI 分支仅显式端口 9339 可达，生产从不生效

## 改进建议

- [ ] 实现 gNMI Get/Set 的 Path/Val 编码，或明确弃用 gNMI 并从协议选择移除
- [ ] 为 NETCONF Get/Set 增加有界退避重试
- [ ] `Release` 实现引用计数/空闲回收
- [ ] `CloseAll` 聚合并上报错误
- [ ] 增加连接健康检查（可选后台探活）
