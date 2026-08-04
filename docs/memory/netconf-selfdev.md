---
name: netconf-selfdev
description: scrapligo 自研替代进度与门禁——碰 NETCONF client/netconfcore/连接问题排查前必读
metadata:
  type: project
---

# scrapligo 自研替代（2026-08-04，Wave 1-3 已合入）

- **现状**：双路径并行期。自研核心 `pkg/yang-runtime/client/netconfcore`（封帧/hello/Session/RPC 引擎/SSH 传输/8 动作）已全量交付（PR #258/#260/#261/#262）；`USMP_NETCONF_IMPL=core` 切自研芯，**缺省仍 scrapligo**。总纲与决策记录：`openspec/tasks/netconf-client-selfdev.md`。
- **等价性已锁定**：现网 20+ 测试文件在两芯上各自全量 -race 绿；CI compliance 有专门双路径步骤（改 client 相关代码两路都得绿）；本地 `make test-netconf-core`。
- **Wave 4 门禁（未完成）**：华为 CE 真机灰度（重点 1.1 chunked/大回包/hello 怪癖）通过前**禁止把缺省切到 core、禁止删 scrapligo**。
- 排查连接问题时注意先确认当前跑的是哪个芯（env `USMP_NETCONF_IMPL`）。
- 自研芯行为契约（改动前必读 ncdriver.go 注释）：rpc-error=业务错误进 Failed 不判死；传输错误→ErrSessionDead 判死绝不复用（自愈交 ClientPool 重拨）；无截止 ctx 补 60s 默认操作超时；错误文案 "failed to open NETCONF connection" 是 AUTO 分派测试锁定的契约。
- 实测积累的两个协议兼容坑（真机调试直接参考）：①按行读流的服务端要求 EOM 定界符后补 `\n`（scrapligo 同款行为）；②EOM→chunked 切换会残留帧尾空白，读端必须宽容帧间 `\r\n`。
- 相关：[[scrapligo-concurrency-pitfalls]]（三大缺陷=自研动机，core 芯已根治勿复刻）、[[go-122-pin]]（netconfcore 禁 1.23+ 语法）。
