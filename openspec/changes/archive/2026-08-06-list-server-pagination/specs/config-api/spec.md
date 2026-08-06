# Delta: config-api

## ADDED Requirements

### Requirement: BR-13 list 分页/过滤/排序查询

`GET /api/v1/config/:ip/*path` SHALL 支持可选查询参数 `limit`（1..1000，出现即进入分页模式）、`offset`（缺省 0）、`filter`（可重复，语法 `<leaf><op><value>`，op 仅 `==` 等值与 `~=` 大小写不敏感包含，多条件 AND，`<leaf>` 支持嵌套路径）、`sort` + `sort_dir`（单字段；两侧均可解析为数值时按数值比较，否则字符串比较）。分页模式 SHALL 仅当目标路径解析为 YANG list 节点时生效：行提取 → 过滤 → 排序 → 切片在后端**整树缓存/回读结果之上**完成，SHALL NOT 改变缓存键与缓存存储粒度（仍为整棵子树）。

分页模式响应 data SHALL 为 `{"rows": [...], "total": N, "limit": L, "offset": O}` 并保留既有新鲜度字段（`cached`/`cache_age_seconds`/`ttl_seconds`/`source`）；`rows` 元素 SHALL 为原 list 条目的 RFC7951 对象（保留类型，禁止 key/value 平铺）。未给 `sort` 时同一快照内翻页顺序 SHALL 稳定（保持原序）。

**不携带 `limit` 时，响应形状与行为 SHALL 与现状完全一致**（回读子树剥层契约不变）。

目标路径非 list 节点却携带 `limit` SHALL 返回 400 及含路径的明确错误；list 判定 SHALL 优先走 schema（谓词剥除后查询），schema 未覆盖时以「子树根下唯一数组值」兜底判定，两者均失败才拒绝。

**嵌套 list（谓词锚定下钻）**：目标 list 的祖先段可含键谓词（如 `/fib:fib/unicast-afs/unicast-af[vrf-name=x][af-type=y][position=z]/routes`，FIB 万级路由的唯一寻址形态）。分页模式 SHALL 在整树/快照之上按谓词键值索引唯一行后继续下钻提取目标行数组；谓词未命中任何行 SHALL 返回空页（total=0，设备无该行=合法空态），命中多行 SHALL 400（键不完整）。**无分页参数时的谓词读取契约（停剥返回父容器）SHALL 保持不变**。

#### Scenario: 谓词锚定嵌套 list 分页

- **WHEN** 对 `/fib:fib/.../unicast-af[三键谓词]/routes` 携 `limit=50&offset=100` GET（af 行内含千级 route）
- **THEN** rows SHALL 为该 af 行内 route 数组的第 101–150 行，total 为该行内 route 总数

#### Scenario: 谓词未命中返回空页（边界）

- **WHEN** 谓词键值在快照中无匹配行时携 `limit` GET
- **THEN** SHALL 返回 `rows=[]`、`total=0`，信封码 200

#### Scenario: 分页读取大 list

- **WHEN** 对 66 行的 `/huawei-ifm:ifm/interfaces/interface` 携 `limit=10&offset=20` GET
- **THEN** data.rows SHALL 恰含第 21–30 行条目对象，`total=66`，且携新鲜度字段

#### Scenario: 过滤与排序组合

- **WHEN** 携 `filter=admin-status==up&filter=name~=ge&sort=mtu&sort_dir=desc&limit=10`
- **THEN** rows SHALL 仅含 admin-status 为 up 且 name 含 "ge"（忽略大小写）的行，按 mtu 数值降序，total 为过滤后总数

#### Scenario: 无参数形状不变（回归锚点）

- **WHEN** 对同一 list 路径不带任何分页参数 GET
- **THEN** 响应 data.data SHALL 为以请求路径为根的整棵子树（现状形状），SHALL NOT 出现 rows/total 字段

#### Scenario: 非 list 路径拒绝分页（负路径）

- **WHEN** 对 container 路径（如 `/huawei-ifm:ifm/global`）携 `limit=10` GET
- **THEN** SHALL 返回信封码 400 且错误信息含该路径，SHALL NOT 触达设备

#### Scenario: offset 越界返回空页（边界）

- **WHEN** total=66 而携 `limit=10&offset=100`
- **THEN** SHALL 返回 `rows=[]`、`total=66`，信封码 200

### Requirement: BR-14 状态读快照缓存

`include_state=true` 的状态读 SHALL 从「恒打设备」改为快照缓存优先：命中未过期快照 SHALL 直接返回（响应 `cached=true`、`source="cache"` 及快照年龄）；未命中或过期 SHALL 经 `<get>`（DP-09）全量回读并回填快照。`force_refresh=true` SHALL 绕过快照直打设备并以新结果覆盖回填。快照 SHALL 存于独立于运行配置缓存的 TTL+LRU 内存实例（默认 TTL 30 秒、可经环境变量独立配置），配置下发/删除的缓存失效逻辑 SHALL NOT 触及状态快照。BR-13 分页参数 SHALL 与 `include_state=true` 可组合：切片作用于快照，同一快照内翻页 SHALL NOT 重复打设备。

#### Scenario: 快照命中翻页不打设备

- **WHEN** 状态 list 首次读取回填快照后，30 秒内携不同 offset 再次读取
- **THEN** SHALL 从快照切片返回（source=cache），SHALL NOT 向设备发起 `<get>`

#### Scenario: force_refresh 取实时

- **WHEN** 快照仍新鲜时携 `force_refresh=true&include_state=true` 读取
- **THEN** SHALL 直打设备取新数据、覆盖快照，响应 source=device

#### Scenario: 快照过期自动重拉

- **WHEN** 快照超过 TTL 后再次读取
- **THEN** SHALL 经 `<get>` 全量回读、回填快照并按参数返回切片

#### Scenario: 写操作不失效状态快照（边界）

- **WHEN** 对同设备同模块下发配置（触发运行配置缓存失效）
- **THEN** 既有状态快照 SHALL 保持有效直至自身 TTL 过期
