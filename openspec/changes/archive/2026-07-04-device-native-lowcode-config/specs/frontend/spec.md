## ADDED Requirements

### Requirement: 设备原生配置页由动态 YANG schema 驱动

设备原生模块的配置页 SHALL 以 `GET /api/v1/yang/schema/:module` 返回的**动态** YANG schema 为源，经当前活跃的通用低码引擎（`DynamicForm`/`FieldRenderer`/`DynamicTable`）渲染表单/表格（R05：仅定义通用控件，不手写固定表单）。SHALL NOT 为设备原生面新建第二套渲染引擎。

#### Scenario: 动态 schema 渲染设备配置
- **WHEN** 用户打开某设备原生模块配置页
- **THEN** 前端 SHALL 拉取该模块的动态 YANG schema 并经通用低码引擎渲染其可配置属性

#### Scenario: 编辑联动下发
- **WHEN** 用户编辑并提交表单
- **THEN** 前端 SHALL 经 config-api 提交，语义为「已接受、异步下发」，并可展示设备/缓存/下发/异常状态

### Requirement: 退役设备侧静态 YANG 死代码

设备侧静态 YANG 渲染路径（`components/yang/*` + `types/yang-schema.ts`）SHALL 在动态路径切换验证通过后移除，消除双代动态表单（迁移债 D9 设备侧）。

#### Scenario: 切换后清理
- **WHEN** 动态 YANG schema 驱动路径经双路径验证通过并成为唯一活跃路径
- **THEN** 设备侧静态 YANG 组件 SHALL 被删除，且构建/路由无残留引用
