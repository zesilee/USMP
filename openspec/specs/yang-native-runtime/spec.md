# yang-native-runtime — 自研 YANG 类型系统运行库

## Purpose

自研 YANG 类型系统运行库与生成管线的行为契约（retire-ygot-runtime 交付）：
Object 接口族（`pkg/yang-runtime/object`）、构建期生成的 RFC7951 JSON 编解码、
Schema IR 构建期序列化/运行期加载（`tools/schemagen` 直读 YANG 源）、基于 IR 的
服务端校验子集（`pkg/yang-runtime/validate`），以及发布二进制零 ygot/goyang 的
守护门禁（含 go.mod 层——tools 为独立 go module）。

## Requirements

### Requirement: YN-01 Object 标记接口与类型约定冻结

系统 SHALL 提供自研 YANG 类型系统运行库（`pkg/yang-runtime/object`，命名以实现为准）：极小 `Object` 标记接口替代 `ygot.GoStruct`，配套 `KeyedObject`（替 `ygot.KeyHelperGoStruct`，提供 list key 元数据）与 `Enum`（替 `ygot.GoEnum`，提供枚举值域名映射）接口，以及指针 helper（`String()`/`Uint16()` 等）。接口 SHALL 保持「哑类型」原则（参考 K8s runtime.Object 范式）：类型不承载编解码/校验逻辑之外的行为。生成结构体的结构约定 SHALL 与既有 ygot 约定字节级等价并冻结：字段名、`path:"..."`+`module:"..."` struct tag、YANG list 表示为 `map[K]*T`、复合键 `...Key` struct、枚举常量命名——使 diff 引擎、xmlcodec 主体、驱动描述符消费方免改。

#### Scenario: 标记接口替换后既有反射消费方免改
- **WHEN** diff 引擎与 xmlcodec 以新生成结构体（实现 `Object`）作为输入运行既有 B1 套件
- **THEN** 行为 SHALL 与 ygot 结构体输入完全一致（tag/map-list/`...Key` 约定未变）

#### Scenario: key 元数据经 KeyedObject 获取
- **WHEN** 对 list 条目类型调用 key 元数据接口
- **THEN** SHALL 返回与 ygot `ΛListKeyMap()` 等价的 key 名→值映射（单键与复合键一致适用）

#### Scenario: 枚举值域名映射
- **WHEN** 对枚举值查询值域名（或以值域名反查）
- **THEN** SHALL 返回与 ygot `ΛMap`/`EnumName` 等价的映射结果，未映射值返回明确未命中

### Requirement: YN-02 生成式 RFC7951 JSON 编解码

RFC7951 JSON 编解码 SHALL 由构建期生成的 per-type `MarshalJSON`/`UnmarshalJSON` 方法承载（`encoding/json` 驱动递归），SHALL NOT 依赖运行时通用反射引擎（ygot `ytypes.Unmarshal`/`ygot.EmitJSON` 退役）。生成方法 SHALL 完整覆盖 RFC7951 语义并与既有 ygot 行为等价：list map↔JSON 数组、(u)int64 字符串化、枚举编码为值域名、union 按成员类型试探解码、模块限定 key 兼容（接受带/不带模块前缀的字段名，输出与既有 `EmitJSON(RFC7951)` 形状一致）、未知字段与类型不符 SHALL 返回命名字段的明确错误（R08）。

#### Scenario: JSON 通道 golden 等价（迁移对拍）
- **WHEN** 以同一 RFC7951 输入分别经旧 `ytypes.Unmarshal`+`EmitJSON` 与新生成方法做 解码→编码 往返
- **THEN** 两者输出 SHALL 逐字节一致（全模块 fixture 集 + union/int64/枚举/嵌套 list 构造样本）

#### Scenario: (u)int64 字符串化
- **WHEN** 编码含 int64/uint64 叶的结构体
- **THEN** JSON 输出 SHALL 为字符串形态（RFC7951 §6.1），解码 SHALL 接受字符串与数字两种形态

#### Scenario: union 成员试探解码
- **WHEN** 解码 union 叶（如 234 个既有 union 类型之一）
- **THEN** SHALL 按成员类型序试探并还原为对应 union 具体类型，全部不匹配时返回命名该叶的明确错误

#### Scenario: 非法输入负路径
- **WHEN** 解码类型不符（如枚举叶给未知值域名）的 JSON
- **THEN** SHALL 返回命名字段的明确错误，SHALL NOT panic、SHALL NOT 静默丢字段

### Requirement: YN-03 Schema IR 构建期序列化与运行期加载

