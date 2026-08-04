# Design: crd-to-yang-codegen

## Context

- 北向「网络自动化服务」以 CRD 发布业务配置契约（`refactor-by-crd.md` 文字描述了 5 个 CRD：NativeDeviceConfig + BusinessSwitch/BusinessVlan/BusinessInterface/BusinessRoute），正式 YAML 尚未到位。
- USMP 既有 `tools/crdgen`（YANG→CRD，357 行）：单顶层容器约定、`mapScalar` 可映射集（int*/uint*/string+pattern/bool/enum）、fail-fast 负路径、固定 status 子资源（BIC-04）、golden 测试。
- 业务网络配置既有链路：`internal/yang/models/*.yang`（唯一 schema 源）→ gen.conf manifest 管线（`make gen-yang VENDOR=<pkg>`，加包=建目录+gen.conf 零脚本改动）→ ygot 结构体 → `AddYgotSchemaWithVendor` 进 schema 树 → `/yang/schema` 前端自动渲染（R05）→ `internal/intent` 展开为华为原生配置下发。
- 硬约束：Go 1.22 钉死、R04 生成物 regen-and-diff、R10 零无关依赖、业务控制台前后端当前**单 Kind 写死**（`/business/vlan-services`、`cr.go` GVK、前端 `api/index.ts`）。
- `refactor-by-crd.md` 中与红线冲突的部分（Scrapligo/前端直连 K8s/Stack A 旧 Kind 复活）**均不在本 change 采纳范围**——本 change 只做契约转换能力。

## Goals / Non-Goals

**Goals:**

1. `backend/tools/crd2yang`：CRD manifest → 合法 YANG 模块文件，类型映射与 crdgen 逐条互逆。
2. 北向 CRD 接入注解约定（元信息契约），北向正式 YAML 到位后**换输入文件即可，工具零改动**。
3. 示例 CRD `BusinessVlanNet`（对齐 huawei-vlan/huawei-ifm 真实约束）打通验证链：golden → 往返对账 → ygot 生成 → schema 树注册 → 前端派生黄金。
4. R04 闭环：生成物入库 + `make gen-crd2yang` + CI/pre-commit regen-and-diff 同口径。

**Non-Goals:**

- 示例模型不进运行期 `load.go`（避免业务菜单出现无 API 支撑的死页面）。
- 不做意图展开/下发规则、不做多 Kind 业务 API 泛化、不碰 `deploy/crds/` 与 RBAC——待北向正式 CRD 到位另立 change。
- 不处理 NativeDeviceConfig（纯透传型，无结构可转 YANG，与本工具无关）。
- 不支持 CRD 多版本（versions[] 取 storage=true 那个，多 served 版本 fail-fast 提示）。

## Decisions

### D1 工具形态：镜像 crdgen，独立目录 `tools/crd2yang`

`main.go`（flag+WriteFile ~30 行）与 `crd2yang.go`（`Options`/`Generate([]byte) ([]byte, error)` 纯函数）分离，golden 在 `testdata/`。不与 crdgen 合并成双向工具——两个方向的 authority、输入形态、失败语义都不同，合并只省一个目录，代价是 flag 面互相污染。

### D2 Authority 边界：不动 BIC-01

| 契约归属 | schema 源 | 派生方向 | 工具 |
|---|---|---|---|
| USMP 自有意图模型（BusinessVlanService） | YANG | YANG→CRD | crdgen（不变） |
| **北向拥有的 CRD**（本 change） | CRD | CRD→YANG | crd2yang（新） |

同一个模型**禁止双向都跑**（会出现两个 source of truth 打架）；生成的 YANG 文件头部注释 + 模块 description 标注 `generated-from: <CRD name>`，与 crdgen 的 `usmp.io/generated-from` 注解对称，regen-and-diff 据此锚定。

### D3 元信息注解约定（北向接入契约）

CRD OpenAPI 表达不了 YANG 模块级信息，用 metadata.annotations 携带：

| 注解 | 必填 | 映射 | 缺省 |
|---|---|---|---|
| `usmp.io/yang-module` | 否 | module 名 | 由 singular 派生 `usmp-<singular>` |
| `usmp.io/yang-namespace` | 否 | namespace | `urn:usmp:yang:<module>` |
| `usmp.io/yang-prefix` | 否 | prefix | module 名首字母缩写规则 |
| `usmp.io/yang-revision` | 是 | revision 日期 | 无（fail-fast——revision 决定契约版本，不允许隐式） |
| `usmp.io/yang-container` | 否 | 顶层容器名 | singular 名 |
| `usmp.io/task-name` | 否 | task-name 扩展值 | `business-network` |

选注解不选 flag：注解随 CRD 文件走（北向一份文件自描述），flag 会把契约信息散落在 Makefile 里。

### D4 结构映射（与 crdgen `mapEntry`/`mapScalar` 互逆）

只取 `.spec` 子树（`.status` 整体丢弃——BIC-04 status 是 USMP 侧固定结构，反向无意义）；`.spec` object 包成单顶层容器（D3 容器名），闭合 crdgen 的单容器约定。

