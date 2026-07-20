# internal/controller — Reconciler 与 YANG 模型的映射

> 目录**刻意不与 YANG 模型一一对应**（full-yang-onboarding design D4）。
> 映射的唯一事实源是驱动描述符注册表 `internal/drivers/`：每个 YANG 模块一条
> `driver.Descriptor`，Reconciler 按描述符形态分两层。

## 两层映射

| 层 | 模块 | 描述符所在 | Reconciler | 控制器装配 |
|----|------|-----------|------------|------------|
| 专属包（5 个） | system / vlan / ifm / bgp / network-instance | `drivers/huawei.go` 手写块 | `controller/{system,vlan,ifm,bgp,networkinstance}` | `main.go` 逐个显式注册 |
| 泛型共用（57 个） | 其余全部表内模块（ntp、qos、mstp、acl、tunnel-management…） | `drivers/huawei_modules.go` 的 `plainModules` 表（一行一模块） | **共用 `controller/plainmodule`**（按锚点参数化） | `main.go` 按 `driver.All()` 循环实例化 |

路由规则：`TriggerReconcile` 以描述符 `ControllerToken` **精确匹配**控制器名
`<vendor>-<token>`（全量控制器下子串匹配必撞：routing⊂routing-policy、
ifm⊂ifm-trunk），并以描述符 `EncodeAnchor` 归一入队（desired 以锚点为 key）。

## 为什么这 5 个保留专属包

不是「单容器根整树收敛」的通用形状：

- **vlan / ifm**：锚定嵌套子根（`/vlan:vlan/vlan:vlans`、`/ifm:ifm/ifm:interfaces`），
  diff/删除按 list 中心语义处理
- **system**：无 XML 编解码通道（描述符 XML 为 nil，走降级路径）
- **network-instance**：跨模块 augment 的 per-node namespace 映射（XC-06）+
  `/ni:` 与根名双口径兼容（业务意图编排层仍以 `/ni:` 调用）
- **bgp**：公网根精确锚定（排除 `/bgp-flow:` 等 feature 前缀与 per-VPN 增强）

历史注记：tunnelmgmt/xpl/routingpolicy/acl 四个专属包曾存在（BGP 2b 波次），
与泛型实现逐行同构且锚定的旧 YANG-prefix 路径与前端派生口径不符，已随
full-yang-onboarding 退役——测试职责由 `plainmodule` 的参数化 sim 集成矩阵承接。

## 常见操作

- **加新模块**：`plainModules` 表加一行 + `gen.conf` 加模块名 + `make gen-yang`。
  **不要**新建控制器包、不要写手写描述符块；参数化测试矩阵（drivers 往返 /
  plainmodule sim 集成 / api 编包与 LT-04 基线）自动覆盖。
- **某表内模块需要特化语义**（如深层 list 精细 diff）：把它从表里拆出去建
  专属包 + 手写描述符块，而**不是**往 `plainmodule` 加 if 分支。
- **路径口径军规（DR-06）**：运行时配置路径前缀 = 根容器名
  （`/<root>:<root>/...`，前端 `configPathFor` 同口径）；YANG prefix 只属
  XML namespace 层，禁止用作路由 token。
