# Proposal: list-server-pagination

## Why

真机接入前唯一的硬性能缺口：当前 `GET /config` 对 YANG list 一律整树回读，前端 `ModuleListTab` 在浏览器内全量过滤+分页。模拟网元几十行没问题，真机上的大表会线性恶化——最极端的是 `huawei-fib` 转发表（`/huawei-fib:fib/unicast-afs/unicast-af/routes/route`，万级行、五键复合、`config false`）：它走 `include_state=true` 状态通道，该通道**刻意不进缓存**，每次刷新/翻页都全量打设备，真机上必炸。NCE 抓包分析（getUiData4Grid/getUiData4GridCount + searchFromUIDB）证实商用做法是「控制器侧持快照、出口分页」，而非让设备分页（NETCONF 无标准分页能力）。

已拍板（2026-08-06）：大状态表接受「短 TTL 快照 + 手动强制刷新」换取翻页秒开。

## What Changes

- **`GET /config` 增加可选 list 查询参数**：`limit`/`offset`/`filter`/`sort`，仅当目标路径解析为 YANG list 节点时生效；携带分页参数时响应 data 变为行切片 + 总数 + 分页元信息。**不带参数时的响应形状保持不变**（回读子树剥层契约 PR#281 有存量消费方，不破）。
- **状态通道快照缓存**：`include_state=true` 读取从「恒打设备」改为「短 TTL 快照缓存 + `force_refresh=true` 绕过」；快照 TTL 独立于配置缓存可单独调整（默认值由 design 定），仍遵守 R03（TTL+LRU 内存，无持久化）。分页从快照切片，翻页不重复打设备。
- **前端 ModuleListTab 双模式**：行数超过阈值时自动切服务端分页（翻页/搜索/排序走后端参数）；小表维持现状纯前端模式，零网络往返。只读状态大表（FE-14 只读 list）同样受益。
- 过滤语义一期只做**叶字段等值 + 包含（like）**，与前端现有高级搜索面板（support-filter 字段集）对齐；复杂条件（NCE 的 orFilterCondition/complexFilterCondition）明确不做。
- 明确不做（对齐 NCE 分析结论）：独立 count 接口（快照在手总数免费）、列裁剪 fields 参数（行是带类型 JSON，收益有限）、key/value 平铺行格式。

## Capabilities

### New Capabilities

（无——全部为存量能力的需求变更。）

### Modified Capabilities

- `config-api`：新增 list 分页/过滤/排序查询参数需求（仅 list 节点生效、带参时响应含总数与分页元信息、无参形状不变）；状态读通道从「不缓存」改为「短 TTL 快照 + force_refresh 绕过」（BR-01/BR-04 口径扩展到状态快照）。
- `config-cache`：新增状态快照缓存需求（独立短 TTL、与配置缓存共存于同一 TTL+LRU 机制、大条目受 LRU 容量约束）。
- `frontend`：FE-11 列表 Tab 分页需求扩展为双模式（超阈值自动切服务端分页，翻页/搜索/排序下推后端；小表维持纯前端）；只读状态 list（FE-14）适用同一机制。

## Impact

- 后端：`backend/internal/api/config_handler.go`（GetConfig 分页参数解析与切片）、`config_codec.go`（list 行提取，复用多键解码 PR#290）、`internal/cache`（状态快照 TTL 支持）；不动 NETCONF 编解码与 Reconciler 链路。
- 前端：`frontend/src/components/config/ModuleListTab.vue`（双模式切换）、`src/api/index.ts`（新查询参数）、`utils` 中 filterRows 语义与后端 filter 对齐。
- 测试：B1（切片/过滤/边界/race）+ B3（GET /config 契约：带参/无参双形状）+ B2（模拟网元大表集成）+ F1/F2（双模式切换）+ 派生黄金不受影响（不改派生逻辑）。
- 风险：状态快照突破「状态求实时」旧约定（已拍板接受）；大条目进 LRU 需评估内存上限；翻页跨 TTL 过期触发整树重拉的延迟尖刺需在 UI 呈现加载态。
