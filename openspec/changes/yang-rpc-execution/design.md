# yang-rpc-execution — 设计

## Context

平台的模型驱动链路（YANG→FieldDef→前端渲染→NETCONF→设备）目前只覆盖**配置树**。YANG 的 `rpc` 语句是另一类一等公民——"带输入参数触发一次设备操作、拿回结果"，与配置的"声明期望态→对账收敛"是不同范式。

用户诉求：`rpc` 在界面上像配置一样自动渲染、可执行。给定实例 huawei-ifm 的 `reset-if-counters-by-name`（input：`if-name`，leafref→接口名，mandatory）。显示层级：

```
接口管理 / 接口基础 / huawei-ifm
    ├─ 通用接口          （container，配置）
    └─ 按接口名清除统计   （rpc，操作）
```

即**同一模块内 container 与 rpc 平级呈现**。

### 四段已就位（探索实证）

| 环节 | 现状 |
|------|------|
| goyang 解析 | `Entry.RPC *RPCEntry`（含 Input/Output 子 Entry），构建期可提取 rpc 输入结构 |
| i18n | snd res 已含 rpc 及 input 叶翻译（`/huawei-ifm:reset-if-counters-by-name` + `/…/input/if-name`，另有 `/rpcs/<module>/` 分组约定） |
| 前端渲染 | FieldRenderer + 约束引擎（leafref/mandatory/pattern）已能渲染任意 FieldDef 树 |
| NETCONF 执行 | scrapligo `Driver.RPC(opts...)` 支持发任意 `<rpc>` payload（`driver/netconf/rpc.go`） |
| 模拟网元 | `classifyRPC` 结构化分发，未知 rpc 返回 ok——可扩 custom-rpc 分类 |

缺的只是把四段接起来 + 一个构建期提取工具 + 前端执行/回显交互。

### 关键约束

- **R05** 模型驱动：所有模块的所有 rpc 自动渲染，零前端硬编码。
- **R04** ygot 运行期 schema 不含 rpc → 必须构建期提取（同 lefttreegen/tasknamegen/blacklistgen 管线，生成物入库、运行镜像零 snd 依赖）。
- **R08** 禁止误操作：rpc 是**有副作用的设备操作**，执行前守护。
- **R02** 仅 NETCONF/gNMI；gNMI 仍为规划能力，rpc 走 NETCONF `<rpc>`。
- **§8** rpc 非配置态 → **不进 TTL+LRU 配置缓存**（无 desired/actual/对账语义）。

## Goals / Non-Goals

**Goals:**

- YANG `rpc` 模型驱动渲染：所有模块所有 rpc 自动出现在其模块内，与 container 平级；input 由 schema 渲染（含 leafref 下拉、mandatory 校验）。
- 端到端执行：选参数→执行→NETCONF `<rpc>` 下发→回显 `<rpc-reply>` 结果或 rpc-error。
- 高危 rpc（restart-if 等）执行前守护，防误触。
- 新接模块的 rpc 自动入列（不靠人记得补）。
- 首模块 huawei-ifm 端到端打通，管线泛化到全部模块。

**Non-Goals:**

- rpc `output` 复杂结构化表格渲染（首期回显 reply 原文/ok/error）。
- rpc 执行审计入 CRD（首期只前端回显）。
- 订阅类/长运行 rpc、gNMI 遥测（规划能力，不实现）。
- 用 rpc 做配置下发（配置仍走 edit-config 对账链路，两者不混）。

## Decisions

### D1 — 构建期提取 rpc，生成 .gen.go 入库

**决定**：新增 `backend/tools/rpcgen`，用 goyang 解析 YANG 源，从 `Entry.RPC` 提取每模块的 rpc（名称 + input 叶树：名/类型/leafref-path/mandatory/units/range/pattern），生成 `internal/yangschema/rpc.gen.go` 入库。`make gen-rpc`，CI regen-and-diff 验证零漂移。

**备选与否决**：

