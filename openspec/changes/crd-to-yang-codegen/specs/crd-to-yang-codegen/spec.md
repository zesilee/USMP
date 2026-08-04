# crd-to-yang-codegen — 北向 CRD→YANG 反向代码生成

## ADDED Requirements

### Requirement: C2Y-01 CRD manifest SHALL 可转换为合法 YANG 模块

`tools/crd2yang` SHALL 读入一个 `apiextensions.k8s.io/v1` CRD manifest（YAML），只取 storage 版本的 `.spec` schema 子树，生成一个可被 goyang 解析、可被 ygot 消费的 YANG 模块文件；`.status` 子树 SHALL 整体丢弃（status 为 USMP 侧固定契约 BIC-04，不属于北向 spec 语义）。`.spec` object SHALL 被包成单一顶层 data container，闭合 crdgen 的单容器约定。

#### Scenario: 示例 CRD 转换成功

- **WHEN** 以示例 CRD `BusinessVlanNet` manifest 为输入运行 `crd2yang`
- **THEN** 系统 SHALL 输出 `usmp-business-vlan-net.yang`，goyang 解析零错误，与 golden 文件逐字节一致

#### Scenario: 多 served 版本 fail-fast

- **WHEN** 输入 CRD 有多个 `served: true` 版本且无唯一 `storage: true`
- **THEN** 工具 SHALL 报错退出并指明版本歧义，SHALL NOT 静默取第一个

### Requirement: C2Y-02 元信息 SHALL 由注解约定携带，revision 缺失即失败

模块级 YANG 信息 SHALL 从 CRD `metadata.annotations` 读取：`usmp.io/yang-revision` 为**必填**（决定契约版本，SHALL NOT 隐式生成）；`usmp.io/yang-module`、`usmp.io/yang-namespace`、`usmp.io/yang-prefix`、`usmp.io/yang-container`、`usmp.io/task-name` 可缺省，缺省值 SHALL 按确定性规则派生（module=`usmp-<singular>`、namespace=`urn:usmp:yang:<module>`、container=singular、task-name=`business-network`）。生成的模块 SHALL 含 task-name 扩展声明与 `generated-from: <CRD name>` 溯源标注。

#### Scenario: 缺 revision 注解

- **WHEN** 输入 CRD 无 `usmp.io/yang-revision` 注解
- **THEN** 工具 SHALL 报错退出并提示补注解，SHALL NOT 用当天日期兜底

#### Scenario: 全缺省派生

- **WHEN** 输入 CRD 仅带 revision 注解，其余元信息注解缺省
- **THEN** 生成的模块名/namespace/prefix/容器名/task-name SHALL 与文档化的派生规则一致且两次运行结果逐字节相同

### Requirement: C2Y-03 结构与类型映射 SHALL 与 crdgen 互逆，未知构造 fail-fast

映射 SHALL 遵循：object→container；object 数组+`x-kubernetes-list-map-keys`→list（key=map-keys，key 属性从 mandatory 剥离）；scalar 数组→leaf-list；integer+min/max→最小可容纳宽度的 uintN/intN 并带 `range`；无界 integer→int64；string+pattern(s)→string+pattern(s)；string+enum→enumeration；boolean→boolean；required→`mandatory true`；description→description。可映射集之外的 OpenAPI 构造（oneOf/anyOf/additionalProperties/format 语义类型/nullable、无 map-keys 的 object 数组等）SHALL fail-fast 并报出字段 JSON 路径，SHALL NOT 静默降级或丢弃。

#### Scenario: 映射表全分支覆盖

- **WHEN** 输入含 uint16 range、多 pattern string、enum、boolean、带 key 的 list、leaf-list、嵌套 container、mandatory 字段的 CRD
- **THEN** 生成的 YANG SHALL 对每种构造产出上表对应语句（表格驱动测试逐分支断言）

#### Scenario: 无 key 的对象数组

- **WHEN** `.spec` 内某 object 数组无 `x-kubernetes-list-map-keys`
- **THEN** 工具 SHALL 报错并给出该字段 JSON 路径（YANG config list 必须有 key）

### Requirement: C2Y-04 往返对账 SHALL 零失真

对任一可映射 CRD：`crd2yang` 生成的 YANG 再经 `crdgen` 生成 CRD′，其 spec schema 与输入 CRD 的 spec schema 在归一化后（剥离 `x-kubernetes-*`、list key 并回 required）SHALL 深度相等。此闭环 SHALL 以集成测试固化（`testing.Short()` 跳过）。

#### Scenario: 示例 CRD 往返

- **WHEN** 示例 CRD → crd2yang → YANG → crdgen → CRD′
- **THEN** 归一化后的 spec schema 深度比对 SHALL 零差异（min/max/pattern/enum/required/嵌套结构全保持）

### Requirement: C2Y-05 生成 YANG SHALL 接入既有业务配置管线并可自动渲染

生成的示例 YANG SHALL 通过 gen.conf manifest 管线产出独立 `businessdemo` ygot 包（R04 生成物入库）；测试 SHALL 断言其 schema 可经 `AddYgotSchemaWithVendor` 注册进 schema 树；schemadump SHALL 纳入该包产出 schema fixture，前端派生黄金套件（GD-01）SHALL 覆盖该模块。示例模型 SHALL NOT 接入运行期 `load.go`（业务控制台当前单 Kind 写死，接线属后续 change）。

#### Scenario: ygot 与 schema 树

- **WHEN** 执行 `make gen-yang VENDOR=businessdemo` 后运行后端测试
- **THEN** businessdemo 包 SHALL 编译通过且 schema 树注册测试 SHALL 断言模块树形完整（container/list/leaf 层级与约束元数据在位）

#### Scenario: 前端派生黄金

- **WHEN** 执行 `make gen-schema-fixtures` 后运行前端派生黄金套件
- **THEN** `usmp-business-vlan-net.json` fixture SHALL 存在且 deriveTabs/deriveColumns/deriveKeyField 黄金 SHALL 全绿

### Requirement: C2Y-06 生成物 SHALL 受 regen-and-diff 门禁保护

`make gen-crd2yang` SHALL 幂等重生成示例 YANG；CI compliance 与 `.githooks/pre-commit` SHALL 同口径校验生成物零漂移（R04 方法论、CG-03 对称），手改生成的 `.yang` SHALL 被门禁拦截。

#### Scenario: 手改生成物被拦

- **WHEN** 直接编辑生成的 `usmp-business-vlan-net.yang` 后提交
- **THEN** regen-and-diff SHALL 检出漂移并使检查失败
