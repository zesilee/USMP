---
name: test-server-process-leak
description: 排查「8080 端口被占/后端起不来」时先看是不是遗留的 test-server 孤儿进程
metadata: 
  node_type: memory
  type: project
  originSessionId: d10c6dad-b098-4d16-8f18-ac7efad12905
---

`backend/cmd/test-server`（内存版 VLAN REST fixture，无 NETCONF）在开发中被编译成 `/tmp/tsrv` 之类的二进制后台跑起来后，**worktree 删除 / 会话结束时进程不会随之清理**，会长期霸占 8080。

**具体案例**：一个 `/tmp/tsrv`（本项目 `github.com/leezesi/usmp/backend/cmd/test-server` 编译产物）以 **root** 后台运行、PPID=1（孤儿守护），cwd 指向已删除的 worktree `.claude/worktrees/refactor-sim-t5t6-testserver-netsim/backend (deleted)`，连续跑了 ~4 天霸占 8080，挡住正常 docker compose / 真实 backend（都映射 8080）。

**排查手法**：
- `ss -ltnp 'sport = :8080'` 拿到 pid
- `ls -l /proc/<pid>/exe` 看真实二进制、`ls -l /proc/<pid>/cwd` 看来源 worktree
- `strings <bin> | grep usmp` 确认是本项目 `cmd/test-server`
- root 进程需 `sudo kill <pid>` 清理（Claude 会话内多半无权限，让用户用 `!` 前缀跑）

**How to apply**：遇到「8080 被占 / 后端起不来」先怀疑这类遗留孤儿进程，别急着改配置换端口。相关：[[dual-stack-migration]]、[[arch-optimization-roadmap]]（test-server 属 Stack B 直连调试工具链）。

> 2026-07-17 注：泄漏来源之一的 backend/test/integration（要求手工起 8080 服务器的 B0 腿）已随 retire-stacka-residue 物理删除；坑本身（cmd/test-server 编译成 /tmp/tsrv）仍需警惕。
