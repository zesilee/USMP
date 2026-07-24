---
name: vlan-config-stackb
description: 华为 VLAN 配置已交付(全属性)；关键架构事实——设备配置走 Stack B 直连 API，非 Stack A K8s CRD
metadata: 
  node_type: memory
  type: project
  originSessionId: e83cdcca-14a0-4971-8bbb-0da5ae835af1
---

2026-07-05 交付华为 VLAN 全属性配置（PR #64 后端 7ec75ee + #65 前端 27673cb）。关联 [[dual-stack-migration]] [[frontend-contract-gen]]。

**关键架构事实（改任何设备配置页前必读）**：设备配置的**能用链路是 Stack B 直连**：
- schema：`GET /api/v1/yang/schema/{module}`（buildYangSchema 动态从 ygot 模型生成，`?form=nested` 出嵌套树）
- 读：`GET /api/v1/config/{ip}/{path}`；下发：`POST /api/v1/config/{ip}/{path}` → convertToTypedStruct 按 path 子串路由（含 `vlan:`→VLAN、`system:`→System、`ifm:ifm`+`interfaces`→IFM）→ ConfigStore → 对账 → NETCONF。
- ~~Stack A 前端 CRD 死路（useConfigPage/useK8sCRD/ConfigPage、/native/*、/config/route）~~ **已于 2026-07-13 整链退役删除**（change native-config-reposition，PR #143）——前端零 CRD 消费，下发链路唯一 = Stack B 直连。华为模块在 schema 树里键名是 `vlan`/`ifm`/`system`（非 `huawei-*`、非 `openconfig-*`）。**概念分层已拍板**：「原生配置」=模块控制台直连 YANG 配置（侧边栏菜单已更名）；「业务网络配置」=未来编排层（openspec/tasks/business-network-config.md，业务 YANG 模型→编排为原生配置，`business*` 标识符已清场留给它）。

**已泛化为通用配置流（PR #68 后端 964293f + #69 前端 c630adf）**：VLAN 与**接口(Huawei IFM)**共用 `useDeviceConfig(opts)` + `DeviceConfigPage.vue`（props 驱动），路由 `/config/vlan`、`/config/interface` 均指它。新增华为模块（system 等）只需加一条路由参数（module/configPath/itemListSuffix/listKey/keyField/columns）。VlanConfig.vue/useVlanConfig.ts 已删。接口 module 键=`ifm`，路径 `ifm:ifm/ifm:interfaces`，`mapEntryToInterface` 6 枚举已改 enumInt。route 无华为模型（huawei 只有 vlan/ifm/system）；~~其 openconfig 死路页面~~ 已随 2026-07-13 退役删除（/config/route 路由已不存在）。staging-smoke 有 VLAN + 接口两条「新增表单动态渲染」断言（真浏览器绿）。

**路由复用坑（PR #70 9028ec7）**：VLAN/接口共用同一个 DeviceConfigPage，vue-router 在**同组件相邻路由间复用实例** → setup/onMounted 只跑一次 → SPA 内从 VLAN 切到接口时 schema 不重载、接口表单显示 VLAN 字段。修法：`App.vue` 的 `<router-view :key="$route.path">` 强制重建。**测试盲区教训**：单测测组合式函数隔离、e2e 每条都 `page.goto` 全量重载各自页面 → 从不走「应用内导航」→ 复用路径从未触发。已补：真 App+真路由的 SPA 内导航单测（`DeviceConfigPage.nav.test.ts`）+ staging-smoke「SPA 内 VLAN→接口点侧栏」断言。**加同组件多路由页时必测应用内导航，别只 goto 各页**。

**测试加固（PR #71/#72/#73）——补 P0/P1/P2/P3 用例共暴露并修复 4 个真 bug**（有力证明原有测试只测 happy-path 管道、不保证稳定交付）：
1. **member-ports 下发崩溃**（PR #71）：对账 diff 的 VLAN map 走通用 `xml.Marshal`(不支持嵌套 member-port map)→ 路由到 `buildHuaweiVlanVlansXML` + 补全该 builder 缺字段(statistic-discard/unknown-multicast-discard/suppression/unkown-unicast-discard)。
2. **增量配置抹除既有 VLAN**（PR #71，数据丢失）：声明式对账把 desired 当完整状态,UI 每次只发单个 → `SetConfig` 下发前 `mergeConfig` 并入已存 desired。
3. **unkown-unicast-discard 静默丢弃**（PR #73）：`mapEntryToVlan` case 用了纠正拼写 `unknown`,而 YANG 实为 `unkown`(缺 n)→ 归一化不匹配 → 整容器丢失。**华为该字段拼写就是 unkown,sim 解析/converter/builder 都要按此**。
4. **并发配置竞态+丢更新 R09**（PR #73）：Get-merge-Set 非原子 + mergeConfig 原地改共享 map → `storeConfigMerged`(单锁 configMergeMu 串行) + mergeConfig 改构造新副本。
- 校验（PR #72）：VLAN ID 无 YANG range → `validateConfig` 显式 1-4094 返回 400；前端 `DeviceConfigPage` el-form 提交前 `validate()`。
- **框架级并发债（未修，独立于 VLAN）**：TTL-LRU cache 与 client pool **对同一设备连接非并发安全**；对账器直接并发用同一 NETCONF 连接会 race。生产中对账由控制器**事件队列串行化**故不触发；若未来改并发对账须先加固 cache/pool。VLAN 并发测试因此只并发「配置存储」不并发对账。
- 测试文件：`vlan_hardening_integration_test.go`(member-ports/merge/newVlanSimStack/applyVlan/storeConfigMerged)、`vlan_comprehensive_test.go`(全属性/幂等/编辑/替换/读回/并发-race/畸形)、`vlan_validation_test.go`。sim 已能解析 member-port(`AssertHuaweiVlanMemberPort`)。

**VLAN/接口实现要点**：
- 前端通用流：schema `getYangSchema(module,'nested')`→`extractItemFields`(DFS 按 path 后缀找 list)→`DeviceConfigPage` 动态渲染；list/下发走 `getConfig`/`setConfig`。
- `FieldRenderer` 支持嵌套：`group`（对象）/`list`（可重复子表单行，member-ports 端口成员）。数据键用 **YANG 叶子名**（path 末段），非 full path，对齐后端 convertMapToHuaweiVlan。
- 后端 `convertMapToHuaweiVlan` 枚举用 `enumInt`（数字直通 + 字符串名经 `huawei.ΛEnum` 反查）；`mapToMemberPorts` 转端口成员列表。**踩坑**：nodeToNestedField 的 case 顺序 list 必须先于 container（list 同时满足两接口）；表单发枚举字符串名而非数字，后端此前只认数字是断裂点。
- **裸相对 fetch 坑（PR #66 515c140）**：staging 的 nginx（frontend/nginx.conf）**只服务静态站、不代理 `/api`** → 前端任何 `fetch('/api/v1/...')` 裸相对请求都被 SPA 回退成 index.html（dev 下 vite 代理 /api→:8080 才正常，故只在 staging 暴露）。**必须走 api 客户端**（axios，绝对 baseURL `http://localhost:8080/api/v1`，CORS 允许）。（useConfigPage 的同类隐患已随 2026-07-13 整链删除消失；`stores/menu.ts` 的 loadNativeModules 仍是裸相对 fetch——dev 正常、staging 靠回退项兜底，属已知既存形态。）已加 `getYangSchema(module, form?)` 走客户端。
- **回归门禁（PR #67 77c17c1）**：staging-smoke 增「VLAN 新增表单应动态渲染出 YANG 字段」——进 /config/vlan→选设备→新增→断言 admin-status 字段可见，真浏览器在部署栈上兜住「表单空」回归。
- 华为 VLAN 可配置属性（生成结构=真源）：id/name/description/admin-status/type/mac-aging-time/mac-learning/broadcast-discard/statistic-discard/statistic-enable/unknown-multicast-discard/super-vlan + 嵌套 member-ports(interface-name/access-type/tag-mode)/suppression/unkown-unicast-discard。sim 的 HuaweiVlanTestData **不含 member-ports**（集成断言只到扁平+枚举字符串端到端）。
