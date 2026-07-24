---
name: scrapligo-concurrency-pitfalls
description: 改 NETCONF client/连接池/排查「持续 500 EOF」前必读：scrapligo v1.4.0 三缺陷（非并发安全/死连接 Close 死锁/被强杀时内部竞态）与对应防线
metadata: 
  node_type: memory
  type: project
  originSessionId: 4bb4e275-ee8a-4f8f-a253-0dc201cb83f6
---

scrapligo v1.4.0（已是最新版，无上游修复）三个缺陷及 USMP 防线（PR #131 合入 main；前身 #129 堆叠在 #128 上被误合进死分支从未到达 main，见下）：

1. **Driver 非并发安全**：`buildPayload` 的 `messageID++` 无锁（并发 RPC → 重复 message-id → 响应被错领/丢失 → 挂到 60s op-timeout）；`Channel.Write` 无锁（帧字节交错，设备端解析卡死）。防线：`NETCONFClient.opMu` 串行化 Get/Set/DiscardCandidate（#128 先加，本分支保留并重构）。
2. **死连接上 `Driver.Close()` 必死锁**：read loop 阻塞在无缓冲 `d.errs<-` 发送、Close 阻塞在无缓冲 `d.done<-` 发送，互等。防线：`markDisconnected` 绕过 Driver.Close，异步 `driver.Channel.Close()` + recover；卡在 errs 的 read goroutine 会泄漏（量与断连次数同阶，接受）。
3. **连接被对端强杀时内部数据竞态**（channel reader vs `Channel.Read`）：-race 下测试杀掉带活跃连接的 sim 必误报。防线：重连回归测试用「优雅关闭底层 driver 但保留 connected=true」注入死连接状态，不真杀 sim。

**故障签名**：「GET /config 持续 500 EOF + reconcile 某路径 error: EOF 直到重启后端」= 传输层死后 `connected` 恒 true、ClientPool 的 IsConnected() 复用死连接（已修：isTransportError → markDisconnected → 重拨，Get 幂等重试一次）。触发源常是前端模块控制台并行拉多个 YANG 子树。

**Why:** 这三个坑排查成本极高（生产只见 EOF/挂起，根因在第三方库锁缺失），且「升级 scrapligo」不可行。
**How to apply:** 任何绕过 NETCONFClient 直接摸 `*netconf.Driver` 的代码都要过 opMu；写涉及连接死亡的测试用优雅关闭注入，别杀 sim；netconfsim.Simulator.Stop 已会强关活跃会话（否则 wg.Wait 永挂）。

**堆叠 PR 合序坑（2026-07-09 实翻车）**：base 为特性分支的堆叠 PR，若 base 分支的 PR 先合入 main（squash 后 base 分支即死），堆叠 PR 再点合并会合进死分支、**内容永远到不了 main**，且 GitHub 界面照样显示 MERGED 紫标。合并堆叠 PR 前必须先确认 base 已重定向到 main（base PR squash-merge 后 GitHub 不一定自动重定向）；合完必须验证 `git show origin/main:<file>` 里改动真实存在。#129 因此重投递为 #131。

相关：[[config-delete-semantics]]、[[reconcile-conninfo-debt]]、[[reconcile-convergence-3rootcauses]]