| 方案 | 否决理由 |
|------|----------|
| 运行期从 ygot schema 取 rpc | ygot 生成物是配置树，**根本不含 rpc**（实证 grep=0） |
| 运行期解析 snd YANG 源 | 运行镜像不带 snd 文件（R04 既定：零 snd 运行依赖） |
| **构建期 goyang 提取 → .gen.go** ✅ | 与 lefttreegen/tasknamegen/blacklistgen 同管线，运行期零依赖、regen-and-diff 守护 |

### D2 — rpc 并入 `/yang/schema/:module` 响应，input 复用 FieldDef

**决定**：在既有 schema 响应加一个 `rpcs` 数组（与 `fields` 平级）：`[{ name, label, path, input: FieldDef[], highRisk }]`。input 叶用**与配置同一套 FieldDef**（type/options/required/leafref/units…），前端渲染管线零改动即可复用。

**理由**：前端一次 schema 拉取即拿到 config + rpc；input 复用 FieldDef 意味着 leafref 下拉、mandatory 校验、单位后缀等既有能力**免费复用**，不为 rpc 造第二套渲染。与 schema-fixture/golden（本仓已建）天然兼容——rpc 也会被 fixture 快照钉住。

### D3 — NETCONF `<rpc>` 执行走 device-protocol 新方法，不碰读写路径

**决定**：device-protocol 新增 `ExecuteRPC(ctx, module, rpcName, inputs) (RPCResult, error)`，内部 scrapligo `Driver.RPC` 发送 `<rpc><module:rpcName>…input…</></rpc>`，解析 `<rpc-reply>`（`<ok/>` / 数据 / `<rpc-error>`）。get/edit-config/commit 路径**完全不动**。

**理由**：rpc 是独立协议动作，隔离实现避免污染对账读写。input→XML 编码复用既有 xmlcodec 思路（命名空间 = 模块 namespace）。

### D4 — rpc 执行结果不进配置缓存

**决定**：`POST /rpc/:ip/:module/:rpc` 执行后直接返回 reply，**不写 TTL+LRU 缓存、不触发对账**。

**理由**（§8）：rpc 无期望态/实际态，不是配置。清计数器、重启接口都不产生"可对账的配置差异"。硬把它塞进配置缓存会污染缓存键空间与对账语义。执行是"一次性动作 + 即时结果"。

### D5 — 所有 rpc 执行前确认；高危 rpc 升级警告

**决定**：**每个** rpc 执行前弹确认（展示 rpc 名 + 输入值 + 目标设备）——因为 rpc 皆有设备副作用（非幂等只读）。**高危子集**（名称匹配 `restart`/`reboot`/`reload`/`clear`/`reset` 等，或后续可扩为模型扩展注解）额外升级：红色警示 + 更强确认文案。

**备选与否决**：

| 方案 | 取舍 |
|------|------|
| 只高危确认、普通 rpc 直接执行 | 普通 rpc 也改设备状态（清计数不可逆），无确认易误触 |
| 全部 rpc 同级确认 | 安全但对无害 rpc 略啰嗦 |
| **全部确认 + 高危升级** ✅ | 平衡：默认有防线，高危更醒目；R08 一致 |

**高危判定策略**：首期用**名称模式启发式**（restart/reboot/reload/reset/clear/delete 前缀词），构建期打 `highRisk` 标到 rpc.gen.go。**明确记为可演进**——若华为模型有语义扩展标注高危，后续切模型驱动。名称启发式的局限（漏判/误判）在 Risks 说明。

### D6 — rpc 在模块控制台与 container 平级渲染

**决定**：模块控制台在既有 config Tab（deriveTabs 从顶层 container/list 派生）旁，增加 rpc 条目区——与 container Tab **平级**（用户指定层级）。点某 rpc → 打开其执行面板（input 表单 + 执行按钮 + 结果区）。左树/导航中模块节点展开即见 container 与 rpc 并列。

**理由**：贴合用户规格"同一 yang 模型下 container 和 rpc 同级呈现"，且复用既有模块控制台外壳，不新起页面。

### D7 — 模拟网元 custom-rpc：校验 + 记录 + 结果，供 B2 集成

**决定**：netconfsim `classifyRPC` 增 `rpcCustom`——识别非 get/edit-config 的模块 rpc，校验 input（mandatory 存在、leafref 目标存在于当前 running 树），记录 `(rpcName, inputs)` 调用，返回 `<ok/>` 或注入的结果/错误。未识别 rpc 仍返回 ok（NS 既有降级不变）。

