# Tasks: list-server-pagination

> TDD（T01/T05）：每组先写红灯测试再实现。按 §5.6 选层：B1+B3+B2 / F1+F2。
> 单 PR ≤1000 行（TM04）：建议按「组1-3 后端」「组4-5 前端」拆两个 PR 合入。

## 1. 后端：list 行查询引擎（B1 纯逻辑）

- [ ] 1.1 红灯：`internal/api` 新增 list 查询单测（表格驱动）——行提取（schema 判定 list / 数组兜底 / 两者失败）、filter（`==`/`~=`、嵌套路径、多条件 AND、值字符串化）、sort（数值/字符串、asc/desc、无 sort 保序）、切片（offset 越界空页、limit 边界 1/1000）、race（并发查询同一快照）
- [ ] 1.2 实现 `listquery`（行提取 → 过滤 → 排序 → 切片，纯函数、不触缓存），单测全绿
- [ ] 1.3 基准护栏：10k 行 filter+sort+slice 基准测试，确认毫秒量级（design D2 承诺）

## 2. 后端：GET /config 分页参数与状态快照缓存（B3 契约 + B1）

- [ ] 2.1 红灯：B3 契约测试——BR-13 五个 Scenario（分页读取/过滤排序组合/**无参形状不变回归锚点**/非 list 400/offset 越界空页）+ 分页模式响应含新鲜度字段
- [ ] 2.2 实现 GetConfig 分页参数解析与出口切片（缓存仍存整树、缓存键不变），2.1 全绿
- [ ] 2.3 红灯：B1/B3 测试——BR-14 四个 Scenario（快照命中翻页不打设备/force_refresh 直打/过期重拉/写操作不失效快照）+ CC-07 三个 Scenario（独立 TTL 环境变量/LRU 淘汰降级/race）
- [ ] 2.4 实现状态快照缓存实例（`internal/cache` 复用、`USMP_STATE_SNAPSHOT_TTL`、include_state 通道接入、与分页参数组合），2.3 全绿

## 3. 后端：模拟网元大表集成（B2）

- [ ] 3.1 集成测试（`*_integration_test.go`，`testing.Short()` 跳过）：netconfsim `SetStateDataXML` 注入千行级 FIB 状态子树（五键复合 list）→ 经 API 分页读取：首读回填快照、翻页不新增 sim 请求计数、force_refresh 重拉、过滤命中正确行
- [ ] 3.2 配置类大表同链路验证：sim 种入 500 行 vlan/interface，无参读取形状回归 + 带参分页一致性（total=无参行数）

## 4. 前端：双模式列表（F1 + F2）

- [ ] 4.1 红灯：F1 单测——api 层新查询参数编码（limit/offset/filter/sort 序列化、include_state 组合）；搜索面板条件 → `filter` 参数映射（等值/包含）
- [ ] 4.2 红灯：F2 组件单测——FE-25 五个 Scenario（小表零回归不发新请求/大表翻页带参重请求/搜索下推且页码复位/pending create 本地叠加不计 total/获取数据源复位第一页）+ loading 态
- [ ] 4.3 实现 ModuleListTab 双模式（首读 limit=200 自适应、服务端模式翻页/搜索/排序下推、total 展示、loading、pending 行叠加），4.1/4.2 全绿
- [ ] 4.4 覆盖率棘轮：新增用例后按 T08 上调前端 vitest thresholds 与后端 `.coverage-baseline`

## 5. 收尾门禁

- [ ] 5.1 `make e2e-local` staging smoke 全绿（含 frontend/ 改动的 pre-push 门禁）；派生黄金全量跑一遍确认零漂移（未改派生逻辑的证据）
- [ ] 5.2 `go test ./...` 全量 `-race` 全绿 + `go-code-review-check` 通过 + What/Why/How 提交
