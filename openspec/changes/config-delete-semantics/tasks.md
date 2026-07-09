# tasks — config-delete-semantics

> TDD 红绿循环（T01/T05/T07）：每组先测试（红）再实现（绿）。测试层按 §5.6：涉协议编解码与下发 → B1+B2 强制。
> 单 commit ≤500 行、原子功能；What/Why/How 三段式。

## 1. client：DeleteChange 删除编码（B1，DP-07）

- [ ] 1.1 B1 红：`netconf_delete_marshal_test.go`——vlan 键式删除 XML（operation 属性+仅 key 叶）、ifm 同构、未知模型报错、rpc-error 反映失败（表格驱动）
- [ ] 1.2 绿：`marshalChange` DeleteChange 分支替换死代码 stub——按 OldValue 模型路由 per-model 删除 XML builder（vlan/ifm）

## 2. 后端 API：DELETE 通道 + desired 移除 + 门禁（B1+B3，BR-09/BR-10）

- [ ] 2.1 B1 红：desired 键移除——vlan/ifm 分支、幂等（无键 no-op）、并发与合并写互斥（race）；key 解析——类型/缺失/未知路径负例
- [ ] 2.2 绿：`storeConfigDeleted`（复用 configMergeMu，新对象不原地改）+ per-model key 解析
- [ ] 2.3 B1 红：门禁判定——operation-exclude∋delete 拒绝、readonly 拒绝、schema 未覆盖放行（降级）
- [ ] 2.4 绿：`yangschema` 门禁查询接入 handler
- [ ] 2.5 B3 红：DeleteConfig 契约——成功（desired 移除+缓存失效+审计+triggered）、非法 key 400、门禁 400、下发失败透出且不失效缓存/不审计
- [ ] 2.6 绿：`DeleteConfig` handler + main.go 路由 + swagger 注解；`make gen-contract`

## 3. B2 集成：端到端删除（netconfsim）

- [ ] 3.1 B2 红：`config_delete_integration_test.go`——建 vlan→删→回读消失→二轮对账 0 change（不复活）；删不存在→data-missing 透出；ifm 接口同构；并发删+下发交错
- [ ] 3.2 绿：跑通全链路（依赖组 1/2 实现，此处修集成暴露的缺口）

## 4. 前端：启用行删除（F2，FE-16）

- [ ] 4.1 F2 红：ModuleListTab——按钮可用态（门禁允许时非 disabled）、确认→调 deleteConfig（断言 ip/path/key）→ 刷新；取消→零请求；失败→错误可见列表不变
- [ ] 4.2 绿：`src/api` 增 deleteConfig；按钮启用 + ElMessageBox confirm + 成功刷新/新鲜度 + 失败透出

## 5. 收口

- [ ] 5.1 全量验证：后端 `go test ./...`（-race）、前端单测 + vue-tsc + gen-contract 漂移、`make e2e-local`
- [ ] 5.2 覆盖率对齐棘轮（后端 57.8 / 前端 74/71/67/74），补测后按需上调
- [ ] 5.3 `go-code-review-check` + What/Why/How 提交整理 + PR 体积自检
