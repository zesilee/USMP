# system-architecture — delta (retire-ygot-runtime)

## MODIFIED Requirements

### Requirement: SC-04 模型驱动渲染

新增 YANG 模块并在前端展示配置时，前端 SHALL 由模型 schema 自动渲染表单，SHALL NOT 手写固定表单（R05）；对应 YANG 结构 SHALL 由自研生成器自动生成（构建期工具 MAY 使用 goyang 解析 YANG 源），SHALL NOT 手写 YANG 结构体（R04）。

#### Scenario: 新增模块零手写表单
- **WHEN** 接入一个新 YANG 模块并在前端展示
- **THEN** 前端 SHALL 依模型 schema 自动渲染表单，后端结构 SHALL 由自研生成器生成，二者均无手写固定内容

## ADDED Requirements

### Requirement: SC-07 发布二进制零外部 YANG 运行库

商用发布二进制（`usmp-backend`）的 import 闭包 SHALL NOT 含 `github.com/openconfig/ygot` 与 `github.com/openconfig/goyang` 任何子包（商用自主可控要求）；YANG 类型系统、RFC7951 编解码与 Schema 加载 SHALL 由自研运行库承载（yang-native-runtime）。两库 SHALL 仅存在于构建期工具、测试与 simulator 依赖面。守护测试 SHALL 以 import 闭包审计拦截回引（同 NC-01 scrapligo 禁回引口径）。

#### Scenario: 发布二进制依赖审计
- **WHEN** 对 `backend/main.go` 的 import 闭包做依赖审计（`go list -deps` 或 `go version -m` 口径）
- **THEN** SHALL NOT 出现 openconfig/ygot 与 openconfig/goyang 任何包

#### Scenario: 运行时包回引被拦截（负路径）
- **WHEN** 任一运行时包新增 ygot/goyang import 并提交
- **THEN** 守护测试 SHALL fail 并指明违规包路径，CI 拦截合入
