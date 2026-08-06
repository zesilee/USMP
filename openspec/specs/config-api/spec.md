# config-api — 设备配置读写北向接口

## Purpose

config-api 是 Stack B 北向 REST 接口，提供设备运行配置的**读**（`GET /api/v1/config/:ip/*path`，带 TTL 缓存 + `force_refresh` 绕缓存回读）与**声明式下发**（`POST …` → 存入 ConfigStore → 触发异步对账）。连接信息（IP/端口/凭据/协议）统一由共享 DeviceStore 解析（见 [[device-store]]）。

## Requirements

### Requirement: BR-01 配置读取（缓存优先）

`GET /api/v1/config/:ip/*path` SHALL 优先返回运行缓存（§8 TTL 30s）中的新鲜配置；缓存未命中时 SHALL 经共享 DeviceStore 解析的连接从设备回读（NETCONF `<get>`，含 config=false 状态数据），并回填缓存。回读结果 SHALL 为 RFC7951 结构（如 `{"interface":[{"name":…}]}`），可被前端列表化，而非裸 XML 字节；设备返回的 config=false 状态子树（如接口 `dynamic`、VLAN `statistics`）SHALL 原样包含在回读结果中——有则带出，无则不构造占位。响应 SHALL 携带 `cached` / `cache_age_seconds` / `ttl_seconds` / `source`（`cache`\|`device`）。

#### Scenario: 缓存命中
- **WHEN** 距上次读取 < TTL 且未带 `force_refresh`
- **THEN** SHALL 返回缓存数据，`source="cache"`、`cached=true`，不访问设备

#### Scenario: 缓存未命中回读设备
- **WHEN** 缓存过期/无 且设备已在 DeviceStore 注册
- **THEN** SHALL 用库中凭据回读设备，返回 RFC7951 结构，`source="device"`，并回填缓存

#### Scenario: 回读含状态数据
- **WHEN** 设备 `<get>` 回读返回含 config=false 子树（如接口 `dynamic`）的数据
- **THEN** 回读结果 SHALL 含对应状态字段（RFC7951 结构），前端只读控件可回显；写路径 payload 仍不含状态字段

#### Scenario: 设备无状态数据
- **WHEN** 设备回读仅返回配置数据（无状态子树）
- **THEN** 回读结果 SHALL 与改动前等值，SHALL NOT 构造空状态占位节点

### Requirement: BR-02 读取降级（离线/未连接/未注册）

读取路径 SHALL NOT panic（R08）。设备连接建立失败 SHALL 返回 `code=500`；连接存在但未就绪（`IsConnected()=false`）SHALL 返回 `code=503`。设备未在 DeviceStore 注册时以 AUTO/无凭据连接、认证失败 SHALL 归为连接错误返回。

#### Scenario: 设备未连接
- **WHEN** 回读时客户端 `IsConnected()=false`
- **THEN** SHALL 返回 `code=503` "Device is not connected"

#### Scenario: 建连失败
- **WHEN** 连接池建连报错
- **THEN** SHALL 返回 `code=500`，其余请求不受影响

### Requirement: BR-03 读取超时

设备回读 SHALL 受 10s 上下文超时约束；超时 SHALL 返回 `code=500` 且不阻塞。

#### Scenario: get-config 超时
- **WHEN** 设备回读超过 10s
- **THEN** context 取消，SHALL 返回 `code=500`

### Requirement: BR-04 force_refresh 绕缓存回读

`force_refresh=true` 查询参数 SHALL 绕过缓存、强制从设备回读并回填缓存（已实现；取代早期"参数被忽略"的行为）。

#### Scenario: 强制刷新
- **WHEN** 带 `force_refresh=true`
- **THEN** SHALL 跳过缓存直接回读设备，`source="device"`

### Requirement: BR-05 声明式下发