| OpenAPI 输入 | YANG 输出 | 备注 |
|---|---|---|
| `object` | container | |
| `array` + items object + `x-kubernetes-list-map-keys` | list，key=map-keys | 无 map-keys 的 object 数组 **fail-fast**（YANG config list 必须有 key） |
| `array` + items scalar | leaf-list | |
| `integer` + min/max | min≥0→最小能容纳 max 的 uintN，否则最小 intN；带 `range "min..max"` | 确定性规则，保证往返幂等（uint16 1..4094 → integer[1,4094] → uint16 1..4094） |
| `integer` 无 min/max | int64 | 与 crdgen 无 Range 不出 min/max 对称 |
| `string` + pattern / allOf patterns | string + pattern（多条对应多 pattern） | |
| `string` + enum | enumeration | |
| `boolean` | boolean | |
| `required` 含某属性 | 该 leaf `mandatory true`；list item 的 required 中**属于 key 的不再重复出 mandatory**（crdgen 会把 key 并进 required，反向要剥离） |
| `description` | description 语句 | |
| 其余（oneOf/anyOf/additionalProperties/format 语义类型/nullable…） | **fail-fast**，报字段 JSON 路径 | 与 crdgen 负路径哲学一致，禁止静默降级 |

### D5 往返对账 = 最强正确性门禁

测试内闭环：示例 CRD →`crd2yang.Generate`→ YANG（写临时目录）→`crdgen.Generate`→ CRD′。比对**归一化后的 spec schema**（丢 `x-kubernetes-*`、把 list key 并回 required、忽略 printer columns/status/metadata 差异）。crdgen 作为库函数被测试 import——两工具同在 `tools/` 下、包名均为 main 不能互相 import，故把 crdgen 核心提为可 import 的内部包**不做**（避免范围蔓延）；改为往返测试放在 crd2yang 包内，用 `go run ../crdgen` 子进程或直接复制断言所需的归一化逻辑。**拍板：测试内用 `os/exec` 跑 `go run ../crdgen`**（B2 集成测同款思路，零重构，`testing.Short()` 跳过）。

### D6 示例 CRD `BusinessVlanNet` 的华为规范对齐

testdata 内容（不进 `deploy/crds/`）。字段与约束逐条取自已集成模型：

- `vlan-id`: integer 1..4094（huawei-vlan `vlan-id` typedef）
- `name`: string pattern 长度 1..31（huawei-vlan `name` length "1..31"）
- `description`: string 1..80（huawei-vlan）
- `devices[]`: map-keys=[ip]，ip 带 IPv4 pattern；`access-ports[]`/`trunk-ports[]` leaf-list（接口名，对齐 huawei-ifm 接口命名域）
- 覆盖映射表全部分支：uint16/string+pattern/enum（如 `mode: access|trunk`）/boolean/list/leaf-list/嵌套 container/mandatory

Kind 取 `BusinessVlanNet`、模块名 `usmp-business-vlan-net`——与现役 `BusinessVlanService`/`usmp-business-vlan` 明确区隔，规避 SC-01 旧 Kind 名（BusinessVlan 是被物理删除的 Stack A Kind，禁复活）。

### D7 生成 YANG 的落点与管线接入

- 生成物：`backend/internal/yang/models/usmp-business-vlan-net.yang`（与手写意图模型同目录，头注释标 generated）。
- ygot：新建 `backend/internal/generated/businessdemo/gen.conf`（manifest 管线自动消费，零脚本改动）→ `make gen-yang VENDOR=businessdemo`。**独立包不并入 `business` 包**：demo 生命周期独立（北向正式 CRD 到位后可整目录替换/删除），并入会让退役变成外科手术。
- schema 树：测试内 `AddYgotSchemaWithVendor(ds, businessdemoSchema, "usmp")` 断言注册成功、树形完整——不改运行期 `load.go`。
- 前端渲染证明：`tools/schemadump/main.go` 追加加载 businessdemo 包 → `make gen-schema-fixtures` 出 `business-vlan-net.json`（模块名=顶层容器名，AddYgotSchemaWithVendor 命名口径） fixture → 前端派生黄金套件（GD-01）自动覆盖新模块，`deriveTabs/deriveColumns/deriveKeyField` 黄金入库。schemadump 是构建期工具，加载 demo 包不影响运行期 `/yang/modules`。

### D8 Makefile / 门禁

- `make gen-crd2yang`：跑工具重生成 `usmp-business-vlan-net.yang`。
- `compliance.yml` regen-and-diff 块追加 `make gen-crd2yang`（在 gen-yang 之前——YANG 是下游 ygot 的输入）；`.githooks/pre-commit` 同口径（CG-03）。
- 生成的 `.yang` / fixture / 前端黄金若撑体积：`.yang` 预计 <100 行，无需动 pr-size 豁免；如超再按两处同口径（pr-size.yml + commit-msg）加排除。

## Risks / Trade-offs

- [北向正式 CRD 可能用注解约定之外的表达（如 CEL validation、defaults）] → 注解约定文档化在 spec；工具对未知构造 fail-fast 而非静默丢弃，接入时报错清单即改造清单。
- [integer 宽度推导有损（北向 int32 但 range 落在 uint16 内会被转成 uint16）] → 往返对账只保证 schema 语义等价（min/max 不变），Go 侧宽度差异对 RFC7951 JSON 载荷无影响；spec 中明示此规则。
- [`go run ../crdgen` 子进程测试依赖构建环境] → `testing.Short()` 跳过，CI 全量跑；与 B2 集成测同风险面，无新增。
- [demo 模型长期滞留变僵尸] → 独立 businessdemo 包 + testdata 隔离，北向正式接入 change 的 tasks 里显式含「退役 demo」项；本 change 的 spec 标注 demo 为 fixture 性质。
- [schemadump 加载 demo 包后 fixture 集合语义从「运行期全集」变为「运行期全集+demo」] → fixture 头部/README 注明；前端黄金本就按 fixture 文件驱动，无运行期影响。

## Open Questions

（无——北向正式 CRD YAML 到位时间不影响本 change 交付；接入时另立 change 处理运行期接线与多 Kind 泛化。）
