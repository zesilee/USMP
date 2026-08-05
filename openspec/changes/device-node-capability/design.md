# device-node-capability — 设计

## Context

- 模块级协商已交付（CN-01 hello caps 连接层缓存、CN-02 `?device=` 模块收窄、CN-03 blacklist 注解）。hello 只到模块粒度，模块内节点差异（设备软件版本没有某容器）不可预知。
- 真机对 `<get-config>`/`<get>` 过滤器中不存在的节点回 `rpc-error`：`error-tag=unknown-element`、`error-info-code=313`、`bad-element=<节点名>`、`error-path` 指向请求过滤器路径。netconfcore 已把 rpc-error 结构化为 `*RPCReplyError`（Tag/Severity/Message，会话仍可用）。
- 控制台每页签路径 = `configPathFor(module, seg2)`（`/devm/cards` → `devm:devm/devm:cards`），读走 `GET /config/:ip/*path`（只读 Tab 带 `include_state=true`，PR#282）。
- 模拟网元来者不拒，此类真机行为现无任何测试防线。

## Goals / Non-Goals

**Goals:**
- 设备没有的节点：用户看到「当前设备不支持此功能」占位，而非裸 rpc-error；同一设备同一路径不重复打设备。
- 运行时学习、零预配置：不依赖任何离线的设备型号→节点映射表。
- 学习事实可重测（force 逃生）、随重连自然刷新（设备升级场景）。
- netconfsim 可注入「按路径拒绝」，让此类行为从此有 B2 防线。

**Non-Goals:**
- `<get-schema>`（RFC6022）拉取设备真实 YANG 做叶级精确协商——工作量大（依赖闭包抓取+运行时 goyang 解析），留待叶级写失败高频出现后另立 change。
- 叶级（页签内单字段）差异的预先发现——被动学习仅覆盖到实际被请求的路径粒度；写失败时同样入集并友好提示，但不承诺表单预先隐藏个别叶。
- 跨实例共享学习结果——学习成本=一次失败请求，各实例独立学习（对齐 CN-01 不持久化）。
- 主动预探测（进控制台并发预读全部页签）——首次点开即学已满足体验底线，预探测作为可选 follow-up。

## Decisions

### D1 学习信号：unknown-element 归因到请求路径前缀

后端在 `GET/POST /config` 的设备错误分支识别 `RPCReplyError` 中 `Tag ∈ {unknown-element, bad-element}` 且 severity=error 的条目，将**本次请求的 YANG 路径**（非 error-path 原文——命名空间前缀形式不可控）记入该设备的不支持集。归因保守：仅当 bad-element 与请求路径某段名匹配才入集，否则按普通错误透传（防把瞬时/无关错误误学为「不支持」）。

### D2 存储：连接层内存集，毗邻 hello caps（CN-01 口径）

`map[deviceID]map[path]struct{}` + RWMutex，挂在连接/client 生命周期（与 hello caps 同处）：断线重连即清空重学；不落 CRD/磁盘（R03；学习成本一次请求，持久化不值得为此破 CN-01 先例）。并发安全走 B1 race 测试。

### D3 读写快速失败 + force 逃生（BR-12）

- 命中不支持集的 `GET /config`：不打设备，直接返回结构化错误 `{code:!=0, reason:"node-unsupported"}`（HTTP 200 统一响应格式口径不变）。
- `force_refresh=true` 绕过快速失败真打设备：成功 → 移除标记（设备升级后手动「获取数据源」即恢复）；再失败 → 标记保留。
- `POST /config`/变更集 commit：命中不支持路径的项拒绝并带同款 reason；不打设备。

### D4 北向透出：并入既有查询而非新端点

`GET /api/v1/yang/schema/:module?device=<ip>`：有 `device` 参数时响应附 `unsupported:["<seg2>",…]`（该模块下已学习的不支持子路径，相对模块根）。前端进控制台本来就取 schema，零额外请求拿到预标记。无 `device` 参数行为不变（向后兼容）。

### D4b hello 原文透出（CN-06）：deviations 侦察入口

`GET /devices/:ip/capabilities` 返回连接层缓存的 hello capabilities 原文数组（零加工）。用途：①诊断；②用户真机环境回传后评估设备是否自报 `deviations=`——若报，二期可只经 `<get-schema>` 拉小体积裁剪文件做「提前精确预知」，替代/叠加被动学习。本期不解析 deviations、不实现 get-schema。

### D5 前端占位态（FE-24）：诚实透出、不隐藏

- 页签照常渲染；命中预标记或运行中收到 `reason=node-unsupported` → Tab 内容区显示占位（icon+「当前设备不支持此功能」+一句解释），无创建/编辑/下发入口，Tab 头加淡化样式。
- 不隐藏页签：与 CN-03「仅注解不硬裁剪」同口径，避免用户找不到功能时归因困难；占位区提供「重试」（即 force_refresh 通道）。
- 错误识别以响应体 `reason` 字段为准，不做报错文案字符串匹配。

### D6 netconfsim 注入：按路径拒绝

sim 新增 `SetUnknownElementPaths([]string)`（或等价注入）：get-config/get 过滤器命中注入路径时回带 `unknown-element`+`bad-element` 的 rpc-error（复刻华为 313 形态）。B2 集成测试用它覆盖学习→快速失败→force 恢复全链路。

## Risks / Trade-offs

- **sim 注入盲区（检视记录）**：sim 的无 filter 全量读提前返回，不受按路径注入约束——现有消费路径全带 filter 暂无实害；若后续链路出现全量读需回补。
- **归因面（检视记录）**：bad-element 与请求路径任意段名匹配即入集，配置体内与路径段重名的后代节点被拒时会标记整条路径——force 逃生+重连清空兜底，接受为残余风险。

- **误学风险**：设备瞬时异常若恰以 unknown-element 形态返回会被误标——force_refresh 逃生 + 重连清空兜底；D1 归因保守进一步压低概率。
- **粒度上限**：被动学习最细到「实际请求的路径」；页签可读但内含设备没有的叶时，读不报错（get-config 子树过滤只匹配存在节点）、写才暴露——写失败同样入集友好提示，但表单不预知。接受，叶级精确属 get-schema 二期。
- **多实例学习不共享**：每实例首次各踩一次错——每设备每路径至多一次失败请求，代价可忽略。
- **`?device=` 透出耦合**：schema 响应体加可选字段，契约生成（frontend-contract-gen）需同步再生成，漂移门禁会拦。
