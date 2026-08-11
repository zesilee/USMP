# Design: retire-ygot-runtime

## Context

发布二进制对 ygot/goyang 的运行时依赖已收敛为三块（评估结论，2026-08-11）：

1. **生成代码**（`internal/generated/huawei` 约 25 万行 + `business`/`businessdemo`）：实现 `ygot.GoStruct`/`ValidatedGoStruct` 接口，含 1.3 万枚举、234 个 union、3 万行 gzip schema。
2. **ygot 运行库四能力**：`ytypes.Unmarshal`（RFC7951 JSON→结构体，写路径）、`ygot.EmitJSON`（读路径）、生成的 `Validate()`（仅 `intent/cr.go`）、`GzipToSchema`（启动期 goyang Entry 反序列化）。
3. **胶水引用**（约 15 个非测试文件）：`driver.Descriptor` 签名、xmlcodec 的 `GoStruct`/`EnumName`/`KeyHelperGoStruct` 触点、`ygot.String()` 等指针 helper。

**已经与 ygot 无关的部分（不动）**：diff 引擎（自研反射，只认 map-list/`...Key` 约定）、xmlcodec 主体（只认 `path`/`module` tag + schema）、netconfcore、前端全链路（消费框架内部 Schema 模型，启动期由 `schema/entry.go` 一次性从 goyang Entry 转换而来）。运行时不解析 YANG 源文件（goyang parser 仅存活于 tools 与测试）。

约束：Go 1.22 钉死；R04（禁止手写 YANG 结构体——语义更新为「自研生成器生成」）；R08/R09；对外 HTTP API 与 NETCONF XML 行为不可变；构建期工具（`backend/tools/*`）允许继续使用 goyang。

## Goals / Non-Goals

**Goals:**

- `usmp-backend` 发布二进制 import 闭包零 `openconfig/ygot`、零 `openconfig/goyang`，守护测试拦截回流。
- 生成代码约定与 ygot 现状字节级等价（字段名/`path`+`module` tag/list=map[key]*T/`...Key` 字段/枚举类型名），使 diff/xmlcodec/drivers/intent 免改或最小改。
- 对外行为不变：HTTP API（RFC7951 形状）、NETCONF XML、`/yang/schema` 契约、前端零感知。
- 解除 ygot v0.29.20 依赖天花板债。

**Non-Goals:**

- 不引入 k8s.io/apimachinery 承载 YANG 类型（GVK/DeepCopyObject/conversion 语义错配；只借鉴其范式）。
- 不做多版本模型转换（运行时单一厂商模型版本）。
- 不改 diff 引擎算法、xmlcodec XML 行为、netconfcore、前端。
- 不在本 change 内决定 tools 是否拆独立 go module（收尾阶段单独拍板）。
- tools 与测试继续使用 goyang（不在退役范围）。

## Decisions

### D1: K8s runtime.Object 范式——极小标记接口 + Scheme 注册表

自研 `Object` 标记接口替代 `ygot.GoStruct`（配套 `KeyedObject` 替 `KeyHelperGoStruct`、`Enum` 替 `GoEnum`），类型保持「哑」，聪明逻辑在类型外。driver 注册表按 Scheme 范式收口类型元数据（现 `Descriptor` 已是八成 Scheme：`NewStruct`=构造、`Unmarshal`=codec、`EncodeAnchor`=元数据），注册体验「一条描述符」不变。

*备选*：直接复用 apimachinery Scheme——否决，GVK 语义错配且让南向模型耦合 K8s 机器（同「CRD 当载体不当架构通道」拍板精神）。

### D2: RFC7951 编解码 = 构建期生成 per-type JSON 方法，非运行时通用反射引擎

生成器为每个容器/枚举/union 生成 `MarshalJSON`/`UnmarshalJSON`，`encoding/json` 驱动递归；RFC7951 怪规则（模块限定 key、(u)int64 字符串化、list map↔数组、枚举名映射、union 按类型试探）烘焙进每个类型的生成代码。错误局部化、可逐类型 golden 对拍。

*备选*：手写通用反射引擎（对标 ytypes）——否决，边角行为兼容风险集中在一个递归引擎深处，排障与对拍成本高。

### D3: Schema IR 构建期直出，替换 gzip-goyang-Entry 链路

新建构建期工具：goyang 解析 YANG 源 → 直接序列化为框架内部 Schema 模型（`pkg/yang-runtime/schema` 现有 Node 模型）的自有格式（gzip JSON，自定 schema 版本号字段防漂移）→ 生成物入库 → 运行期加载自有格式。现 `schema/entry.go` 的 Entry→内部模型转换逻辑**迁移进构建期工具**（逻辑复用，不重写）；`schema/loader.go` 一并迁出运行时包。

