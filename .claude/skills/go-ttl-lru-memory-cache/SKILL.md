---
name: go-ttl-lru-memory-cache
description: 无数据库架构，设备YANG配置内存缓存，TTL自动过期、LRU淘汰、协程安全，作为 ConfigStore 后端给 Manager 使用
---

# 技能详情

## 一、激活时机
1. 当用户需求包含「内存缓存」「TTL过期」「LRU淘汰」「缓存清理」等关键词时激活。
2. 涉及「设备配置读取」「缓存管理」「配置下发后缓存失效」时启用。

## 二、核心约定（R03 载体）
1. **无数据库**：运行配置只存内存，禁止 MySQL/Redis/SQLite 等自管数据库；持久元信息走 K8s CRD（§8）。
2. **Key = 设备IP + YANG路径**，默认 TTL 30s，过期自动重拉；**配置下发后主动失效对应 Key**。
3. **协程安全**：所有操作经 `sync.RWMutex` 保护；**RLock 临界区内禁止写 map / 更新 LRU 队列**——过期剔除、LRU 触碰这类写动作必须持写锁（历史上这里出过读锁写 map 的竞态样例，评审重点盯）。
4. **生命周期**：定时清理协程必须有 `Stop()` 退出机制，避免协程泄漏（R09）。
5. 状态读（config=false）走 `<get>` 通道另有约定，见 `docs/memory/state-read-get-channel.md`。

## 三、开发口径
- **权威实现在 `backend/internal/cache/`**，新需求先读现有实现与其 `*_test.go`（表格驱动+并发 race 用例齐全），在其上扩展，不要另起炉灶重写一份缓存。
- 作为 ConfigStore 后端接入 Manager（§4 C1），reconcile 读 desired/actual 均经它。
- 改动必补 B1 层测试（正常/异常/边界/**并发 -race**），见 CLAUDE.md §5.6。

## 四、联动
- `yang-controller-runtime-dev`：ConfigStore 供 Reconciler 读取
- `netconf-switch-protocol`：下发成功后失效缓存，下次读触发重拉
