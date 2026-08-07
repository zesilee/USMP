---
name: list-server-pagination
description: 做大表/分页/状态读/堆叠PR前必读：BR-13/BR-14 已交付（快照出口分页+阈值200双模式）、谓词下钻取数截断、sim config壳坑、堆叠PR删分支连坐坑、时序护栏要容 -race 弱机
metadata: 
  node_type: memory
  type: project
  originSessionId: 43e33343-cbbe-4b0f-b033-6db5a10e47d5
  modified: 2026-08-06T09:54:13.187Z
---

大 list 服务端分页已全量交付（2026-08-06，PR #292/#295/#294，归档 2026-08-06-list-server-pagination）。源起 NCE 抓包分析：控制器侧快照+出口分页是商用唯一解（NETCONF 无设备侧分页；NCE getUiData4GridCount 带 searchFromUIDB=true 实锤其从 UIDB 分页）。

**Why:** huawei-fib 万级路由（五键 route 挂三键 unicast-af 内）走 include_state 无缓存通道，真机每次翻页全量打设备必炸——真机接入前唯一硬性能缺口。

**How to apply:**
- 契约：`GET /config` 可选 `limit(1..1000)/offset/filter(<leaf>==v|<leaf>~=v 可重复)/sort/sort_dir`，仅 YANG list 生效，带参 data={rows,total,limit,offset}，**无参形状不变**（[[readback-subtree-peel]] 契约有回归锚点测试）；状态通道快照缓存独立实例，`USMP_STATE_SNAPSHOT_TTL` 可调（默认 30s），写后 InvalidatePrefix 刻意不触及。前端 ModuleListTab 首读 limit=200 自适应：≤200 纯前端零回归、>200 服务端模式（搜索/排序/翻页下推）。
- **谓词下钻双动作缺一不可**：深路径 subtree filter 选不回祖先 list 的键叶（sim/真机同 RFC6241 语义）→ 分页模式设备取数路径截到首谓词段前父容器（predicateFetchPath，快照按父容器共享），提取器再按谓词键值索引唯一行下钻（descendPredicates）。谓词值含 `/` 暂不支持（pathLocals 局限，明确报错）。
- **sim 坑**：`SetRunningConfigXML` 勿包 `<config>` 壳——壳会成为 running 树根，`<get-config>` subtree filter 匹配不到回空；存量带壳用例全是只走 `<get>` 才没炸。
- **堆叠 PR 连坐坑**（#129 变种实锤）：merge `--delete-branch` 会把以该分支为 base 的下游 PR 直接 CLOSE 且**无法重开**（base 分支已删）。正确顺序：合入时不删分支 → 用 `gh api -X PATCH .../pulls/N -f base=main` 改下游 base（本机旧版 `gh pr edit` 被 Projects classic GraphQL 弃用错误噎死）→ 再删分支。
- **时序护栏**：10k 行查询护栏 500ms 在 CI 弱机 `-race` 下实测 0.61s 误报，已放宽 3s（拦 O(N²) 够用）——新增性能护栏一律按「-race 弱机 ×5」估上界（[[backend-ci-flaky-tests]] 同源教训）。
- 未抄 NCE 的（评估过、刻意不做）：独立 count 接口、fields 列裁剪、key/value 平铺行、复杂 or/complex 过滤。
- 两个 follow-up 已闭环（2026-08-07）：①约束人话占位=FE-22 补默认值段已交付（PR#298，`整数 合法范围: [10, 600]，默认值: 300`，dynamicDefault 优先不变）；②危险配置字段标记**拍板不做**——snd 模型 20 种厂商扩展里无危险标记数据源（NCE riskOperation 是自家私有元数据且抓包全 false），词表猜误伤、自造注解无领域知识（[[ext-ui-annotations]] 推迟决策同口径），等真机/业务给危险字段清单再议。
- D01 门禁坑：delta spec 的 MODIFIED 需求块必须**整块携带全部存量 Scenario**（归档=整块替换，缺=静默丢失），只写新增 Scenario 会被 CI 拦。