`POST /api/v1/config/:ip/*path` 的请求体 SHALL 为**以 path 为根的 RFC7951 子树**（YANG 真名、枚举用名字字符串）；服务端 SHALL 将其解码为强类型 ygot 结构 → 存入 ConfigStore → 触发对账，返回 `status="ACCEPTED"`。下发即接受语义：配置**存储成功即返回**，实际对齐设备由异步对账完成。ConfigStore SHALL 只存类型化结构，SHALL NOT 存原始 map。desired 的存储与对账触发 SHALL 以描述符编码锚点路径（DR-05）为 key（子路径下发归一化）：解码值以锚点为根，周期对账按模块路径入队读到同一 key，SHALL NOT 在子路径 key 留分叉副本。

#### Scenario: 下发被接受
- **WHEN** 提交合法 YANG 路径 + RFC7951 子树
- **THEN** SHALL 存入 ConfigStore、触发对账，返回 `ACCEPTED` + `reconciliation.triggered`

#### Scenario: 表单子路径扁平载荷
- **WHEN** 对 `/system:system/system:system-info` 提交 `{"sys-name": "sw-01"}`
- **THEN** SHALL 解码为对应容器的类型化结构并接受（锚点相对包裹语义）

#### Scenario: 子路径下发锚点归一化
- **WHEN** 对锚点的子路径下发成功后，周期对账以模块路径（锚点）读取 desired
- **THEN** SHALL 读到本次下发的类型化结构；子路径 key SHALL NOT 存有分叉副本

### Requirement: BR-06 注册表锚点解码路由

下发解码 SHALL 经 driver 描述符注册表单一路径：按 path 查得编码描述符 → 按其编码锚点（DR-05）把请求体机械包裹为锚点相对 RFC7951 → `ygot.Unmarshal` 根级解码。未注册路径、path 与锚点不构成前缀、path 段含 list 谓词、或解码失败时 SHALL 返回信封码 400（错误信息透出原因），SHALL NOT 回退手写转换器、SHALL NOT 静默存原始 map。

#### Scenario: 注册表命中并解码
- **WHEN** path 含 `ifm:ifm/ifm:interfaces` 且请求体为 `{"interface":[...]}`
- **THEN** SHALL 经 ifm 描述符解码为 `HuaweiIfm_Ifm_Interfaces`

#### Scenario: 未注册路径显式拒绝（负路径）
- **WHEN** 提交无编码描述符覆盖的路径
- **THEN** SHALL 返回信封码 400 并说明路径未注册，SHALL NOT 存入 ConfigStore

#### Scenario: 旧形状拒绝（负路径）
- **WHEN** 提交 legacy 形状（如复数键 `{"vlans":[...]}` 或 camelCase 叶名）
- **THEN** SHALL 返回信封码 400 透出 ygot 解码错误，SHALL NOT 下发

### Requirement: BR-07 对账异步触发

`TriggerReconcile(ip, path)` 的返回 SHALL 表示是否命中对应 Controller；无论是否命中，配置 SHALL 已完成存储。

#### Scenario: 无匹配 Controller
- **WHEN** 该 path 无注册 Controller
- **THEN** `reconciliation.triggered=false`，但配置仍已存储、响应 `ACCEPTED`

### Requirement: BR-08 无效请求拒绝

非法 JSON 或类型转换失败 SHALL 返回 `code=400`，SHALL NOT 存储或触发对账。

#### Scenario: 非法 JSON
- **WHEN** 请求 body 非合法 JSON
- **THEN** SHALL 返回 `code=400` "Invalid request"，不写 ConfigStore

### Requirement: BR-09 行删除通道（DELETE，命令语义）

`DELETE /api/v1/config/:ip/*path?key=<条目主键>` SHALL 以命令语义删除列表单条目：按 path 路由模型分支解析 key → 从 ConfigStore desired 移除该键（与合并写同临界区，幂等）→ 构造仅含 key 叶、带 `operation="delete"` 的 edit-config **同步下发**（candidate→commit）→ 成功后失效该设备运行缓存、记录审计、触发对账验证收敛。下发失败（含设备 data-missing）SHALL 原样透出错误、SHALL NOT 失效缓存或写审计。POST 合并语义与 `walkMap` merge/subset 收敛语义 SHALL NOT 因删除通道改变。

#### Scenario: 删除成功端到端

