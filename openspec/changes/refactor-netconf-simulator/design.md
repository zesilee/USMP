# 重构 NETCONF 模拟网元 — design

> change：`refactor-netconf-simulator` | 依赖：`proposal.md`

## 1. 目标架构

```
cmd/netconf-simulator/main.go        # 新增：独立可部署二进制（对齐 deploy/manifests/netconf-simulator）
backend/simulator/netconfsim/
  ├── transport.go   SSH 传输：hello/capability 协商、EOM(1.0)+chunked(1.1) framing、会话循环
  ├── rpc.go         encoding/xml 解码 RPC（get-config/edit-config/commit/discard-changes/lock…）→ 结构体分发
  ├── datastore.go   ★结构化数据存：running/candidate 均为通用 XML 数据树（*dataNode）
  ├── editconfig.go  edit-config 语义：merge/replace/create/delete（按 operation 属性）
  ├── filter.go      get-config subtree filter（按请求裁剪树后 Marshal）
  ├── scenario.go    故障注入（ErrorOnRPC / RejectAuth），去 testing 依赖
  └── testsupport/   Assert* 助手（仅此子包 import testing/testify）
backend/simulator/netsim/            # 删除（迁移末期）
```

## 2. 核心设计决策

### D1 结构化 datastore（最关键）—— ⚠️ 已由 ygot 树修订为通用 XML 数据树

**实现期发现（推翻原 ygot 方案）**：核实代码后确认三点使"ygot `*Device` 树"不可行——
1. 存在**两个独立 ygot 根**（`huawei.Device{Ifm,System,Vlan}` 与 `openconfig.Device{Interfaces,Vlans}`），类型/ schema 各异，单棵树装不下；
2. ygot `Unmarshal` 只吃 **RFC7951 JSON**，不解析 XML；生成结构体**零 `xml` tag**（仅 `path:`/`module:`）；
3. 客户端代码自注：ygot 结构体含 map，`xml.Marshal` 处理不了，故一直手搓 XML。
即 NETCONF(XML) 与 ygot(JSON) 之间无现成桥，"edit-config 走 ygot.Unmarshal"无法落地——这正是当年 1068 行 string-parse 的根因。

**修订方案（Option C，通用 XML 数据树，模型无关）**：
- `Datastore` 持 `running *dataNode` / `candidate *dataNode`——用 std `encoding/xml` 自建的**通用 XML 节点树**（不引入新依赖，R10）。设备本就是"存数据树"而非"存厂商结构体"，故此设计更贴近真实网元，且 huawei/openconfig **统一处理**。
  - `edit-config`（T3）→ 解析 config XML 为子树 → 按 `operation` 在树上 merge/replace/create/delete（元素名 + list 键启发式匹配）。
  - `commit` → candidate 树深拷贝覆盖 running 树。
  - `get-config`（T4）→ 对 running 树应用 subtree filter → 序列化回 XML。
  - 断言：`testsupport` 改为按路径查通用树（如 `ds.Running().find("vlan/vlans/vlan[id=100]")`），逐步删除 `ExtractVLANs`/`ExtractHuawei*`/`*TestData`。
- **收益**：消除 string-parsing、双 XML 形态兼容与 `cleanNamespaces`/`normalizeOpenConfigXML`/`fixXMLTagNames` 手术；获得真实 merge/delete/filter 语义；模型无关。
- **对 R04 的说明**：R04 约束的是**生产配置模型**用 ygot 生成，不管测试替身的内部存储；模拟器用通用 XML 树不违反 R04。
- **权衡**：无 schema 时 list 键匹配为**启发式**（元素名 + 键子节点），对测试替身足够；如需 schema 精确匹配可后续增强。
- **风险**：序列化需与客户端产出的 XML 往返等价——T2 先建对拍测试锁定（见 §4 步骤 2/3）。

### D2 edit-config 语义
- 解析 `<config>` 下每个节点的 `operation`（`nc:operation="merge|replace|create|delete|remove"`，默认 merge）。
- merge：递归并入；replace：替换子树；delete/remove：从 candidate 树删除对应 keyed list 项/容器。
- 覆盖平台实际用法：reconciler 的 `client.Change{Add/Delete/Modify}` → edit-config，对应 create/delete/merge。

### D3 真 XML 解析
- 定义 `type rpc struct { XMLName; MessageID; GetConfig *getConfig; EditConfig *editConfig; Commit *struct{}; ... }`，`xml.Unmarshal(msg, &rpc)` 后按非 nil 字段分发，替换 `strings.Contains`（`server.go:161`）与 `strings.Index` 提取（`server.go:279-315`）。

### D4 capability 与 framing
- hello 广告：`base:1.0`、`base:1.1`、`:candidate`、`:writable-running`、`:startup?`（按需）。
- framing：保留 `]]>]]>`(1.0)，若客户端 hello 含 1.1 则切 chunked framing（`\n#<len>\n…\n##\n`）。scrapligo 默认行为需实测（§4 步骤 4，若客户端只用 1.0 则此步可降级为"仅广告不实现 1.1"）。

### D5 测试/生产解耦
- `netconfsim` core 包移除 `import "testing"`/testify。
- `Assert*` 迁到 `netconfsim/testsupport`（该子包可依赖 testing）。集成测试改 `testsupport.AssertInterfaceExists(t, ds, name)`。
- 新增 `cmd/netconf-simulator/main.go`：flag 配端口/初始配置文件，`sim.Start()` 阻塞运行 → 可容器化。

### D6 合一与 test-server
- 删 `netsim`。`cmd/test-server` 两选一（提案推荐后者）：
  - (a) 显式内存 REST 桩（诚实命名，不再冒充 NETCONF）；或
  - (b) 复用 `netconfsim` + 真 NETCONF 客户端，E2E 全链路真实。
- 决策留待 apply 首步确认（见 tasks T0）。

## 3. 依赖与接口

- ygot 生成模型：`internal/generated/{openconfig,huawei}`（R04，已有）。
- 被测客户端：`pkg/yang-runtime/client`（NETCONF）——重构须保持其 Get/Set 契约不变，模拟器仅提高对端保真度。
- 消费测试：改断言调用点，不改测试意图。

## 4. 渐进迁移策略（遵循 §5.3 / .openspec.yaml legacyMigration）

> 禁止一次性重写。旧 XML datastore 与新树 datastore **并行**，用同一批集成测试对拍，绿灯后再切换、最后删除。

1. **保留旧代码**：不动现有 netconfsim/netsim，先补测试脚手架。
2. **新旧并行**：新增结构化 datastore + 新 server 走 build tag 或独立类型，与旧实现并存。
3. **双路径验证**：现有 `*_integration_test.go` 分别对旧/新实现各跑一遍（表驱动 fixture），断言等价。
4. **切换入口**：集成测试与 test-server 指向新实现；补 capability/filter/edit-config 语义测试。
5. **删除旧代码**：移除旧 XML datastore、`Extract*`、`*TestData`、`netsim`、旧 server 分支。

## 5. 验收标准

- 集成测试全绿；新增测试覆盖：edit-config merge/delete、get-config subtree filter、capability 广告。
- `netconfsim` core 包 `go list` 无 `testing` 依赖；`go build ./cmd/netconf-simulator` 产出可执行文件。
- `backend/simulator/netsim` 已删除，无残留引用。
- `datastore.go` 行数与 string-parsing 显著下降（目标：消除三套 `Extract*` 与 `*TestData`）。
- 满足 R02/R04/R06/§5.3；PR 分批 ≤500 行/commit、≤800 行(或 >20 文件时 ≤3000)。
