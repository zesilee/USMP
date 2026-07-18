## Context

Stack B 配置链路（`/yang/schema` + `/config` → diff → NETCONF edit-config）自 VLAN/IFM 交付起全部走华为原生模型，openconfig 路径无生产流量：

- `backend/internal/generated/openconfig/`（all.gen.go 907 行 + gen.conf，模块 `openconfig-vlan`/`openconfig-interfaces`）仅被三处引用：`yangschema/load.go`、`client/netconf.go` 遗留分支、`controller/interfaces/`。
- `controller/interfaces/` 是 openconfig 接口 Reconciler，`main.go` 从未注册（华为接口走 `controller/ifm`），除自身测试外零引用——死代码。
- `client/netconf.go` `marshalChange` 降级链：注册表命中（XC-04 主路径）→ `OpenconfigVlan_Vlans` 特判 → `OpenconfigInterfaces_Interfaces` 特判（`buildOpenConfigInterfacesXML`）→ 通用 `xml.Marshal` 兜底 → map 反射兜底。`yang-xml-codec` spec 现行条款明文要求保留 openconfig 遗留分支。
- `backend/internal/yang/models/` 同目录混放 4 个 openconfig `.yang` 与 `usmp-business-vlan.yang`（openconfig 与 business 两份 gen.conf 共享 `yang_path`）。

ietf 现状：代码中 `ietf` 均为 NETCONF 协议命名空间（`urn:ietf:params:xml:ns:netconf:*`、`iana-if-type`）或 yang-models submodule 内华为模型的 import 依赖，**不存在可删除的"ietf 模型支持"**。

## Goals / Non-Goals

**Goals:**
- 运行期 schema 树与 `/yang/modules` 仅含 huawei + usmp business 模块
- `generated/openconfig` 包、`controller/interfaces`、netconf.go openconfig 特判分支物理删除
- `marshalChange` 降级链收敛为：注册表 → `xml.Marshal` 兜底 → map 反射兜底（R08 不变）
- spec 契约同步：`yang-xml-codec` 删 openconfig 保留条款，`yang-api` 固化厂商边界

**Non-Goals:**
- 不裁剪 yang-models submodule（上游镜像，ietf/openconfig `.yang` 是华为模型构建期编译依赖）
- 不移除 `github.com/openconfig/ygot|goyang` Go 依赖（R04 codegen 工具链本体，与"openconfig 模型支持"无关）
- 不改 NETCONF 协议层任何 `urn:ietf:*` 命名空间
- 不动通用 `xml.Marshal`/map 反射兜底行为（含其 camelCase→kebab-case replacer），仅删 openconfig 类型特判

## Decisions

**D1：物理删除而非 deprecated 标记。** 先例 retire-idle-scaffolds（PR#199-#201）已确立"空转载体物理删除、按需重建"的模式；openconfig 路径零生产引用，保留只增加 regen 管线与测试载体成本。回滚 = git revert。

**D2：`netconf_registry_dispatch_test.go` / `netconf_delete_marshal_test.go` 中以 openconfig 类型为"注册表未命中"载体的用例，改用本地未注册 GoStruct 伪类型。** 备选是改用 huawei 类型，但 huawei 类型全部命中注册表，无法测未命中分支；测试自定义一个实现 `ygot.GoStruct` 的哑类型即可精确覆盖降级链。

**D3：`backend/internal/yang/models/` 保留目录与 `usmp-business-vlan.yang`，仅删 4 个 openconfig 文件。** business gen.conf 的 `yang_path` 指向该目录，改目录结构会牵连 business 生成管线与 tasknamegen `go:generate` 行，超出本次范围。

**D4：前端合成测试数据中 `vendor:'openconfig'` 改为中性标签（如 `vendor:'other'`）。** 测的是通用厂商分组逻辑而非 openconfig 本身，改标签消除字面残留、行为断言不变。

**D5：删除后跑 `make gen-yang` 全量 regen-and-diff 验证零漂移**（huawei/business 两包不受影响），并校验覆盖率棘轮（删除死代码通常抬升覆盖率，必要时按 T08 上调基线）。

## Risks / Trade-offs

- [openconfig 生成包被未察觉的隐式引用] → 编译期即暴露；`go build ./... && go test ./...` 全量验证
- [删除 `controller/interfaces` 后覆盖率基线波动] → T08：完成后重算 `backend/.coverage-baseline`，只升不降
- [`IanaIfTypeNS`/`xmlEscape` 等辅助符号被其他路径共用] → 实现时逐符号查引用，仅删真死符号
- [未来真要接 openconfig 设备] → yang-models submodule 与 gen-yang 管线（目录驱动，加 gen.conf 即扩厂商）完整保留，重建成本 = 一个目录 + 一份 gen.conf

## Migration Plan

单 PR 交付（预估 ≤1000 行净删除），worktree 隔离，顺序：spec delta（红线 R17）→ 测试改造（D2 伪类型）→ 物理删除 → regen-and-diff + 全量测试 → 覆盖率棘轮校验。无部署迁移动作（无持久化数据涉及）。

## Open Questions

（无——范围已在 explore 阶段全量摸底。）
