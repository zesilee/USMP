# Proposal: retire-ygot-runtime

## Why

商用发布二进制（`usmp-backend`）目前链接 openconfig/ygot 与 openconfig/goyang 两个第三方运行库：25 万行生成代码实现 ygot 接口，写路径 JSON 解码（`ytypes.Unmarshal`）、读路径 JSON 编码（`ygot.EmitJSON`）、意图校验（生成的 `Validate()`）、启动期 schema 解压（`GzipToSchema` + goyang Entry 反序列化）四条运行时能力依赖它们。商用交付要求发布二进制完全自主可控（无外部 YANG 运行库依赖），且 ygot v0.29.20 是 Go 1.22 钉死下的依赖天花板之一，退役后一并解债。

参考 K8s apimachinery 的 runtime.Object/Scheme 范式（极小标记接口 + 中央类型注册表 + 构建期生成样板代码），但**不**引入 apimachinery 承载 YANG 类型（GVK/conversion 语义错配；构建期工具继续允许使用 goyang）。

## What Changes

- **新增自研构建期代码生成器**（tools，允许用 goyang）：从 YANG 源生成结构体/枚举/union（保持与 ygot 现约定字节级等价：字段名、`path:"..."`+`module:"..."` tag、list=map、`...Key` 字段），并为每个容器/枚举/union 生成 `MarshalJSON`/`UnmarshalJSON` 方法（RFC7951 规则烘焙进生成代码，运行时零通用反射引擎）。
- **新增自研运行库** `yang-native-runtime`：极小 `Object` 标记接口（替 `ygot.GoStruct`/`KeyHelperGoStruct`/`GoEnum`）、Scheme 式类型注册、指针 helper、基于内部 Schema IR 的服务端校验子集（pattern/range/length/enum）。
- **schema 通道自立**：构建期直接把 goyang 树序列化为框架内部 Schema 模型自有格式，运行期加载自有格式，替换「ygot gzip JSON → goyang Entry → entry.go 转换」链路；`schema/loader.go`（运行时 goyang 解析器，仅测试在用）迁出运行时包。
- **胶水置换**：driver 注册表签名、xmlcodec 触点（`GoStruct`/`EnumName`/`KeyHelperGoStruct`）、intent/api 层引用切换到自研接口。
- **BREAKING（内部契约）**：`driver.Descriptor`、`xmlcodec`、`schema.AddYgotSchema` 等内部 API 签名从 ygot 类型换为自研类型；对外 HTTP API 与 NETCONF 行为不变（golden 双路径对拍保证）。
- **守护测试**：仿 NC-01，禁止 ygot/goyang 回流到发布二进制 import 闭包（tools/测试豁免）。
- 全程渐进迁移：旧链路保留 + 新链路并行 + 双路径对拍 → 切换 → 删除旧链路。

## Capabilities

### New Capabilities

- `yang-native-runtime`: 自研 YANG 类型系统运行库——Object 标记接口、Scheme 类型注册、生成式 RFC7951 JSON 编解码契约、内部 Schema IR 构建期序列化/运行期加载、服务端校验子集、发布二进制零 ygot/goyang 依赖门禁。

### Modified Capabilities

- `yang-codegen-pipeline`: 生成器从 openconfig/ygot 切换为自研生成器；输出约定不变（regen-and-diff 门禁保留），新增每类型 JSON 方法与 Schema IR 产物。
- `device-driver-registry`: 描述符契约中 `ygot.GoStruct`/`ytypes.UnmarshalOpt` 替换为自研 `Object`/编解码签名；注册体验（一条描述符）不变。
- `yang-xml-codec`: 编解码引擎输入类型与枚举转名从 ygot 接口切换为自研接口；XML 行为字节级不变。
- `system-architecture`: R04 表述由「ygot 自动生成」改为「自研生成器自动生成（构建期工具可用 goyang）」；新增发布二进制禁引 ygot/goyang 红线。

## Impact

- **代码**：`internal/generated/*`（重新生成，约 25 万行）、`tools/`（新增生成器）、`pkg/yang-runtime/{schema,driver,xmlcodec,diff}`、`internal/{drivers,api,intent,yangschema}`。
- **不受影响**：diff 引擎与 xmlcodec 主体逻辑（只认 `path`/`module` tag，约定保持）、netconfcore、前端全链路（消费内部 Schema 模型，`/yang/schema` 契约不变）、CRD/apimachinery 北向依赖（继续用于元信息存储）。
- **依赖**：发布二进制 import 闭包移除 ygot/goyang；go.mod 中两库降级为 tools/测试专用（工具是否拆独立 module 在收尾阶段拍板）。
- **风险**：RFC7951 边角行为兼容（union/64 位整数字符串化/模块前缀/list map↔数组）——以生成式 per-type 方法 + golden 双路径对拍收敛；ygot gzip 往返的扩展存活行为差异——schema IR 构建期直出反而消除此隐患。
- **测试**：全量 B1/B2/B3 + golden 等价对拍（JSON/XML/schema 三通道）+ e2e smoke；覆盖率棘轮不下降。
