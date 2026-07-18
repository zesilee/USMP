## Context

- `NarrowModulesByCapabilities`（schema/capabilities.go）已实现：无模块能力→全量（model tree 权威）；有→按 namespace/`module=` 参数/名称子串匹配。调用方仅测试。
- `NETCONFClient.ServerCapabilities()` 已读 hello（netconf.go:369），现仅喂 `supportsConfirmedCommit`。
- Device CRD（api/core/v1）经 crdgen 管线出 `deploy/crds/devices.core.usmp.io.yaml`；device store 缓存连接信息（含 CRD 与内存两种后端）。
- blacklist.xml：23 条 module+revision（含 huawei-system 2020-03-02——正是在用 revision），NCE 语义未完全澄清。
- 用户拍板：role 人工指定、只做标签不参与裁剪；能力集由 hello 实测。

## Goals / Non-Goals

**Goals:**
- Device CRD + devices API + 前端设备页支持 `role`
- `/yang/modules?device=<id>` 返回 hello 协商子集；无参数/协商不可得时全量降级（R08，`negotiated` 标记诚实透出）
- blacklist 构建期解析 + 模块项 `blacklisted` 注解
- netconfsim 支持定制 hello capabilities（B2 前置）

**Non-Goals:**
- role 不参与模块裁剪（拍板；将来若要按角色预置白名单另开 change）
- blacklist 不做硬裁剪（含在用模块，语义澄清前只注解）
- 不改前端左树消费方式（③期统一重构时消费 `?device=` 与注解）
- 不做 gNMI 能力协商（gNMI 仍为规划能力）

## Decisions

**D1：能力缓存挂在 ClientPool 连接层，按 DeviceID 缓存 hello caps，断线重连后以新 hello 刷新。** 备选「存 Device CRD status」被否：能力是连接时事实、随固件升级变化，CRD status 写放大且引入 apiserver 依赖；连接层缓存零持久化符合 R03。API 查询时从 pool 取活跃连接的 caps，无连接→尝试建连一次，仍失败→降级全量+`negotiated:false`。

**D2：`role` 为自由字符串 + kubebuilder 枚举不设死（仅前端提示常用值 DCGW/EOR/TOR/BORDER）。** 网络角色词汇表随组网演进，CRD 枚举写死会让新增角色变 schema 升级；校验只限长度与字符集（域名安全字符 ≤32）。

**D3：blacklist 构建期 codegen（tasknamegen 同模式：tools/blacklistgen → yangschema/blacklist.gen.go），不做运行期 XML 解析。** 与 task-name 同理：运行期零 snd 文件依赖，升级包时重跑生成管线，R04 regen-and-diff 门禁顺带守护。

**D4：`?device=` 未注册设备返回 404 明确错误；已注册但离线降级全量。** 未注册是调用方错误（诚实报错），离线是运行态波动（降级可用，R08）。

**D5：netconfsim 增加 `SetHelloCapabilities([]string)` 场景注入。** 现 sim hello 固定；B2 需要「设备只声明 vlan+ifm」之类的裁剪场景。默认行为不变（零回归）。

## Risks / Trade-offs

- [真机 hello 能力 URI 形态与启发式匹配不符→漏配模块] → 匹配已有三重启发（ns 相等/module=参数/名子串）+ B2 用华为真实 URI 形态（`urn:huawei:yang:huawei-vlan?module=huawei-vlan&revision=…`）做用例；不匹配时宁可全量（协商失败降级）
- [ClientPool 缓存与设备实际能力漂移（固件升级不重连）] → 缓存随连接生命周期；重连即刷新；文档标注
- [前端 role 编辑与 CRD 校验不一致] → 校验规则单点在 CRD marker，前端仅提示常用值
- [覆盖率棘轮] → 新增逻辑全带表格驱动测试，完成后校验（留 0.1 抖动余量）

## Migration Plan

单 PR 预估可容纳（手写面 <1000 行）；超限则拆：PR-1 后端（CRD+协商+API+sim）、PR-2 前端 role UI。顺序：spec delta → sim hello 注入（D5）→ B1/B2/B3 红灯 → 实现转绿 → CRD regen（make gen-crd）→ 前端 F1/F2 → 收官 sync/archive。回滚 = revert（CRD 新字段 omitempty，向后兼容）。

## Open Questions

（无——角色/裁剪边界已拍板，blacklist 硬裁剪显式推迟。）
