---
name: state-read-get-channel
description: "config=false 状态读链路已交付（PR#219/#220）：sim <get>+状态 overlay、client WithStateData、GET /config 切 <get>、FE-14 深层剥除；含四个易踩坑"
metadata: 
  node_type: memory
  type: project
  originSessionId: 606223f8-4de5-4f34-a963-26469a9a23c3
  modified: 2026-08-05T08:34:23.100Z
---

config=false 状态字段端到端显示已交付（2026-07-21，PR#219 代码 + PR#220 spec sync/归档，change `netconf-get-state-read` 已归档）。链路：sim 状态 overlay（`SetStateDataXML`/`DemoStateSeed`）→ `<get>` 合并回读 → client `WithStateData()` → `GET /config` → FieldRenderer 禁用态回显。对账（Reconciler）仍走 get-config，diff 不受状态叶影响。

**Why:** 后续做状态相关功能（状态刷新、真机接入、新模块状态面）时避免重新踩坑。

**How to apply:**
- 真机也能显示状态：读路径已是 `<get>`（RFC6241 语义正确），不是 sim 专属 hack。
- 坑1 `<config>` 壳：sim running 树种子形态带/不带 `<config>` 壳不一致，合并/过滤须经 `unwrapConfig` 归一化（层级错会让状态树变顶层幽灵兄弟、filter 命中错树、配置叶静默丢失）。
- 坑2 payload 深层泄漏：Encode 是 populated-means-pushed；可写 group/嵌套 list 内的 readonly 子叶值会随组对象下发，`useConfigForm.stripReadonlyDeep` 按 schema 递归剥除（顶层排除不够）。
- 坑3 scrapligo filter：`Driver.Get(filter)` 会把传入串再包一层 `<filter type="subtree">`，不能复用 get-config 的 `<filter select>` 包装；`constructSubtreeFilter` 由 path 构造真 subtree 体（谓词值含 "/"，先剥 `[...]` 再切段）。
- 坑4 模型差异：该型号 vlan 条目无 `status` 叶，VLAN 状态面是 `statistics` 计数器容器（uint64 RFC7951 序列化为字符串）。
- sim 状态合并语义：复用 edit-config 键匹配（wellKnownListKeys），无配置匹配的 keyed list 条目丢弃（防删除后幽灵）；纯状态容器照常并入；get-config 恒不泄漏状态。
- 覆盖率棘轮已上调：后端 71.0、前端 84/78/77/84（[[test-governance-military-rules]] 的旧值过期）。

**读通道拆分（2026-08-05 真机回归，覆盖本条早期结论）**：GET /config 缺省已改走 **<get-config> config-only 快通道**（真机 <get> 在 ifm interfaces 收集全量硬件状态 30s+ 不回首字节，wire 实证）；config=false 状态改**按需单行读**——`include_state=true` + 谓词路径 `/…/interface[name='X']`（constructSubtreeFilter 已把谓词转 RFC6241 content-match），绕缓存不写缓存，详情面板 mergeReadonlyState 只合只读字段。列表页状态列一期不显示。

**只读 Tab 例外（2026-08-05 真机回归二发，PR#282）**：整棵 config false 子树（FE-14 只读 Tab，如 devm `physical-entitys`/`schedule-reboot`）是「单行谓词读」约定的**合法例外**——running 配置 schema 无此类节点，真机对 <get-config> 过滤器里的 state 节点回 `unknown-element`（error-info-code 313），**<get> 是唯一读法**；模拟网元不校验此约束，F4/E2E 拦不住，防线=F2 mock 参数断言（ModuleListTab/ModuleFormTab 断言 getConfig 第 4 参）。配套：`api/index.ts` getConfig 在 includeState 时 per-request timeout 30s（全局 15s 会先于后端 30s 超时，前端报错而后端成功）。若真机上只读 Tab 仍报 unknown-element = 设备软件版本无该节点，按能力隐藏 Tab（未做）。另存量债：`frontend/coverage/` 67 个陈旧报告文件被 git 追踪且未 ignore，跑一次覆盖率即污染 status，待清理。
