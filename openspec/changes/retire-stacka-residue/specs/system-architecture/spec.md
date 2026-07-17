## MODIFIED Requirements

### Requirement: SC-01 单一权威栈与单一进程入口

系统 SHALL 以 Stack B（yang-controller-runtime）作为**唯一权威且唯一生产栈**（R01），生产进程 SHALL 只有一个入口 `backend/main.go`（真·单进程）。原 Stack A（K8s CRD + Actor）入口 `cmd/controller` 已删除，SHALL NOT 存在第二条生产运行路径。Stack A 代码与载体（`pkg/yang-runtime/actor` 包、`api/biz/v1` 旧意图 CRD 类型、`api/core/v1` 的 NativeDeviceConfig 类型、`internal/crdsource` 桥接、`pkg/translator` 翻译引擎、`backend/{deploy,config}` 部署清单目录、`backend/test/{e2e,integration}` B0 测试层）已全部物理删除，SHALL NOT 重新引入。

#### Scenario: 生产启动入口唯一
- **WHEN** 启动生产进程
- **THEN** SHALL 经由 `backend/main.go` 单一入口拉起 yang-controller-runtime（Manager→Controller→Reconciler→Source），不存在 Stack A 入口

#### Scenario: Stack A 载体物理不存在
- **WHEN** 审计代码库
- **THEN** `pkg/yang-runtime/actor`、`backend/api/biz/v1`、`internal/crdsource`、`pkg/translator`、`api/core/v1` 的 NativeDeviceConfig 类型、`backend/deploy`、`backend/config`、`backend/test/e2e`、`backend/test/integration` SHALL NOT 存在（历史契约由 actor-transaction/business-crd/translation-engine 的 LEGACY spec 承载）
