# config-cache — 差异 / 补全清单（反向还原）

## spec 与代码差异

- [ ] **读缓存未落地**：REST `GetConfig` 每次直连设备；`force_refresh`/30s 读缓存为 TODO（`config_handler.go:32`）
- [ ] **TTL 与 §8 不符**：实际 5min（`manager.go:125`），§8 描述运行配置缓存 30s
- [ ] **GlobalCache 死代码**：`InitGlobalCache`/`GetGlobalCache` 无调用者（`ttl_lru.go:24`）
- [ ] **LRU O(n) 扫描**：容量大时淘汰成本线性（当前 cap=1000 可接受）

## 改进建议

- [ ] 实现设备配置读缓存（key=IP+路径，TTL 30s，下发后主动失效），落地 `force_refresh`
- [ ] 统一 TTL 语义：区分「desired-state 存储」与「device-config 读缓存」两类
- [ ] 移除 GlobalCache 死代码或接入真实调用点
- [ ] LRU 改用链表/堆将淘汰降为 O(1)（视规模）
