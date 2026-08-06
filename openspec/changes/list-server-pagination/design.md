# Design: list-server-pagination

## Context

- 现状读链路：`GET /config/:ip/*path` 整树回读（缓存优先，TTL 30s），前端 `ModuleListTab.vue` 在浏览器内做全量 filter + slice 分页（`filteredRows`/`pagedRows`）。
- 状态通道：`include_state=true` 走 NETCONF `<get>`（DP-09），**恒打设备、不读不写缓存**（config_handler.go 明确注释）。只读 Tab（FE-14）整树走此通道。
- 极端场景：`huawei-fib` `/fib/unicast-afs/unicast-af/routes/route`，`config false`、五键复合、真机万级行。每次翻页 = 一次全量 `<get>`。
- NETCONF 无标准分页能力（list-pagination 仍是 IETF draft，华为设备不支持）——「让设备分页」不是选项。NCE 的商用答案：控制器侧持快照（UIDB），出口分页。
- 约束：R03（仅 TTL+LRU 内存缓存）、回读子树剥层契约（无参响应形状不可变，存量消费方多）、R08（降级不崩溃）、多键解码已支持（PR#290）。

## Goals / Non-Goals

**Goals:**
- 万级行 list 在前端翻页/搜索秒开：全量拉取每 TTL 至多一次，出口只给一页。
- `GET /config` 无参数时行为与形状 100% 不变。
- 状态大表可用：快照缓存 + `force_refresh` 手动取实时（2026-08-06 拍板）。
- 前端小表零回归：阈值以下交互与现状完全一致。

**Non-Goals:**
- 不做设备侧分页 / 不改 NETCONF 编解码与 filter 构造（DP-09 原样）。
- 不做独立 count 接口、列裁剪（fields）、复杂过滤（or/complex 条件）——NCE 分析明确不抄。
- 不做跨页全选批量操作、不做服务端游标（offset 足够，快照内稳定）。
- 不改派生逻辑（deriveTabs/deriveColumns 等）——派生黄金不受影响。

## Decisions

### D1 分页位置：后端出口切片，缓存仍存整树

`GetConfig` 在取到整树（缓存命中或设备回读后）之后做 行提取 → 过滤 → 排序 → 切片。缓存键与存储粒度不变（`ip|path` → 整棵子树）。

- 为什么不是新端点：`GET /config` 已是唯一读通道，新端点会撕裂契约（缓存/新鲜度/节点不支持快速失败等语义要重复实现）。
- 为什么缓存存整树而非存页：页是视图不是数据；存整树使 total 免费、任意 filter/sort 组合免费、且与现有下发后失效逻辑零冲突。

### D2 查询参数与响应形状（仅目标为 YANG list 节点时生效）

```
GET /config/:ip/*path?limit=50&offset=0&filter=name~=GE&filter=admin-status==up&sort=name&sort_dir=asc
```

- `limit`（1..1000）出现即进入分页模式；`offset` 缺省 0。
- `filter` 可重复，语法 `<leaf><op><value>`，op 仅 `==`（等值）与 `~=`（包含，大小写不敏感）；多条件 AND。`<leaf>` 支持嵌套路径（`bandwidth-type/bandwidth-mbps/bandwidth`），值按字符串比较（RFC7951 数值/布尔先字符串化）。
- `sort` 单字段 + `sort_dir`；两侧均可解析为数值时按数值比较，否则字符串比较。未给 `sort` 时保持快照内原序（同一快照内翻页顺序稳定）。
- 分页模式响应 data：`{"rows": [...], "total": N, "limit": L, "offset": O}` + 既有新鲜度字段（cached/cache_age_seconds/ttl_seconds/source）。`rows` 元素 = 原 list 条目对象（带类型 JSON，不做 NCE 式 key/value 平铺）。
- 目标路径非 list 节点却带 `limit` → 400 明确报错（不静默忽略，防前端误用）。list 判定优先走 schema；schema 查不到时以「子树根下唯一数组值」兜底判定（与前端 normalizeRows 同规则），仍无数组则 400。

