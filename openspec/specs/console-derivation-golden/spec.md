# console-derivation-golden Specification

## Purpose
TBD - created by archiving change schema-driven-test-harness. Update Purpose after archive.
## Requirements
### Requirement: GD-01 全模块控制台派生黄金覆盖

前端 SHALL 对 **fixture 目录中的全部模块**运行既有控制台派生纯函数（Tab 派生、list 主键派生、列派生、可过滤字段派生、架构树派生）与字段级派生（控件类型、必填、只读、约束存在性、单位），并把派生结果钉为黄金快照。

模块集合 SHALL 由 fixture 目录内容动态发现，SHALL NOT 硬编码模块名单或数量。任一模块的派生结果与黄金不符时，测试 SHALL 失败。

派生 SHALL 在纯函数层完成，SHALL NOT 依赖浏览器、docker 或运行中的后端。

#### Scenario: 全部 fixture 模块参与派生比对

- **WHEN** fixture 目录含 N 个模块，执行黄金套件
- **THEN** 系统 SHALL 对全部 N 个模块执行派生并与各自黄金比对
- **AND** 任一模块缺失对应黄金时 SHALL 失败

#### Scenario: 新增模块自动纳入覆盖

- **WHEN** fixture 目录新增一个模块的 fixture，黄金套件代码未作任何修改
- **THEN** 执行黄金套件 SHALL 报告该模块缺少黄金而失败，直至为其生成黄金

#### Scenario: 无浏览器依赖

- **WHEN** 在无 Chromium、无 docker、无后端服务的环境执行黄金套件
- **THEN** 套件 SHALL 正常执行并给出结论

### Requirement: GD-02 黄金内容只含派生结论

黄金 SHALL 只记录派生**结论**，SHALL NOT 包含 schema 原文的副本，SHALL NOT 包含 i18n 本地化后的标签文案。

- 不含 schema 原文：schema 自身的变化已由 fixture 漂移门禁（SF-04）覆盖，重复记录会使黄金变动丧失信息量。
- 不含本地化结果：本地化是渲染管线的独立一层，纳入黄金会使 i18n 资源更新震动全部黄金，制造与派生逻辑无关的噪声。标签 SHALL 以后端给出的原始 YANG 节点名形态参与。

#### Scenario: schema 语义无关变更不震动黄金

- **WHEN** 某模块 schema 发生不影响任何派生结论的变化
- **THEN** 该模块黄金 SHALL 保持不变（该变化由 SF-04 fixture 门禁承担）

#### Scenario: i18n 资源更新不震动黄金

- **WHEN** SND i18n 本地化资源更新导致界面标签文案变化
- **THEN** 全部模块黄金 SHALL 保持不变

#### Scenario: 派生逻辑变更震动黄金

- **WHEN** 列派生、主键派生或控件类型映射等派生逻辑被修改
- **THEN** 受影响模块的黄金比对 SHALL 失败

### Requirement: GD-03 派生变化可定位到模块

黄金 SHALL 按模块分文件存储，一个模块一份，SHALL NOT 合并为单一巨型快照。黄金格式 SHALL 为结构化、键有序的可 diff 文本。

目的是让前端派生逻辑的合法变更可被有效评审：变更影响面 SHALL 能以「受影响模块清单」形式呈现，而非单一大文件的整体变动。

#### Scenario: 局部派生变更只影响相关模块文件

- **WHEN** 一处派生逻辑变更仅影响部分模块的派生结论
- **THEN** 仅这些模块的黄金文件 SHALL 发生变化，未受影响模块的黄金文件 SHALL 逐字节不变

#### Scenario: 变更影响面可枚举

- **WHEN** 黄金因派生逻辑变更被刷新
- **THEN** 受影响模块 SHALL 可由变更的黄金文件名直接枚举

### Requirement: GD-04 黄金保证边界的诚实声明

黄金门禁 SHALL 只声称它实际证明的范围：**「schema → 控制台形态的派生是确定的，且未发生非预期变化」**。

黄金 SHALL NOT 被表述为证明派生结果对用户合理、控件可用或视觉正确。门禁产出的报告与相关文档 SHALL NOT 以覆盖模块数暗示强于上述范围的保证。首次生成黄金时的人工审阅是唯一一次语义确认，此后黄金只承担回归防线职责。

#### Scenario: 门禁报告不夸大保证范围

- **WHEN** 黄金套件全绿并产出报告
- **THEN** 报告 SHALL 表述为「派生结构无非预期变化」，SHALL NOT 表述为「全部模块功能验证通过」或等价说法

#### Scenario: 语义错误不在黄金职责内

- **WHEN** 某模块的派生结论稳定但对用户不合理（如列选取不当）
- **THEN** 黄金比对 SHALL 通过——该类问题由首次人工审阅与后续视觉层验证承担，不构成黄金门禁的失效

