# device-protocol — 设备协议层架构设计（as-built）

> **权威性**：✅ NETCONF 为权威（R02）；gNMI 为规划能力，工厂返回显式未实现错误（空壳已随 retire-idle-scaffolds 删除）。
> **还原基准**：2026-08-25 配置自审计随笔刷新（原 `main@b1cfbae` scrapligo 版已过期重写）。行为契约以同目录 `spec.md` 为权威。
> **上层导航**：`system-architecture/design.md`；本能力即 C5。

## 1. 职责

统一设备接入层：为每台设备维持连接、屏蔽协议差异，向 Reconciler 提供 actual 回读与配置下发能力（Get / Set / Close / IsConnected 及删除命令通道）。

## 2. 组件

### 2.1 NETCONF 引擎 = 自研 netconfcore（`backend/pkg/yang-runtime/client/`）
- **NC-01：scrapligo 已于 2026-08-04 整体删除、禁止回引**，守护测试 `client/no_scrapligo_guard_test.go` 以 import 闭包审计拦截，无运行时回退。封帧兼容坑与真机验证状态见 `docs/memory/netconf-selfdev.md`、`docs/netconf-core-field-validation.md`。
- 传输 SSH，默认端口 830（RFC6241）；get-config / edit-config / commit / RPC 执行（`ExecuteRPC` 有副作用不重试）。
- **写事务串行化**：写操作经 opMu 串行（历史根因：并发 messageID 竞态与死连接 Close 死锁，#131 修复沿革见 `docs/memory/scrapligo-concurrency-pitfalls.md`）。
- 编解码走通用 XML 引擎（`xmlcodec`，见 `yang-xml-codec` spec）：模块顶层容器嵌套树下发（扁平根曾致设备匹配不到，#145）；回读按请求路径剥层（peelToPath，`readback-subtree-peel` 契约）。

### 2.2 连接池与连接信息
- `ClientPool`：按设备复用连接、断线 lazy 重连、超时重试、异常上抛由 Reconciler requeue。
- 连接信息经共享 device store 收敛（Device CRD + Secret 凭据，ResolveConn，PR#175-#180）；协议缺省 `ProtocolAUTO` 实践恒解析为 NETCONF。
- **能力协商**：节点级设备能力运行时被动学习 + 快速失败；能力查询必须 Peek 不拨号（`device-capability-negotiation` spec，PR#284-#286）。

### 2.3 gNMI — 规划能力（未实现）
默认端口 9339/9340 仅作规划口径；工厂对 gNMI 协议返回显式未实现错误。旧空壳 stub（空 GetRequest/空 Set）已物理删除，待真实需求驱动再设计。

## 3. 数据流

```
Reconciler
   → ClientPool.Get(设备连接信息，经 device store 解析)     # 复用或新建
   → Client.Get(path)          # get-config(running) → 剥层子树 → 解码
   → Client.Set(changes)       # 嵌套容器 XML edit-config + commit（opMu 串行）
   → 删除走 DELETE 命令通道（先移 desired 再下发，config-delete-semantics）
```

## 4. 并发与异常

- 池内 RWMutex 保护；写事务串行化；-race 为门禁常态（R09）。
- 设备离线：自动重连 + API 返回明确错误（§9 降级契约）；`<rpc-error>` 上抛不吞（Get 曾吞业务 rpc-error 的病灶已修，见 device-node-capability 台账）。

## 5. 关联

`netconf-switch-protocol` 技能（开发口径）；`netconf-simulator`（对端模拟）；`yang-controller-runtime`（C5 上下文）；`device-store` / `device-driver-registry` / `device-capability-negotiation`（连接信息、厂商描述符、能力）。