收益：消除「扩展语句是否存活 ygot gzip 往返」这一已踩过坑的隐患（构建期直读 goyang 全量信息）；xmlcodec 所需的 namespace/key 元数据一并进 IR（替代现在从 `huawei.SchemaTree` 取 `yang.Entry` 的用法）。

*备选*：运行期解析自有格式回 goyang Entry 形状——否决，goyang 类型本身就是要退役的依赖。

### D4: 生成约定字节级冻结 + 三通道 golden 对拍

自研生成器输出与 ygot 现产物在「结构约定」上字节级等价（字段名、tag、map-list、`...Key`、枚举常量名）。迁移期间新旧双路径并行，三通道 golden 对拍：

- **JSON 通道**：同一 RFC7951 输入 → 旧 `ytypes.Unmarshal`+`EmitJSON` vs 新生成方法，输出逐字节比对；
- **XML 通道**：同一配置树 → xmlcodec 新旧类型输出比对；
- **Schema 通道**：`/yang/schema` 全模块 JSON 新旧比对（复用 68 模块派生黄金地基）。

对拍测试属迁移期脚手架，切换完成后随旧链路一起删除（golden 文件保留为回归基线）。

### D5: 服务端校验子集从内部 Schema IR 驱动

替代生成的 `Validate()`（现仅 `intent/cr.go` 使用）：基于 IR 中已有的 pattern/range/length/enum 元数据（前端表单同源）实现校验器。mandatory/when/must 不做运行时求值（现状 ygot Validate 也不做 must/when，设备侧兜底），范围明确为「现状等价」。

### D6: 守护测试禁回引（仿 NC-01）

新增守护测试断言 `usmp-backend` import 闭包（`go list -deps`）不含 `openconfig/ygot`、`openconfig/goyang`；tools/测试/simulator 豁免。go.mod 中两库保留为构建期依赖，验收口径=「发布二进制不链接」（`go version -m` 可验证）。

## Risks / Trade-offs

- [RFC7951 边角兼容：union 234 个、枚举 1.3 万、(u)int64 字符串化、模块前缀归一化] → D2 生成式方法 + D4 三通道 golden 全量对拍；对拍以现有 68 模块 fixture 全集 + 手工构造负路径样本喂入。
- [生成器本身的正确性（25 万行生成物）] → regen-and-diff 门禁保留：首次切换 PR 的 diff 即「新旧生成物全量对比」，约定漂移在 review 期显形；生成器自身按 B1 表格驱动测试覆盖命名/tag/枚举/union 分支。
- [xmlcodec 从 `yang.Entry` 取 namespace/key 的隐式依赖] → D3 把该元数据显式建进 IR，xmlcodec 改读 IR；对拍 XML 通道兜底。
- [迁移期双链路并存增加二进制体积与认知负担] → 每步「并行→对拍绿→切换→删旧」在同一 change 内闭环，不留长期双轨。
- [`intent/cr.go` 校验行为差异（D5 子集）] → 先写现状行为快照测试（红/绿基线）再切换；business 模型小（288 行生成物），风险可控。
- [工作量 6–9 周，跨多 PR] → 7 阶段每阶段独立可合入、可回退（见 tasks.md）；TM03 约束下与其他 worktree 不并行改 `internal/generated`/`pkg/yang-runtime`。

## Migration Plan

7 阶段渐进（每阶段=1 个或多个 ≤1000 行 PR，全程主线可发布）：

1. **Schema IR 自立**（D3）：构建期工具 + IR 格式 + 运行期加载器并行上线，`yangschema.Load()` 切换数据源，对拍 schema 通道 → 删 `entry.go`/`loader.go` 运行时路径。
2. **自研生成器**（D1/D2）：tools 内落地，产出 `internal/generated/native/*`（与旧生成物并存），生成器 B1 测试 + 结构约定 diff 对拍。
3. **JSON 通道切换**：driver 注册表/config 编解码切到新类型 + 生成方法，JSON golden 对拍绿后删旧调用。
4. **XML 通道切换**：xmlcodec 触点换自研接口 + IR 元数据，XML golden 对拍。
5. **校验切换**（D5）：IR 校验器替 `Validate()`。
6. **胶水清扫**：intent/api/simulator 引用清零，删旧生成物 `internal/generated/{huawei,business,businessdemo}` 旧版。
7. **收尾**：守护测试（D6）+ R04 文案更新 + go.mod 整理 + 覆盖率棘轮上调。

回退策略:每阶段切换点均为独立 commit，旧链路删除前保留完整,revert 单个 PR 即可回退。

## Open Questions

- tools 是否拆独立 go module 以让主 go.mod 彻底移除两库（vs 接受「二进制不链接」口径）——阶段 7 拍板，需用户确认验收口径。
- `businessdemo` 生成包（北向 demo 隔离锚点）是否随迁或直接删除——阶段 6 视 demo 存续状态定。
