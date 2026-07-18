## Why

USMP 定位为华为交换机/路由器管理平台，配置链路（Stack B：`/yang/schema` + `/config` → NETCONF）全部走华为原生 YANG 模型（huawei-vlan/ifm/system/bgp/ni/…）。openconfig 模型支持是早期探索遗留：`generated/openconfig` 生成包、未注册的 `controller/interfaces` Reconciler、`netconf.go` 中 openconfig 类型特判分支均无生产流量，却持续产生维护成本（schema 树噪音、测试载体、regen 管线多一个厂商包）。用户已拍板：**只保留 Huawei 的 YANG 模型，不需要支持 openconfig 和 ietf**。

ietf 侧无独立"支持"可删：代码中 `ietf` 字样均为 NETCONF 协议命名空间（`urn:ietf:params:xml:ns:netconf:*`，协议本体必须保留）或 yang-models submodule 中华为模型的构建期 import 依赖（`ietf-inet-types` 等，删除即破坏华为模型 codegen）。本变更将此边界固化进 spec。

## What Changes

- **BREAKING** 删除 `backend/internal/generated/openconfig/` 生成包（含 gen.conf，`make gen-yang` 自动少一个厂商包）
- **BREAKING** `yangschema.Load()` 不再加载 openconfig schema——`/yang/modules` 不再暴露 `interfaces`/`vlans`（openconfig）模块，schema 树仅含 huawei + usmp business
- 删除 `backend/internal/controller/interfaces/`（openconfig 接口 Reconciler，main.go 从未注册，死代码）
- 删除 `client/netconf.go` 中 openconfig 遗留降级分支（`OpenconfigVlan_Vlans`/`OpenconfigInterfaces_Interfaces` 特判、`buildOpenConfigInterfacesXML`、`OpenConfigInterfacesNS`）；降级链收敛为：注册表命中 → `xml.Marshal` 兜底（R08 不变）
- 删除 `backend/internal/yang/models/` 下 4 个 openconfig `.yang` 源文件（`openconfig-vlan/interfaces/types/extensions`）；保留同目录 `usmp-business-vlan.yang`（business gen.conf 仍消费该目录）
- 删除遗留文档 `spec/openconfig-vlan-controller.md`；清理代码中残留的 openconfig 注释（如 vlan reconciler 注释）
- **不动**：NETCONF 协议命名空间中的 `ietf` 字样；yang-models submodule 中 `ietf-*.yang`/`openconfig-*.yang` 文件（华为模型构建期依赖，submodule 为上游镜像不做裁剪）；`github.com/openconfig/ygot|goyang` Go 库依赖（R04 codegen 工具链本体）

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `yang-xml-codec`: 注册表未命中的降级链从「openconfig 遗留分支 → xml.Marshal 兜底」收敛为「xml.Marshal 兜底」；删除 openconfig 遗留分支的保留条款
- `yang-api`: 明确 `/yang/modules` 仅暴露 huawei 厂商模块与 usmp 业务意图模块（openconfig 不再出现），厂商边界固化为 spec 条款

## Impact

- 后端：`internal/generated/openconfig/`（删）、`internal/controller/interfaces/`（删）、`internal/yangschema/load.go`、`pkg/yang-runtime/client/netconf.go` 及其用 openconfig 类型做载体的测试（`netconf_registry_dispatch_test.go`、`netconf_delete_marshal_test.go` 等，改用 huawei 类型）
- 构建管线：`backend/internal/yang/models/` 只剩 business 模型；`scripts/gen-yang.sh` 零改动（目录驱动）
- 前端：无生产代码引用 openconfig；`menu.native.test.ts`/`Sidebar.test.ts` 用 `vendor:'openconfig'` 作合成测试数据，测的是通用厂商分组逻辑，酌情改标签不改行为
- 覆盖率棘轮（T08）：删除死代码可能改变覆盖率基线，完成后校验 `backend/.coverage-baseline`
- 文档：`spec/openconfig-vlan-controller.md`（删）、各 spec design.md 中历史性 openconfig 提法不动（历史记录）
