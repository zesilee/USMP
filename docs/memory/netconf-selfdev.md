---
name: netconf-selfdev
description: NETCONF 引擎已全面自研化、scrapligo 已删（NC-01 红线）——碰 client/netconfcore/连接排查/加依赖前必读
metadata:
  type: project
---

# NETCONF 自研引擎（2026-08-04 全量收口）

- **scrapligo 已彻底移除**（交付红线 NC-01：版本交付编译不得依赖 scrapligo，用户 2026-08-04 拍板）：go.mod/go.sum 零 scrapli，守护测试 `no_scrapligo_guard_test.go` 拦重引；**禁止以任何形式重新引入**。
- **唯一引擎** = `pkg/yang-runtime/client/netconfcore`（封帧/hello/Session/RPC 引擎/SSH 传输）+ `backend_core` 适配层。原双路径开关 USMP_NETCONF_IMPL 已拆除（Wave 3 历史）。交付判据：`go version -m usmp-backend | grep scrapli` 无输出。
- **真机验证仍未完成（唯一敞口）**：移交测试团队按 `docs/netconf-core-field-validation.md` 执行（头号目标 1.1 chunked 封帧——模拟器只走 1.0 覆盖不到）。**没有运行时回退开关**，阻断问题回退=2026-08-04 前旧版本包；结论回来前勿用于生产设备变更。
- 引擎行为契约（改前读 ncdriver.go/session.go 注释）：rpc-error=业务错误进 Failed 不判死；传输错误→ErrSessionDead 判死绝不复用（重拨交 ClientPool）；无截止 ctx 补 60s 默认操作超时；错误文案 "failed to open NETCONF connection" 是 AUTO 分派契约；opMu 仍在（写事务跨 RPC 原子性，非 scrapligo 补丁）。
- 协议兼容坑（真机调试直接参考）：①按行读流服务端要求 EOM 定界符后补 `\n`；②EOM→chunked 切换残留帧尾空白，读端须宽容帧间 `\r\n`；③载荷尾 `]` 与定界符粘连需尾部后缀比对切帧。
- 总纲/波次记录：`openspec/tasks/netconf-client-selfdev.md`（PR #258/#260/#261/#262 + 本移除 PR）。
- 相关：[[scrapligo-concurrency-pitfalls]]（已成历史背景：三缺陷=自研动机，core 已根治）、[[go-122-pin]]（netconfcore 禁 1.23+ 语法）。

**真机首战·get-config 过滤器坑（2026-08-04 修复）**：constructFilter 曾是自造 XPath 形态 `<filter … select="/ifm:ifm/…"/>`——RFC6241 无 type 缺省按 **subtree** 解释，空元素=什么都不选，真机正确回空 `<data/>` → 界面接口/VLAN 列表全空 + 对账拿空实际态永久漂移。已改与状态读同源（constructSubtreeFilter + `<filter type="subtree">` 外裹）。**同修 sim 保真**：netconfsim get-config 曾无视过滤器整树返回（掩体！），现按 RFC 三分支：无 filter=全量 / filter 空=空 data / 有 filter=子树匹配。诊断法宝：`curl /api/v1/config/<ip>/<path>` 返回的 data 是 base64 原始 rpc-reply，解码即见设备真实回复。

**双斜杠丢 namespace 坑（wire 抓包实证，已修）**：路径前导双斜杠（`//ifm:ifm/…`，URL 手拼常见）曾让注册表 HasPrefix 落空→filter 丢 xmlns→严格真机回空；constructSubtreeFilter 已做单斜杠规范化。**排障心法**：sim 的每一处宽容（namespace 通配/无视过滤器/空 filter 当全量）都是真机 bug 的掩体，修 bug 时同步收严 sim。
