# device-native-lowcode-config — tasks（P1 场景① 设备原生面真低码）

> 依赖：`proposal.md` `design.md` `specs/*`。TDD（T01/R06）：每步测试先行。渐进迁移（§5.3）：旧保留 → 新并行 → 双路径验证 → 切换 → 删除。
> 每 commit ≤500 行；每 PR ≤800 行（>20 文件则 ≤3000）；每模块含 netconfsim 集成测试（T02）。独立分支 + PR + CI 全绿合入。

## 0. 基线与前置（P0：修 D7）

- [x] 0.1 基线：`go test ./...` 全绿（对拍快照留待 group 2/3 的 yang-api/config-api 双路径实现时采集）
- [x] 0.2 先写测试：`ConfigStore.List`/`ListDevices` 单测（多设备去重、空列表、-race）+ 新增 `cache.Keys()` 单测（存活/空/排除过期）
- [x] 0.3 实现 `cache.TTLLRUCache.Keys()`（RLock 快照、排除过期）+ `ConfigStore.List`/`ListDevices`（解析 `deviceID:path` 键，纯内存修 D7），替换返回 nil 的 stub

## 1. schema 树加载（修 D4，混合源）

- [x] 1.1 先写测试：ygot yang.Entry→Schema 转换器（合成树：模块/叶类型/list 键/路径/nil 安全）+ 从 huawei/openconfig ygot 模型构建 schema（Modules() 非空含 ifm/system/vlan/interfaces/vlans、叶属性非空）
- [x] 1.2 实现 `schema.AddYgotSchema`（yang.Entry→框架节点模型）+ `internal/yangschema.Load`（huawei+openconfig）+ manager `WithSchema` 选项 + `main.go` 挂载（去 D4 空转，schema 树运行时非空）
- [x] 1.3 先写测试：`schema.NarrowModulesByCapabilities`——命名空间/module= 参数/名称启发式匹配；仅 base caps→回退全集
- [x] 1.4 实现 collapse 逻辑 + client `ServerCapabilities()`（读 scrapligo 服务端 hello caps）+ netconfsim `SetCapabilities`（hello 广告额外模块 caps）；`<get-schema>` 增强留 TODO
- [x] 1.5 集成测试（netconfsim，真 SSH）：sim 广告 huawei-vlan cap → client 读回 → 收敛到 vlan（排除 system/interfaces/ifm）；base-only→回退全集

## 2. yang-api 动态化（去硬编码）

- [x] 2.1 先写测试：FieldDef 生成器（类型映射 bool/number/enum/string、list 键→ListCols）+ handler `GetSchema` 动态（真实 vlan schema 多字段、非 2 字段桩）
- [x] 2.2 实现 `field_gen.go` schema 树→FieldDef 生成器（YANG LeafType→控件类型）+ Module `Vendor()` 贯穿（load 时标 huawei/openconfig）
- [x] 2.3 切 `GetSchema` handler：schema 树命中→动态生成；**保留旧硬编码为回退分支**（legacy 别名，task 2.5/组5 删）
- [x] 2.4 `ListModules` 从 schema 树枚举，vendor 用 `mod.Vendor()`（vlan→huawei、interfaces→openconfig）；删无模块时的硬编码桩；响应结构兼容
- [ ] 2.5 前端切换后（组4）删旧硬编码 schema 回退分支（并入组5清理）

## 3. config-api 通用编解码（去硬编码三条）

- [x] 3.1 先写测试：`encodeToYgot` RFC7951→ygot（vlan list 键）；未注册路径→未命中回退；raw-map 回退加告警日志（不静默截断）
- [x] 3.2 实现 `config_codec.go`：path→ygot 注册表 + 单次 `ygot.Unmarshal`（以 ygot 生成结构为准 R04），替代手写 map 转换器
- [x] 3.3 先写测试：双路径 dispatcher——RFC7951 走通用、legacy 整数枚举 shape 走回退，均得有效 ygot 结构（无回归）
- [x] 3.4 切 `config-api` POST 到 `convertConfig`（通用优先、旧 convert 回退），§5.3 双路径并行
- [~] 3.5 端到端：现有 reconciler 集成套件（config-api→reconcile→netconfsim）经 convertConfig 全绿；真·「非旧三条」新路径 e2e 受限于无第 4 个 reconciler，留待增设控制器
- [ ] 3.6 前端切 RFC7951 后（组4）删旧 `convertToTypedStruct` 硬编码（并入组5清理）

## 4. 前端接通动态 YANG 表单（复用活跃引擎）

- [x] 4.1 Vitest：`useConfigPage` 原生分支 unwrap `{data}` 信封→动态 YANG fields（number/enum）→复用 `Field[]`（DynamicForm 引擎）；`useNativeModules` unwrap data 数组 + vendor 分组
- [x] 4.2 修 `useConfigPage.ts` 原生分支 schema 源信封解包 bug（`yangSchema.fields`→`.data.fields`）+ `useNativeModules`（`data.models`→`.data`）；FieldDef 结构对齐前端 `Field`，复用活跃 DynamicForm，不新建渲染器
- [~] 4.3 编辑→提交：原生页现经 NativeDeviceConfig CRD（Stack A，下发为 D6 stub）；切到 config-api RFC7951 直接下发是较大改动（耦合 D6/Stack A），留作后续
- [~] 4.4 E2E（Playwright）本环境无法无头运行；以 Vitest 单测 + 前端 CI(Node22) 覆盖读路径

## 5. 清理死代码（退役 D9 设备侧）

- [x] 5.1 grep 确认无活跃引用（含 test/ 孤儿测试）
- [x] 5.2 删除设备侧静态 YANG 渲染路径：`components/yang/*`（8 组件）+ `useDeviceConfig` + `api/crd` + 孤儿测试；前端 CI 绿。（残留死类型文件 `types/yang-schema.ts` 1047 行，受 PR ≤800 约束留待单独分批删除）

## 6. 收尾与验收

- [x] 6.1 全量 `go test ./...` 绿（含 netconfsim 集成，-race，历次 PR）；前端 vitest 绿（前端 CI Node22）；`go build ./...` + `go build ./cmd/netconf-simulator` 绿
- [x] 6.2 `system-architecture/{tasks,design}.md` 勾除 D4/D7/D9；delta spec 已随 change 记录（4 能力 ADDED Requirements）
- [x] 6.3 验收：交换机全量 YANG 属性经通用控件 + 动态 schema 低码渲染（vlan 23 字段等）；schema 树可用（6 模块）、设备可枚举、capabilities 收敛；yang-api/config-api 动态优先（旧硬编码作 §5.3 迁移期回退，前端切 RFC7951 后删=2.5/3.6）
- [x] 6.4 满足 R01（落 Stack B）/R03（纯内存）/R04（ygot 生成）/R05（通用控件动态渲染）/R06（TDD）；未触及场景②意图面/Actor（P2 边界守住）
