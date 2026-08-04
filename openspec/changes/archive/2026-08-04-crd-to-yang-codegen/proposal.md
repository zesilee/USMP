# Proposal: crd-to-yang-codegen

## Why

USMP 北向要对接「网络自动化服务」（K8s 技术栈）：北向以 CRD 形式发布业务配置契约（BusinessVlan 等），USMP 需要把这些**北向拥有的 CRD** 转换为自己的业务网络配置 YANG 模型，才能复用既有管线（ygot 生成 Go 结构体、schema 树、前端自动渲染、意图编排下发）。当前代码生成工具只有正向 `crdgen`（YANG→CRD，服务 USMP 自有意图模型），缺反向能力；北向正式 CRD YAML 尚未到位，需要自建一个对齐华为 YANG 模型规范的示例 CRD 来验证工具与约定。

## What Changes

- 新增构建期工具 `backend/tools/crd2yang`：读 CRD manifest（apiextensions.k8s.io/v1 YAML）→ 生成 YANG 模块文件。与 `crdgen` 互为镜像：类型映射表逐条互逆，`Options`/`Generate`/golden 测试同模式。
- 定义 **CRD 携带 YANG 元信息的注解约定**（namespace/prefix/revision/task-name/顶层容器名），作为北向 CRD 的接入契约；spec 结构信息全部用 CRD 原生 OpenAPI 语法表达（type/min/max/pattern/enum/required/`x-kubernetes-list-map-keys`）。
- 自建示例 CRD `BusinessVlanNet`（工具 testdata，非 `deploy/crds/` 部署物）：字段约束直接取自已集成的 `huawei-vlan.yang`/`huawei-ifm.yang` 真实约束（vlan-id 1..4094、name 1..31 字符等），模拟北向 BusinessVlan 契约。
- 质量闭环：① 工具单测 + golden YANG；② **往返对账**——示例 CRD →`crd2yang`→ YANG →`crdgen`→ CRD′，归一化后与输入比对零失真；③ 生成的 YANG 走既有 gen.conf manifest 管线出 ygot 结构体（独立 `businessdemo` 包），测试内验证 schema 树注册（`AddYgotSchemaWithVendor`）成功；④ schemadump 纳入 demo 模块出 fixture，前端派生黄金（GD-01）覆盖，证明 R05 可自动渲染。
- Makefile 新增 `gen-crd2yang` target；CI regen-and-diff 门禁（R04 方法论）与 pre-commit 同口径覆盖新生成物。
- **明确不做（后续 change）**：示例模型不接线运行期 `load.go`（业务控制台前后端当前单 Kind 写死 `/business/vlan-services`，贸然挂菜单会出现点开即报错的页面）；意图展开/下发规则、多 Kind 业务 API 泛化，待北向正式 CRD 到位后另立 change。

## Capabilities

### New Capabilities

- `crd-to-yang-codegen`: CRD→YANG 反向代码生成能力——注解约定、类型映射（与 crdgen 互逆）、fail-fast 负路径、往返对账门禁、示例 CRD 验证链。

### Modified Capabilities

（无——`business-intent-crd` 的 BIC-01「USMP 自有意图模型以 YANG 为唯一 schema 源」不变；本能力只针对**北向拥有**的 CRD 建立反向派生，authority 边界在 design.md 阐明。`yang-codegen-pipeline` 的 gen.conf manifest 管线被复用但要求不变。）

## Impact

- 新增：`backend/tools/crd2yang/`（工具+测试+testdata）、`backend/internal/generated/businessdemo/`（gen.conf + ygot 生成物）、`backend/internal/yang/models/` 下新增生成的示例 YANG。
- 修改：`Makefile`（gen-crd2yang target）、`.github/workflows/compliance.yml` regen-and-diff 块、`.githooks/pre-commit` 同口径、`backend/tools/schemadump/main.go`（纳入 demo 包）、`backend/testdata/schema-fixtures/`（新 fixture）、前端派生黄金（新模块黄金文件）。
- 不动：运行期 `internal/yangschema/load.go`、`internal/intent/`、业务 API/前端业务控制台、`deploy/crds/`。
- 依赖：全部已在 `backend/go.mod`（apiextensions-apiserver、sigs.k8s.io/yaml、goyang），零新增第三方依赖（R10），Go 1.22 兼容。
