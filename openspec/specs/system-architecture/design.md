# system-architecture — 系统架构总览（as-built）

> **文档性质**：由已实现代码还原的 as-built 现状描述。
> **还原基准**：2026-08-25 配置自审计随笔刷新（原 `main@b1cfbae` 双栈版全文已过期重写；当时的「两栈并存待决策」已于 2026-07 完成切栈并物理删除旧栈）。
> **本文是能力地图的锚点**：各子系统详细架构见 `openspec/specs/<capability>/`（§4 能力地图）。行为契约以各 spec.md 为权威。

---

## 1. 一句话定位

无数据库、高并发、模型驱动的交换机设备管理平台（K8s 内多实例 PaaS 组件）：前端由 YANG 模型自动渲染表单 → 后端将「期望配置」经声明式 Reconciler 对齐到设备 → 通过自研 netconfcore 引擎经 NETCONF（SSH 830）下发。

## 2. 单栈事实（历史双栈已终结）

**唯一进程入口 = `backend/main.go`**（yang-controller-runtime Manager + Beego API :8080）。

原 Stack A（K8s CRD + controller-runtime Operator + Actor 2PC + 翻译引擎）已于 2026-07-17 起**物理删除**（retire-businessvlan-bridge PR#186/#187、retire-stacka-residue PR#195），**SC-01 禁止重引**，守护测试拦截。历史脉络见 `docs/memory/{dual-stack-migration,arch-optimization-roadmap}.md`；当年的架构对照仅存于 `actor-transaction`/`business-crd`/`translation-engine` 三份 LEGACY spec。

K8s CRD 在现架构中**只当持久化+watch 载体，不当架构通道**（§8/R03）：Device 设备清单（凭据经 Secret）、AuditRecord 操作审计、业务意图 CRD（`backend/api/core/v1`）。多实例 HA 经 `pkg/yang-runtime/leader` 统一选主。

## 3. 数据流（单栈）

```
前端变更集 preview → commit（2PC，config-changeset）        PeriodicSource 周期轮询
        │  Beego API (internal/api)                              │
        ▼                                                        │
ConfigStore.Set(desired，合并存储+失效缓存) ──► Controller 队列 ◄─┘   （CRD watch 意图源另路汇入）
        ▼
GenericReconciler.Reconcile：desired(ConfigStore) ↔ actual(设备回读，TTL 30s 缓存)
        ▼  DiffEngine 差异计算（删除走 DELETE 命令通道，先移 desired 再下发）
netconfcore edit-config（写事务 opMu 串行）+ commit ──► 设备 (NETCONF SSH 830)
```

- 回读契约=以请求路径为根的子树（peelToPath）；状态读（config=false）走 `<get>` 通道。
- 设备能力：节点级被动学习 + 快速失败，能力查询 Peek 不拨号（device-capability-negotiation）。
- gNMI (9339/9340) 为**规划能力**：空壳已删，工厂返回显式未实现错误（R02 口径）。

## 4. 组件物理布局与能力地图

```
backend/
├── main.go                        # 唯一进程入口（Manager + Beego :8080）
├── api/core/v1/                   # K8s CRD 类型（持久化载体）→ business-intent-crd / device-store / operation-audit
├── pkg/yang-runtime/              # C1-C5 框架 → yang-controller-runtime
│   ├── manager/ controller/ queue/ predicate/   # C1/C2 生命周期与事件循环
│   ├── reconcile/ diff/ object/ validate/       # C3 + 差异引擎 + 约束
│   ├── source/                    # C4 周期轮询 / CRD watch
│   ├── client/ driver/ device/    # C5 连接池 + netconfcore + 驱动注册表 → device-protocol / device-driver-registry
│   ├── schema/ xmlcodec/          # schema 树 + 通用 XML 编解码 → yang-native-runtime / yang-xml-codec
│   └── leader/ audit/ status/     # HA 选主 / 审计 / 收敛状态
├── internal/api/                  # REST 北向 → devices-api / config-api / yang-api / config-changeset
├── internal/cache/                # TTL+LRU → config-cache（R03）
├── internal/controller/{vlan,ifm,system,bgp,networkinstance,plainmodule}/  # 各模块 Reconciler
├── internal/generated/{huawei,native,business}/  # 自研 yanggen 生成（R04，regen-and-diff 门禁）
├── internal/{yang,yangschema,intent,drivers,staticweb}/
├── tools/                         # 构建期独立 module：yanggen/schemagen/rpcgen 等 → yang-codegen-pipeline / crd-to-yang-codegen
└── simulator/netconfsim/          # 模拟网元 → netconf-simulator
frontend/                          # React19+openinula 动态表单 → frontend / frontend-runtime / frontend-ui-adapter
```

LEGACY spec（纯历史契约，代码载体已清零）：`actor-transaction`、`business-crd`、`translation-engine`。

## 5. 红线合规矩阵（as-built，2026-08-25）

| 红线 | 现状裁定 | 证据 |
|------|----------|------|
| R01 架构（禁 Actor） | ✅ 合规 | Actor/Stack A 载体 2026-07-17 物理删除，SC-01 守护禁重引 |
| R02 协议（仅 NETCONF/gNMI） | ✅ 合规 | netconfcore 唯一引擎（NC-01 禁 scrapligo 回引，守护测试）；gNMI 工厂显式未实现 |
| R03 无数据库 | ✅ 合规 | 运行配置仅 TTL+LRU 内存；持久元信息走 K8s CRD 载体；多实例禁本地文件 |
| R04 禁手写 YANG 结构体 | ✅ 合规 | 自研 yanggen 生成 `internal/generated/`；发布二进制+主 go.mod 双零 openconfig，守护测试拦回引 |
| R05 YANG 自动渲染 | ✅ 合规 | `/yang/schema` → 派生 → src/form，派生黄金（GD-01）钉住 |
| R08/R09 禁崩溃/竞态 | ✅ 常态防线 | -race 门禁、写事务串行化、异常降级契约（§9） |

## 6. 迁移债台账

初版所列 D1-D10 已全部闭环（2026-07-18 架构优化收官，见 `docs/memory/arch-optimization-roadmap.md`）：双 CRD 树/Actor/翻译引擎随 Stack A 删除，plugin 与 gNMI 空壳物理删除（扩展点由真实需求驱动再设计），NativeDeviceConfig 通道由 Stack B 直连取代，netsim 收敛为 netconfsim。当前无未决架构债；活跃工作见 `openspec/tasks/`。

## 7. 关联文档

- 北向 API：`openspec/specs/{devices,config,yang}-api/`、`config-changeset`
- 部署约束：`docs/memory/k8s-paas-deployment-constraints.md`（多实例/禁本地存储拍板）
- 工作流与红线原文：根 `CLAUDE.md`