### D3 状态快照缓存：独立实例、默认 TTL 30s

`include_state=true` 从「恒打设备」改为快照缓存：独立的 TTL+LRU 实例（与配置缓存同一实现 `internal/cache`），键同为 `ip|path`，默认 TTL 30s、独立可配（`USMP_STATE_SNAPSHOT_TTL`）。`force_refresh=true` 绕过快照直打设备并回填。

- 为什么独立实例而非复用配置缓存：语义隔离（下发后失效逻辑只作用于配置缓存，状态快照不应被写操作触碰）；TTL 独立可调；避免 `include_state` 与普通读同键互相污染（现状注释里明确担忧的点）。
- 为什么默认 30s：与配置缓存对齐、用户已拍板接受「30 秒前快照」；万级 FIB 一次 `<get>` 数秒，30s 摊薄后翻页体验不受影响。
- 内存量级：万级路由行解码后数 MB/条目，LRU 容量沿用实例上限，淘汰即重拉，R08 无崩溃路径。

### D4 前端双模式：首读带 limit 自适应，阈值 200

列表 Tab 首次读取一律带 `limit=200&offset=0`（含只读状态 Tab，附 `include_state=true`）：

- `total ≤ 200`：一次已拿全量 → **纯前端模式**（现状交互原样：本地 filterRows/slice，零额外往返）。
- `total > 200`：**服务端模式**——翻页/每页条数/高级搜索/排序全部映射为查询参数重新请求；表格下方展示「总记录数 N」（FE-11 现有 UI 不变，数据源换成 `total`）。
- 高级搜索面板字段集不变（support-filter 叶），服务端模式下提交时转为 `filter` 参数（面板的等值/包含语义与 D2 一一对应）。
- 变更集攒批的 pending create 行仍在前端本地叠加展示（现状逻辑保留，不参与服务端分页计数）。
- 「获取数据源」按钮语义不变：`force_refresh=true` 绕缓存/快照全量重拉并回到第一页。

为什么自适应而非配置开关：省一次探测请求、无需人工按模块配置、小表自动零回归。

### D5 翻页跨 TTL 过期的延迟尖刺

快照过期后翻页会触发整树重拉（大表数秒）。缓解：前端翻页请求期间表格 loading 态（现有 el-table loading 复用）；不做 stale-while-revalidate（复杂度不值，force_refresh 已是显式逃生门）。

## Risks / Trade-offs

- [状态快照突破「状态求实时」旧约定] → 已拍板接受；`force_refresh` 保留实时逃生门；快照 TTL 独立可调，真机验证期可调小。
- [万级条目驻留内存] → LRU 容量上限兜底淘汰；行数据为解码后共享结构，无每页复制放大；真机验证期观测 RSS。
- [filter/sort 在动态 map 上逐行反射比较，万级行 CPU] → 单快照内 O(N) 字符串比较，Go 下万级毫秒量级；B1 加基准护栏（10k 行 filter+sort+slice < 50ms）。
- [无 sort 时依赖快照原序，重拉后顺序可能变化] → 同一快照内稳定已满足翻页一致性；跨快照顺序变化与现状（整树重拉）一致，不新增问题。
- [400 拒绝非 list 分页可能误伤 schema 盲区] → 兜底数组判定先行，仅两者都失败才 400；错误信息带路径与原因。

## Migration Plan

纯增量：无参数行为不变，前端旧版本对新后端完全兼容（不带 limit 即旧行为）；新前端对旧后端不兼容（会收到整树），同仓同发不存在版本错配窗口。无数据迁移，无回滚动作（revert 即可）。

## Open Questions

- （无——阈值 200、TTL 30s、filter 语法均已在本文档拍板，实现期如需调整在 tasks 中修订。）
