# Proposal: business-network-config

## Why

USMP 已交付「原生配置」全链路（YANG 模型驱动的单设备单模块配置管理），但 PaaS 底座的真实运维单元是**网络业务**（如「业务 VLAN 打通」）——一个业务动作横跨多设备、多模块，靠人工逐设备下原生配置既低效又易错。本变更引入「业务网络配置」意图层：业务能力用 YANG 模型定义，USMP 将意图编排为原生配置下发，且意图作为 K8s CRD 成为 PaaS 的公共契约（USMP 部署于 K8s、必须多实例、禁止本地存储——2026-07-15 拍板）。

## What Changes

- **意图持久化 = K8s CRD**：意图实例存 apiserver（etcd），CRD 仅作持久化载体 + watch 事件源（C4 Source），Stack B 对账架构（R01）不变；禁止复活 Stack A 式「CRD 当架构」（双重建模/Actor/前端消费 CRD）。
- **CR 是对外公共契约**：kubectl/GitOps 直写受支持 → 新增构建期 **YANG→CRD OpenAPI codegen 管线**（含 CRD 版本策略、安装编排、golden 测试）；when/must 跨字段约束由现有约束引擎在收敛侧兜底。
- **意图编排控制器**：意图 Reconciler 将 CR 展开为 N×(设备, 原生模块, 配置片段)；**跨设备两阶段提交**下发（NETCONF candidate + confirmed-commit，任一 prepare 失败全体 discard），事务成功后才写 desired ConfigStore，稳态漂移修复走每设备声明式对账。
- **生命周期完整闭环**：finalizer 删除展开为 DELETE 命令通道；持久化上一代展开结果做收缩差集（意图 devices 列表移除设备时清理孤儿配置）；generation/observedGeneration 代际追踪；每设备收敛状态存 CR status。
- **软归属标记**：展开认领记录 (设备, 路径) 存 CR status；原生控制台显示「由业务配置 X 管理」徽标；config-api 手改命中认领路径返回警告不拦截（硬锁留二期）。
- **首个业务能力：跨设备 VLAN 打通**——意图 = {vlan-id, name, devices[]{ip, access 口[], trunk 口[]}}，每设备展开 huawei-vlan + huawei-ifm。
- **前端**：意图 YANG 经 task-name 新 category 自动出现「业务网络配置」菜单组（零菜单代码）；新增**平台作用域**业务控制台（现有模块控制台是设备作用域）。
- **多实例就绪**（非全局 HA）：意图层无本地状态、状态存 CR status、预留 leader election 接缝；全局 HA（device store 上共享存储、全控制器 leader election、audit 出本地文件）另立任务。
- **模拟网元**：netconfsim 补 candidate/confirmed-commit/discard-changes 能力以支撑 2PC 集成测试。
- **BREAKING（架构措辞）**：system-architecture spec 的 R03/§8「元信息=本地 JSON」修订为「K8s CRD 载体」（多实例禁止本地存储）；任务文件「禁止复活 CRD 通道」重新表述为「禁止 CRD 当架构，允许当持久化载体」。

## Capabilities

### New Capabilities

- `business-intent-crd`: 意图 CRD 公共契约——YANG→CRD OpenAPI codegen 管线、CRD 版本策略与安装、CR 信封/status 子资源结构、直写校验分层（apiserver OpenAPI + 约束引擎兜底 + Validated 条件）。
- `business-intent-orchestration`: 意图编排——CRD watch 事件源、意图 Reconciler 展开、跨设备 2PC 下发、生命周期（finalizer 删除/收缩差集/代际追踪）、软归属认领、每设备收敛状态呈现、多实例就绪。
- `business-vlan-service`: 首个业务能力「跨设备 VLAN 打通」——意图 YANG 模型字段、每设备 huawei-vlan+huawei-ifm 展开规则、端到端验收。

### Modified Capabilities

- `system-architecture`: 持久化契约修订——元信息存储从「本地 JSON」改为「K8s CRD（etcd via apiserver）」；新增多实例部署约束（禁止本地存储）。
- `device-protocol`: NETCONF 客户端新增 2PC 原语要求——candidate 锁定、confirmed-commit（带超时回滚）、discard-changes、confirming commit。
- `netconf-simulator`: 新增 candidate datastore 与 confirmed-commit/discard-changes 仿真能力。
- `config-api`: SetConfig 命中意图认领路径时返回归属警告（响应体新增字段，不拦截）。
- `frontend`: 新增「业务网络配置」菜单组（category 驱动）与平台作用域业务控制台；原生模块控制台新增归属徽标。

## Impact

- **后端**：新增意图 YANG 模型（`backend/internal/yang/models/` 或独立业务模型目录）+ gen.conf 条目；新增 YANG→CRD codegen 工具（`backend/tools/`）；泛化 `internal/crdsource` 为业务意图 watch 源；新增意图控制器/编排包；`pkg/yang-runtime/client` 增补 2PC 原语；config-api 归属警告。
- **前端**：菜单组自动分桶（存量机制）、平台作用域控制台页面（F2/F3/F4 测试层跟进）、归属徽标。
- **部署**：CRD manifest 安装编排（helm/初始化脚本）、RBAC（USMP ServiceAccount 读写业务 CR + Lease）。
- **依赖**：client-go/controller-runtime v0.19 已在 go.mod，无新增第三方依赖（R10 合规）。
- **测试**：T02b `yang-config-test-design` 完备矩阵强制触发；netconfsim 2PC 能力先行（B2 集成测试前置）；覆盖率棘轮同步上调（T08）。
- **legacy 边界**：`backend/api/biz/v1` 与 `pkg/translator` 仅作思想参考，不复用其 CRD 类型；`internal/crdsource` 按存量改造策略渐进泛化（旧 BusinessVlan 桥接并行保留→切换→退役）。
