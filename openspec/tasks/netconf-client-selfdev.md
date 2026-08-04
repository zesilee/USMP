---
id: netconf-client-selfdev
title: 自研 NETCONF 客户端替代 scrapligo（四波渐进，双路径并行切换）
status: in_progress
priority: high
branch: 每波独立 worktree 分支（wave1: worktree-netconf-core-wave1）
worktree: .claude/worktrees/worktree-netconf-core-wave1
updated: 2026-08-04
origin: 用户拍板 2026-08-04「按4波方案做自研替代」；评估结论见当日会话（使用面=client 单包 8 动作、netconfsim 已自研服务端、真机兼容是唯一硬风险）
---

## 背景与约束

- scrapligo v1.4.0 使用面：仅 `pkg/yang-runtime/client`（netconf.go/rpc.go），8 个动作：
  Open(密码认证/免严格HostKey/端口/socket超时)、GetConfig(datastore+subtree filter)、
  EditConfig、Commit、Commit-confirmed(+timeout)+Confirm/取消、RPC(任意payload)、
  ServerCapabilities、Close。业务代码只依赖自有 NETCONFClient 接口，替换零波及。
- 已知 scrapligo 缺陷（自研要根治，勿复刻）：messageID++ 无锁、死连接 Close 死锁、
  被强杀内部竞态、异常关闭协程泄漏。现网补丁：opMu 串行化 + 有界关闭 + 故意泄漏（见
  netconf.go 注释与记忆 [[scrapligo-concurrency-pitfalls]]）。
- Go 1.22 钉死（记忆 [[go-122-pin]]）：新代码禁 1.23+ 语法；x/crypto v0.33 可用（已在依赖）。
- 军规：TDD 红绿（T01/T05）、B1 表格+race、B2 打 netconfsim（T02/T06）、存量改造
  必须「旧保留+新并行+双路径验证→切换→删除」（§5.3）。netconfsim 只报 base:1.0
  （EOM 封帧），chunked 1.1 路径靠单测 + 真机验证。

## 四波计划

- [x] **Wave 1**：协议核心 `pkg/yang-runtime/client/netconfcore`
  —— RFC6242 封帧（EOM `]]>]]>` + chunked `\n#len\n…\n##\n`，hello 恒用 EOM、
  协商后切换）、hello 构造/解析/能力协商（base:1.1 优先）。纯 stdlib，B1 全矩阵
  （拆包/粘包/畸形头/超限防 OOM/并发 race）。—— PR #258 合入 2026-08-04；
  评审揪出 EOM 定界符粘连丢字节 bug 已修
- [x] **Wave 2（超 TM04 拆两段）**：2a=会话+RPC 引擎（有锁 message-id、超时
  看门狗判死、Close 幂等；net.Pipe 假服务端覆盖 chunked 1.1 全链路）——
  PR #260 合入 2026-08-04；2b=SSH 传输 + 8 动作操作层 + B2 集成（真 SSH ↔
  netconfsim）——PR #261 合入 2026-08-04。
  实测踩坑已记：sim 按行读流，EOM 定界符后必须补 \n（scrapligo 同款），读端
  对帧间空白宽容。原整波 PR #259 因体积关闭，内容无损散入 2a/2b。
- [x] **Wave 3（本分支 PR）**：ncDriver 切换缝 + `USMP_NETCONF_IMPL=scrapligo|core`
  （**缺省仍 scrapligo**）。scrapligo 全部怪癖补丁原样迁入 backend_scrapligo
  （Kill=Channel.Close+recover、有界 Close）；core 芯经 backend_core 适配
  （rpc-error 折算 Failed、无截止 ctx 补 60s 默认操作超时、错误文案对齐
  "failed to open NETCONF connection"——AUTO 分派测试锁定的契约）。
  **两路各自全量绿**：现网 20+ 测试文件在 scrapligo 与 core 上均通过；
  compliance CI 新增双路径步骤 + `make test-netconf-core` 本地入口。
- [ ] **Wave 4**：真机灰度验证（华为 CE，重点：1.1 chunked、超大回包、hello 怪癖、
  keepalive）→ 缺省切 core → 稳定后删 scrapligo 依赖 + opMu 全局串行化等补丁代码，
  go.mod 移除 scrapligo。**门禁：真机验证通过前禁止切缺省。**

## 上下文恢复提示

- 评估细节与代码量分解：会话 2026-08-04；接口清单看 netconf.go 的 NETCONFClient 方法。
- chunked 编解码要点：chunk-size 首位非零、≤4294967295；`\n#` 后窥探 `#`=结束帧；
  防 OOM 设总量上限。hello 阶段永远 EOM。
- 现网测试套（client/*_test.go 20+ 文件）= 换芯验收卷子，Wave 3 起两路全跑。
- 真机窗口是日程不确定项；Wave 4 前所有波次不改变缺省行为（scrapligo 继续在岗）。

## 恢复指令

`/task resume netconf-client-selfdev`；看本文件四波勾选状态接着做未完成波次。
