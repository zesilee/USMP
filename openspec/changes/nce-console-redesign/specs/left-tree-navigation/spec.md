# left-tree-navigation — delta（nce-console-redesign）

## ADDED Requirements

### Requirement: LT-05 左树搜索与一键展开收起

前端左树顶部 SHALL 提供节点名搜索框与「展开全部/收起全部」控件。搜索 SHALL 为客户端过滤：按当前语言标签与原始节点名（zh/en/name 三口径，大小写不敏感子串）递归匹配全树节点（分组/模块叶/container/rpc 均参与）；命中节点 SHALL 保留其祖先链并自动展开到命中层，未命中分支 SHALL 隐藏；清空搜索 SHALL 恢复全树与默认折叠态。搜索过滤 SHALL NOT 改变节点可用性语义（不可用叶命中仍为禁用占位态）。SHALL NOT 变更左树生成物与查询接口契约（LT-01/LT-02 不动）。

#### Scenario: 搜索命中自动展开

- **WHEN** 输入「通用接口」（或 en 环境输入 "Common Interface"）
- **THEN** 树 SHALL 仅显 接口管理 → 接口基础 → huawei-ifm → 通用接口 祖先链并展开到命中节点，其余分支隐藏；清空后 SHALL 恢复全树默认折叠

#### Scenario: 一键展开与收起

- **WHEN** 点击「展开全部」/「收起全部」
- **THEN** 全树分组与模块叶 SHALL 全部展开 / 全部收起

#### Scenario: 命中不可用叶（负路径）

- **WHEN** 搜索词命中 `available:false` 的叶
- **THEN** 该叶 SHALL 出现在结果中且保持禁用占位态，SHALL NOT 因搜索变为可点

#### Scenario: 无命中（边界）

- **WHEN** 搜索词与任何节点不匹配
- **THEN** 树区 SHALL 展示空态提示，SHALL NOT 空白或崩溃（R08）
