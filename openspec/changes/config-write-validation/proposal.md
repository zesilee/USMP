# Proposal: config-write-validation

## Why

原生配置下发路径（`POST /api/v1/config/:ip/*path`）目前**几乎没有服务端约束校验**：
唯一的后端拦截是 `internal/api/config_handler.go` 里硬编码的 `validateConfig`，
它只认华为 VLAN 一个模型的 ID 范围（1–4094），其余全部模块的配置一律放行。

与此同时，仓库里已有一个通用的、Schema IR 驱动的校验器
`pkg/yang-runtime/validate`（YN-04：pattern 锚定全匹配 / range / length /
min-elements），但**只有意图层（`internal/intent/cr.go`）在消费**。原生配置这条
用得最多的路径反而没接。

后果是：违反 YANG 约束的配置（名称格式不合 pattern、数值越界、字符串超长）会被
原样写入 desired 并下发到设备，由设备侧拒绝——错误来得晚、报文已经发出去、
`<rpc-error>` 的措辞是设备方言，前端拿到的提示远不如本地约束提示清楚。
前端表单虽有同源约束校验，但那是 UI 层防线，绕过前端直调 API 就没有任何拦截。

另有一处相关的历史遗留：`schema.Schema` 接口上的 `Validate(path, config)` 方法
是**空壳**——只查路径存在性就 `return nil`，且全仓零调用方。它长得像一道防线，
实际什么都不做，比缺了这个方法更容易误导人。空壳的成因是**包依赖方向**：
`validate` 包 import `schema` 包，`schema` 反过来调 `validate` 会构成循环依赖。

## What Changes

- **校验逻辑迁入 `schema` 包**：把 `pkg/yang-runtime/validate` 的实现整体移入
  `pkg/yang-runtime/schema`（导出为 `schema.ValidateObject`），解开循环依赖。
  语义冻结契约（mandatory 不校验 / min-elements 仅对存在的空 list 生效 /
  must·when 不做运行时求值）随实现一并迁移，**一字不改**。
- **`DefaultSchema.Validate` 填实**：委托到迁入后的实现，不再是空壳。
- **删除 `validate` 包**，三处调用点（`internal/intent/cr.go` 与两个测试）改指
  新位置。意图层校验行为**逐字节不变**。
- **配置写路径接入校验**：`SetConfig` 在类型转换成功之后、写入 desired 之前，
  以请求 path 解析 schema 节点并跑约束校验；**违反即返回 `code=400` 拒绝**，
  SHALL NOT 写 ConfigStore、SHALL NOT 触发对账、SHALL NOT 触达设备。
- 既有的 VLAN ID 硬编码校验**保留**：VLAN ID 范围没有编码进 YANG 模型，
  通用校验器覆盖不到它，两者是互补而非重复。

## Capabilities

### Modified Capabilities

- `config-api`: BR-08「无效请求拒绝」扩展——除非法 JSON / 类型转换失败外，
  新增「违反 YANG 约束」这一类 400 拒绝，并明确拒绝时零副作用。
- `yang-native-runtime`: YN-04 措辞由「现仅意图层消费」更新为「意图层与原生配置
  写路径共同消费」，并记录实现位置由独立 `validate` 包迁入 `schema` 包。

## Impact

- **代码**：`pkg/yang-runtime/schema/`（迁入校验实现 + 填实 `Validate`）、
  `pkg/yang-runtime/validate/`（删除）、`internal/api/config_handler.go`（接入）、
  `internal/intent/cr.go`（改 import）。
- **行为变更（用户可感知）**：以前能提交成功、到设备侧才失败的非法配置，
  现在会在后端直接被 400 拒绝。这是本次改动的**目的**，但也意味着
  **可能拒绝以前"能过"的配置**——凡是被拒的，都是原本也到不了设备、
  或会在设备上产生误配的输入。
- **不受影响**：读路径、删除通道（BR-09）、对账收敛语义、意图层校验结论、
  前端表单校验（同源 IR，结论一致）、`/yang/schema` 契约。
- **风险与兜底**：主要风险是校验过严误伤存量配置。以「全量测试 + 模拟网元
  端到端 + e2e 冒烟」三层暴露拒绝面；任何一处存量用例被拒都要逐条判断是
  真非法还是校验器过严，判断结论写入 tasks。语义冻结契约不得借本次收紧
  （收紧属独立契约变更，须另行拍板）。
- **测试**：B1（校验器迁移后行为等价 + 正常/越界/pattern/长度/边界）、
  B3（API 层拒绝路径 + 零副作用断言）、B2（合法配置端到端仍下发成功）。
