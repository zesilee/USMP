# business-crd — 前端 CRD schema 来源退役（delta）

> 附带头注更新（sync 时执行）：删除「唯一可能仍被前端使用的是 BC-05（CRD 作为前端表单 schema 来源，route/native 的 ConfigPage + useK8sCRD），据实保留」——本 change 后前端零 CRD 消费，全 spec 为纯历史契约。另注明：「业务网络配置」概念已重新定义为未来扩展层（业务侧 YANG 模型定义自动化能力，方向见 openspec/tasks/business-network-config.md），与本 legacy CRD 意图面无实现延续关系，仅思想同源。

## REMOVED Requirements

### Requirement: BC-05 CRD 作为前端表单 schema 来源

**Reason**: 前端最后的 CRD 消费链（`ConfigPage.vue` + `useConfigPage` + `useK8sCRD`，route/native 路由）已随 native-config-reposition 退役删除；本 requirement 是 business-crd spec 中唯一「可能仍被使用」的条款，现据实移除，spec 全量转为历史契约。
**Migration**: 前端表单 schema 一律来自后端 `GET /api/v1/yang/schema/*`（frontend FE-01/FE-04/FE-10），无 CRD 来源。