**理由**：B2 集成测试需断言"执行 reset-if-counters-by-name(if-name=X) → sim 收到并校验 → 返回 ok"，否则执行通道无端到端验证（T02）。leafref 存在性校验让 sim 能测负路径（不存在的接口 → rpc-error）。

## Risks / Trade-offs

**[R1] 高危 rpc 误执行（restart-if 真重启接口，业务中断）**
→ 缓解：D5 全部确认 + 高危升级警示；执行面板显式展示目标设备 + 输入值。名称启发式高危判定的**漏判**风险（某高危 rpc 名不含关键词）→ 记为已知局限，首期辅以"所有 rpc 都要确认"兜底（即便漏判高危升级，仍有基础确认）。

**[R2] 名称启发式高危判定误判/漏判**
→ 接受为首期方案，design 显式标注可演进为模型扩展驱动。误判（把无害 rpc 标高危）只是多一层警示，无害；漏判由 R1 的"全部确认"兜底。

**[R3] rpc input 含 leafref，下拉需要设备实时数据（如接口名列表）**
→ 前端执行面板打开时，按 leafref path 拉取目标列表（复用既有 config 读路径/约束引擎的 leafref 解析）。设备离线时降级为手工输入 + 校验提示（R08）。

**[R4] `<rpc>` input 的 XML 编码与真机差异**
→ 模拟网元校验 + 首模块 huawei-ifm 端到端集成测试兜底；真机验证列入发布前（与既有 config 下发同口径，模拟网元绿≠真机绿的边界照旧诚实标注）。

**[R5] rpc output 结构复杂（部分 rpc 返回数据而非 ok）**
→ 首期回显 rpc-reply 原文/ok/error（Non-Goal 明确不做结构化表格）。够覆盖 reset/restart 这类"ok/error"型 rpc；返回数据型 rpc 的结构化渲染留后续。

**[Trade-off] 全部 rpc 确认对无害 rpc 略啰嗦**
→ 接受。安全优先于一次点击。高危升级让危险操作更醒目，无害操作的确认是轻量单击。

## Migration Plan

无存量迁移——纯新增能力，不改配置读写/对账路径。分阶段引入（DAG，不可颠倒）：

1. **阶段① 构建期提取 + 列 rpc**（地基）：rpcgen 工具 + rpc.gen.go + schema 响应加 rpcs + `make gen-rpc` + 漂移门禁 + B1。此时前端还不渲染 rpc，纯后端可验证（B3 断言 `/yang/schema/ifm` 含 rpc 列表）。
2. **阶段② 执行通道**：device-protocol ExecuteRPC + netconfsim custom-rpc + `POST /rpc` 端点 + B2 集成（端到端执行→sim 校验→结果）。此时可 curl 执行，前端仍未接。
3. **阶段③ 前端渲染执行**：模块控制台 rpc 区 + 执行表单 + 结果面板 + 高危确认 + F2/F3/F4。端到端打通。

每阶段一 PR（TM04 ≤1000 行）。①②纯后端不改前端；③纯前端消费②的 API。顺序不可颠倒：前端渲染依赖①的 rpc schema、执行依赖②的 API。

## Open Questions

- **rpc 执行端点路径形态**：`POST /rpc/:ip/:module/:rpc`（RESTful）vs `POST /config/:ip/rpc`（复用 config 分组）。倾向前者（rpc 与 config 语义不同，独立分组更清晰）。apply 阶段定稿。
- **rpcs 并入 /yang/schema vs 独立 /yang/rpcs/:module**：D2 倾向并入（前端一次拉取）。若 schema 响应体因此过大，改独立端点。实测 huawei-ifm rpc 体量后定。
- **高危关键词清单**：首期 restart/reboot/reload/reset/clear/delete。是否纳入更多（shutdown/format/erase）待 apply 阶段扫全模块 rpc 名后定稿。
- **leafref 下拉的数据源**：执行面板打开时实时拉 vs 复用已加载的 config 列表缓存。倾向实时拉（rpc 执行需最新接口列表），apply 定。