框架内部 Schema 模型（`pkg/yang-runtime/schema` Node 模型）SHALL 由构建期工具直接从 YANG 源（goyang 解析，构建期允许）序列化为自有格式（含格式版本号字段），生成物入库；运行期 SHALL 仅加载自有格式，SHALL NOT 依赖「ygot gzip schema → goyang Entry → 运行期转换」链路（`schema.AddYgotSchema`/`schema/entry.go`/`schema/loader.go` 运行时路径退役）。IR SHALL 承载既有内部模型全部信息（含 when/must/pattern/range/length/enum、config-false 继承、presence、扩展注解），并新增 xmlcodec 所需元数据（模块 namespace、list key 名链），替代从 `huawei.SchemaTree` 取 `yang.Entry` 的用法。

#### Scenario: schema 通道 golden 等价（迁移对拍）
- **WHEN** `/yang/schema` 对全部模块分别以旧链路与 IR 链路产出 JSON
- **THEN** 两者 SHALL 逐字节一致（复用 68 模块派生黄金地基口径）

#### Scenario: 格式版本不匹配快速失败
- **WHEN** 运行期加载的 IR 版本号与运行库支持版本不符
- **THEN** 启动 SHALL 返回指明重新生成的明确错误，SHALL NOT 以半解析 schema 继续运行

#### Scenario: 约束元数据存活
- **WHEN** 对含 when/must/pattern/range/leaf-list/presence/只读继承的模块经 IR 加载
- **THEN** 前端 `/yang/schema` 派生结论与约束提示 SHALL 与既有行为一致（约束引擎零回归）

### Requirement: YN-04 服务端校验子集（现状等价）

系统 SHALL 提供基于 Schema IR 的服务端校验器替代生成的 `Validate()`（现仅意图层消费）：覆盖 pattern/range/length/enum 值域校验，与前端表单校验同源（同一 IR 元数据）。范围 SHALL 为现状等价：must/when SHALL NOT 做运行时求值（与 ygot `Validate()` 现状一致，设备侧兜底）。校验失败 SHALL 返回命名路径与约束的明确错误。

#### Scenario: 意图校验行为等价
- **WHEN** 以既有意图校验（`intent/cr.go`）的行为快照用例分别跑旧 `Validate()` 与新校验器
- **THEN** 接受/拒绝结论 SHALL 一致，拒绝时错误 SHALL 指明违反的叶路径与约束

#### Scenario: 越界值拒绝
- **WHEN** 提交 range 越界或 pattern 不匹配的叶值
- **THEN** 校验 SHALL 拒绝并命名该叶，SHALL NOT 下发到设备

### Requirement: YN-05 发布二进制零 ygot/goyang 守护门禁

`usmp-backend` 发布二进制的 import 闭包 SHALL NOT 含 `github.com/openconfig/ygot` 与 `github.com/openconfig/goyang` 任何子包；SHALL 由守护测试断言（`go list -deps` 口径，仿 NC-01 scrapligo 禁回引），tools、测试、simulator 豁免。两库在 go.mod SHALL 仅作为构建期工具与测试依赖存在。

#### Scenario: 回引被守护测试拦截
- **WHEN** 任何运行时包（`backend/main.go` import 闭包内）新增 ygot/goyang import
- **THEN** 守护测试 SHALL fail 并指明违规包路径

#### Scenario: 构建期工具豁免
- **WHEN** `backend/tools/*` 使用 goyang 解析 YANG 源
- **THEN** 守护测试 SHALL NOT 拦截（构建期依赖不入发布二进制）

### Requirement: YN-06 迁移期三通道 golden 对拍（脚手架）

迁移期间新旧链路 SHALL 并行存在并以三通道 golden 对拍验证等价：JSON 通道（YN-02）、XML 通道（xmlcodec 新旧类型输出逐字节比对）、schema 通道（YN-03）。每通道对拍 SHALL 全绿后方可切换并删除旧链路（W02 同精神）；对拍测试为迁移脚手架，切换完成后 SHALL 随旧链路删除，golden 文件 SHALL 保留为回归基线。

#### Scenario: 对拍未绿禁止切换
- **WHEN** 任一通道对拍存在差异
- **THEN** 对应切换 PR SHALL NOT 合入（差异先归因：新实现缺陷 or 旧行为 bug，后者须显式拍板）

#### Scenario: XML 通道等价
- **WHEN** 同一配置树分别以 ygot 结构体与新生成结构体经 xmlcodec 编码（含删除编码）
- **THEN** 输出 XML SHALL 逐字节一致
