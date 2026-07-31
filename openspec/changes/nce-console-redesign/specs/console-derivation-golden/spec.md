# console-derivation-golden — delta（nce-console-redesign）

## MODIFIED Requirements

### Requirement: GD-01 全模块控制台派生黄金覆盖

前端 SHALL 对 **fixture 目录中的全部模块**运行既有控制台派生纯函数（Tab 派生、list 主键派生、列派生——含**默认显示集与可用列全集双维度**、可过滤字段派生、**详情二级 Tab 派生（deriveDetailTabs）**、架构树派生）与字段级派生（控件类型、必填、只读、约束存在性、单位），并把派生结果钉为黄金快照。

模块集合 SHALL 由 fixture 目录内容动态发现，SHALL NOT 硬编码模块名单或数量。任一模块的派生结果与黄金不符时，测试 SHALL 失败。

派生 SHALL 在纯函数层完成，SHALL NOT 依赖浏览器、docker 或运行中的后端。

#### Scenario: 全部 fixture 模块参与派生比对

- **WHEN** fixture 目录含 N 个模块，执行黄金套件
- **THEN** 系统 SHALL 对全部 N 个模块执行派生并与各自黄金比对
- **AND** 任一模块缺失对应黄金时 SHALL 失败

#### Scenario: 新增模块自动纳入覆盖

- **WHEN** fixture 目录新增一个模块的 fixture，黄金套件代码未作任何修改
- **THEN** 执行黄金套件 SHALL 报告该模块缺少黄金而失败，直至为其生成黄金

#### Scenario: 详情二级 Tab 派生入黄金

- **WHEN** 某模块 list 节点含嵌套 group/list 子节点，执行黄金套件
- **THEN** 该模块黄金 SHALL 记录 deriveDetailTabs 结论（主表单 Tab + 各嵌套子 Tab 的 name/kind 序列），派生逻辑变更 SHALL 震动对应模块黄金

#### Scenario: 无浏览器依赖

- **WHEN** 在无 Chromium、无 docker、无后端服务的环境执行黄金套件
- **THEN** 套件 SHALL 正常执行并给出结论
