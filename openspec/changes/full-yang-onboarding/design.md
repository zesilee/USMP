# Design: full-yang-onboarding

## Context

左树 61 叶仅 11 可用。管线三件套（描述符注册表 / 通用 XML 编解码引擎 / manifest 生成管线）已交付，接入边际成本 = 一条 gen.conf 模块名 + 一条描述符数据。试验性全量生成已验证：51 个新模块（49 叶 + huawei-ip 依赖 + usmp-deviations）可一次生成（13→67 根容器，15MB，编译 12s），失败面收敛为 5 类个别节点（deviation 可豁免）+ 1 个解析期死结（pic，延期）。

同时暴露存量断链：前端 `configPathFor` 以**根容器名**派生路径前缀，而 2b 波次的 tunnel-management/routing-policy/network-instance 描述符锚定 **YANG prefix**（`/tnlm:` `/rtp:` `/ni:`）——根名≠prefix 的三个模块控制台写链路不可达（当时按 leafref 前置目标交付、未走控制台 e2e）。

## Goals / Non-Goals

**Goals:**
- 60/61 叶全部可用：生成 + 注册 + 控制器 + 控制台可路由
- 路径约定统一为根容器名口径（修复三模块断链）
- 参数化 T02b 矩阵：对每个新模块统一断言，代替逐模块手写
- deviation 机制固化（可复现、有据可查、snd 只读）

**Non-Goals:**
- 不做 per-module 深层功能面打磨（各模块深层 choice/presence/ordered-by 精细交互按需后续波次，同 2b「标量边界」先例）
- 不接非华为厂商（P5-4 已剪出）
- 不救 huawei-pic（goyang 上游缺陷，延期项）
- 模拟网元不为 49 个新模块造种子数据（回读空树是合法初态）

## Decisions

**D1 全量单波次接入，不分批**
管线已泛化，逐模块波次的调度成本超过其风险收益；生成物豁免体积门禁，真实代码 diff（管线+表+泛型 reconciler+测试）约 900 行，单 PR 可容（TM04）。

**D2 deviation 模块承载全部生成器豁免**
`backend/internal/yang/deviations/usmp-deviations.yang`，五条豁免（syslog×2 bits-default、cfg anydata、qos binary-key 查询列表、lldp 穿 choice/case leafref）。备选：fork snd 模型改本体——违反 snd 只读原则、升级即冲突，否决。

**D3 表驱动注册：数据行 + 统一注册循环**
`plainModules` 表（module/root/ns/构造子），`registerPlain` 派生谓词（`HasPrefix "/<root>:<root>"`）、锚点、xmlcodec.Spec（SchemaTree key 经反射从构造子类型名派生）。既有 5 模块（system/vlan/ifm/bgp + ni）保留手写块（谓词有历史兼容语义）；tnlm/xpl/rtp/acl 迁入表（谓词改根名口径，xpl/acl 根名=prefix 无行为变化，tnlm/rtp 为断链修复）。ni 根名锚（`/network-instance:network-instance`）与 prefix 锚（`/ni:`）**双谓词兼容**——业务意图编排层（business-vlan-service）以 `/ni:` 口径调用，不能破坏。

**D4 泛型 plain-container Reconciler**
xpl/tnlm/rtp/acl reconciler 完全同构（仅 path 与 GoStruct 类型异）。提取 `internal/controller/plainmodule`：`New(cs, pool, resolver, anchor, newStruct)`；整根收敛 diff（同构注释同 xpl）；gNMI JSON 分支以反射从 Device 根提取对应字段。main.go：既有显式控制器保留，新模块经描述符循环 `ControllerManagedBy("huawei-"+module)` 批量注册（跳过已有显式控制器的 token）。

**D5 参数化 T02b 矩阵（对每模块统一跑）**
- B1 注册表不变量：全部华为描述符 namespace 非空唯一、SchemaTree 入口可解析、根名路径三谓词命中、并发 race
- B1 编解码往返：每模块经 schema 采样构造最小实例（首个 config-true 标量叶）→ Encode→Decode 相等
- B3 API 编包：每模块根路径 `convertConfig` 包裹成功
- B2 sim 端到端（`testing.Short` 跳过）：抽样代表模块（每任务域 ≥1）走 写→回读→收敛
深层功能面（嵌套 list 增删改、when/must 交互）按 2b「标量边界」先例留待需求驱动波次，测试断言当前边界防悄悄越界。

**D6 延期项显式化**
pic 在 gen.conf 注释记录原因；LT-04 基线测试锁定「恰好 60 可用」，缩水或新增延期都红灯。

## Risks / Trade-offs

- [67 根容器 schema 全量加载拖慢启动/内存] → 试验实测编译 12s、包 15MB；启动加载为一次性 gzip 解包，可接受；如超预算再做懒加载（另 change）
- [lldp leafref 替换 string 失去引用校验] → 线格式不变；前端约束引擎仍渲染其余约束；deviation 注释记录影响面
- [新模块深层结构未逐一驗证] → 参数化矩阵保底「可配可读可收敛」；深层交互按需波次（与 2b 同一交付哲学）
- [qos 等巨模块 FieldDef 派生性能] → schema API 为按需请求 + 缓存；F4 smoke 加一个大模块控制台冒烟兜底
- [ni 双口径谓词长期共存] → 注释锚定业务编排调用点；编排层迁根名口径后可收敛（follow-up）