- **WHEN** 设备存在 vlan 10，请求 `DELETE /config/<ip>/vlan:vlan/vlan:vlans?key=10`
- **THEN** SHALL 返回成功，回读 SHALL 不再含 vlan 10（删除元素仅含 key 叶 `<id>`）
- **AND** desired SHALL 已移除键 10，二轮对账 SHALL 0 change（不复活、不漂移）

#### Scenario: 删除不存在条目（负路径）

- **WHEN** 请求删除设备上不存在的条目键
- **THEN** SHALL 透出设备错误（如 data-missing），SHALL NOT 失效缓存

#### Scenario: 非法 key / 未知路径（负路径）

- **WHEN** key 缺失、类型不合模型（如 vlan key 非整数）、或 path 无模型分支
- **THEN** SHALL 返回 `code=400`，SHALL NOT 触达设备

#### Scenario: 并发删除（R09）

- **WHEN** 并发对同一 path 发起删除与合并下发
- **THEN** desired 读改写 SHALL 串行化，SHALL NOT 丢更新或竞态（-race 全绿）

### Requirement: BR-10 删除的模型驱动门禁

DELETE SHALL 在触达设备前按 schema 校验：目标 list 节点 `operation-exclude` 含 `delete`、或目标节点为 readonly（config false）时 SHALL 拒绝（400 + 明确错误），与前端按钮门禁互为防御。schema 查不到该路径时 SHALL 放行（降级，R08——设备/转换层为最终权威）。

#### Scenario: operation-exclude 拒绝

- **WHEN** 目标 list 在模型中标注 `ext:operation-exclude` 含 delete
- **THEN** SHALL 返回 400 且 SHALL NOT 触达设备

#### Scenario: readonly 路径拒绝

- **WHEN** 目标路径为 `config false` state 子树
- **THEN** SHALL 返回 400 且 SHALL NOT 触达设备

#### Scenario: schema 未覆盖路径降级放行（边界）

- **WHEN** 路径在 schema 中查不到对应节点
- **THEN** 门禁 SHALL 放行，由模型分支路由/设备决定成败

### Requirement: BR-11 归属硬锁（缺省 409 拒绝 + force 逃生）

`POST /api/v1/config/:ip/*path` 与 `DELETE` 行删除命中业务意图认领路径时，缺省 SHALL 拒绝写入：信封码 409、message 指引「先删除/修改对应意图，或携带 force=true 强制下发」、data SHALL 携带认领意图 CR 名称列表（`intents`）；被拒请求 SHALL NOT 触达设备编解码与下发，SHALL NOT 产生审计记录。携带 query `force=true` 时 SHALL 放行下发，响应 SHALL 附加归属警告字段（认领意图名称、提示意图收敛会覆盖手改），且审计记录 SHALL 携带 force 标记与被覆盖的认领意图（见 operation-audit OA-01）。归属查询 SHALL 经 API 暴露供前端徽标渲染（一期行为不变）。

#### Scenario: 手改命中认领路径缺省拒绝
- **WHEN** 对被意图 X 认领的 vlan 路径 POST 配置（无 force）
- **THEN** SHALL 返回信封码 409、`success=false`，data.intents 含意图 X，SHALL NOT 下发、SHALL NOT 记审计

#### Scenario: force 放行并附警告
- **WHEN** 对被意图 X 认领的路径 POST 配置且 `force=true`
- **THEN** 下发 SHALL 照常接受，响应 SHALL 含 `ownershipWarning`（意图 X 名称与提示文案）

#### Scenario: 行删除同受硬锁
- **WHEN** 对被认领条目 DELETE（无 force）
- **THEN** SHALL 返回信封码 409 并携带认领意图，SHALL NOT 向设备下发删除

#### Scenario: 未认领路径无警告不拦截
- **WHEN** 对无任何意图认领的路径 POST 配置
- **THEN** 下发 SHALL 照常接受，响应 SHALL 不含归属警告字段

#### Scenario: 兄弟路径不受锁（负路径）
- **WHEN** 意图仅认领 vlan 路径，对同设备 ifm 路径 POST 配置（无 force）
- **THEN** SHALL 照常接受，SHALL NOT 返回 409


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
