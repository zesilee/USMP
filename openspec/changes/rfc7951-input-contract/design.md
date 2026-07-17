# rfc7951-input-contract — 设计

## Context

- generic 解码：`EncoderFor(path)` 命中描述符 → `NewStruct()`（**锚点容器**，如 `HuaweiVlan_Vlan_Vlans`/`HuaweiSystem_System`，非模块根）→ 生成的 `Unmarshal`（RFC7951 语义、锚点相对）。
- spike 实证（2026-07-17）：`{"vlan":[...]}` 在锚点 `/vlan:vlan/vlan:vlans` 直接成功；`/system:system/system:system-info` 的扁平载荷经 `{"system-info": …}` 一层包裹后正确进入容器（YANG 真名 kebab，如 `sys-name`）。控制台的 schema 派生 key 就是 YANG 真名，天然满足。
- 失败根因是**锚点未知**：描述符只有 `MatchEncode` 谓词，无法算「path 相对锚点的后缀段」。

## Goals / Non-Goals

**Goals:** 单一解码路径；form-tab 全模块类型化解码；raw-map 静默入库消灭；legacy 全删。

**Non-Goals:** 不做 path 含 list 谓词（`[key=x]`）的子树写入（现行 UI 不产生此形状，显式 400 留待需要时扩展）；不动 DELETE 通道与 GET 读路径；不改前端载荷生产逻辑。

## Decisions

### D1 锚点相对包裹在服务端做，而非前端拼 RFC7951

包裹 = 纯机械段名嵌套（path 剥锚点前缀后逐段 `{"seg": …}`），服务端一处实现全模块受益；前端保持「body=path 处子树」的直觉契约，也免去 schema 树在前端的完备性依赖。**备选**（前端包裹）被否：每个调用点都要包裹逻辑，且历史上前端 schema 树曾有 choice/嵌套缺口。

### D2 Descriptor 新增 `EncodeAnchor string`

`NewStruct` 容器的规范配置路径（如 `/vlan:vlan/vlan:vlans`、`/system:system`）。包裹段 = `strings.TrimPrefix(path, anchor)` 的斜杠段剥模块前缀。path 与 anchor 不构成前缀关系或段含 `[` 谓词 → 显式 400。这是 SND 声明式化（谓词→数据）的第一块：后续 `MatchEncode` 可由 anchor 派生退役。

### D2b 存储/触发 key 归一化为锚点（apply 期发现补全）

实现中发现：SetConfig 历来以**请求路径**为 desired key，而解码值以**锚点**为根——子路径下发的 desired 周期对账（模块路径入队）读不到，且会留分叉副本。补全：`convertConfigAnchored` 返回锚点，SetConfig 的存储/失效/TriggerReconcile 一律用锚点；审计仍记请求路径（诚实记录用户动作）。

### D3 未注册/不可解 → 400，raw-map 路径删除

`convertConfig` 变为：`EncoderFor` 未命中 → 400（错误信息含 path 与已注册模块提示）；`Unmarshal` 失败 → 400 透出 ygot 错误（§9 诚实）。ConfigStore 从此只存类型化 ygot 结构——下游 diff/XML 编码链的类型假设成为不变式。

### D4 legacy 删除边界

- `convertToTypedStruct` + 三个 `convertMapToHuawei*` + 各 `mapEntryTo*` 助手（config_handler.go:361-1045）。
- yang-api `GetSchema` 硬编码 alias switch（真名模块本就走 schema 树分支；未加载模块保留既有「最小通用 schema」降级，BR-04 行为不变、无 delta）。
- `DeviceConfigPage.vue`（无路由引用）、`types/yang-schema.ts`（零 import）、router 里的暂存注释。

### D5 测试形状与矩阵

旧形状测试（`{"vlans":[…]}`、camelCase）改 RFC7951 真名形状；B1 新增：包裹计算表格驱动（锚点相对/多层/谓词拒绝/非前缀拒绝）、未注册 400、Unmarshal 失败 400；B2：system form-tab 形状经 netconfsim 端到端（新契约下的唯一真实生产流回归）。

### D6 三 PR

①立项制品（本 PR）→ ②实现：anchor 字段+包裹+单一路径+测试改形状（~+500/-250，≤3000 档）→ ③纯删除（legacy 转换器+alias+前端死码 ~2140 行，豁免档）+ sync/archive。②合入后 ③ 才能删（③删除物在 ② 后零引用）。

## Risks / Trade-offs

- **[风险] 某控制台 form-tab 的段名与生成物 tag 不一致（schema 树名 vs ygot path tag）** → 两者同源（goyang Entry.Name），B2 端到端兜底；发现不一致即模型/生成物 bug，修源头。
- **[风险] 存量 desired（内存）无兼容问题** → ConfigStore 仅内存、进程重启清空；意图控制器产出的本就是类型化结构。
- **[取舍] path 带 list 谓词的子树写入不支持** → 现行 UI 不产生；400 显式拒绝优于半吊子实现，需要时按 DR 锚点扩展。

## Migration Plan

无现网（用户拍板），零迁移。回滚 = revert 对应 PR。

## Open Questions

（无）
