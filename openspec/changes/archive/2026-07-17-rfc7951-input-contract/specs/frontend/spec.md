## MODIFIED Requirements

### Requirement: FE-03 配置下发主链路（Stack B 直连）

原生配置 SHALL 走 Stack B 直连主链路：通用模块控制台（ModuleListTab/ModuleFormTab）以 YANG schema 渲染模型驱动表单，编辑→校验通过→提交时 SHALL 经 `useConfigSubmit`（或表单直调）`POST /api/v1/config/:ip/*path`，请求体为以 path 为根的 RFC7951 子树（YANG 真名、枚举名字符串；list 流的包裹键跟随回读的 RFC7951 键），随后 SHALL 以 `force_refresh` 强制回读实际态、轮询单设备 reconcile 结局，驱动 pushing→reading→converged/drifted/error/timeout 进度。下发失败 SHALL 降级、不误报成功（R08）。历史专用页 `DeviceConfigPage.vue` 已物理删除（通用模块控制台 FE-10~FE-16 取代）。

#### Scenario: 编辑并下发触发对账
- **WHEN** 用户在模块控制台提交一条合法（校验通过）配置
- **THEN** SHALL `POST /config` 下发 → `force_refresh` 回读 → 轮询 `getDeviceReconcile`，直到出现推进过 baseline 的终态（收敛/漂移/失败）或超时

#### Scenario: 下发失败降级
- **WHEN** `setConfig` 报错或返回失败信封
- **THEN** SHALL 置 error 相位、SHALL NOT 重读列表、保留原表单，不崩溃（R08）

#### Scenario: 对账超时
- **WHEN** 轮询达到上限仍无终态
- **THEN** SHALL 标注 `timedOut` 停在 reading 相位，SHALL NOT 误报成功
