# Delta: frontend

## ADDED Requirements

### Requirement: FE-25 列表 Tab 双模式分页（阈值自适应服务端分页）

列表 Tab（含 FE-14 只读状态 list）首次读取 SHALL 携 `limit=200&offset=0`（只读状态 Tab 另携 `include_state=true`）自适应选择模式：

- `total ≤ 200`：SHALL 进入**纯前端模式**——已获全量行，过滤/分页/排序在本地完成，交互与 FE-11 现状完全一致，SHALL NOT 产生额外网络往返；
- `total > 200`：SHALL 进入**服务端模式**——翻页、每页条数、高级搜索（support-filter 字段集映射为 `filter` 参数，等值/包含语义与后端一致）与排序 SHALL 映射为查询参数重新请求；表格下方总记录数 SHALL 取响应 `total`；请求期间表格 SHALL 呈现 loading 态（覆盖快照过期重拉的延迟尖刺）。

两种模式下 FE-11 既有 UI 元素（工具区/列设置/操作列/分页器/完成时刻）SHALL 保持不变；变更集攒批的 pending create 行 SHALL 仍在本地叠加展示且 SHALL NOT 计入服务端 `total`；「获取数据源」SHALL 以 `force_refresh=true` 全量重拉并复位到第一页。

#### Scenario: 小表维持纯前端零回归

- **WHEN** list 总行数 66（≤200），用户翻页与本地搜索
- **THEN** SHALL NOT 发起新的 /config 请求，行为与现状一致

#### Scenario: 大表翻页走服务端

- **WHEN** total=12000 的状态 list 处于服务端模式，用户跳转第 5 页
- **THEN** SHALL 携 `limit/offset` 重新请求并渲染返回的 rows，总记录数展示 12000

#### Scenario: 服务端模式高级搜索下推

- **WHEN** 服务端模式下在高级搜索面板提交「name 包含 GE」
- **THEN** SHALL 携 `filter=name~=GE` 重新请求且页码复位第一页，SHALL NOT 在本地对当前页再过滤

#### Scenario: pending create 行本地叠加（边界）

- **WHEN** 服务端模式下变更集含该 list 的未提交新建行
- **THEN** 新建行 SHALL 叠加展示于当前页且带待提交标识，服务端 total SHALL 不含该行

#### Scenario: 获取数据源复位（边界）

- **WHEN** 服务端模式第 5 页点击「获取数据源」
- **THEN** SHALL 携 `force_refresh=true` 重拉、页码复位第一页并刷新新鲜度
