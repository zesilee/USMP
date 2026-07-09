# frontend delta — config-delete-semantics

## ADDED Requirements

### Requirement: FE-16 列表行删除（confirm→DELETE→刷新）

通用模块控制台列表 Tab 的行「删除」按钮在门禁允许（list 级 `operationExclude` 不含 delete 且非只读 Tab）时 SHALL 可用；点击 SHALL 弹出二次确认（含条目主键标识），确认后 SHALL 调用 `DELETE /config/:ip/*path?key=<主键>`；成功 SHALL 刷新列表与新鲜度并提示，失败 SHALL 如实展示后端错误且列表不变（R08/§9）。取消确认 SHALL 无任何请求。

#### Scenario: 删除成功流

- **WHEN** 用户点击某行删除并确认
- **THEN** SHALL 以该行主键调用 DELETE，成功后该行 SHALL 从列表消失（重新拉取）

#### Scenario: 取消确认

- **WHEN** 用户在确认框选择取消
- **THEN** SHALL NOT 发起任何请求，列表不变

#### Scenario: 删除失败如实透出（负路径）

- **WHEN** 后端返回错误（如设备 data-missing / 门禁 400）
- **THEN** SHALL 展示错误信息，列表 SHALL 保持原状

#### Scenario: 门禁不可用态

- **WHEN** list 级 `operationExclude` 含 delete 或 Tab 为只读
- **THEN** 删除按钮 SHALL 不可用/不渲染（沿用 FE-11/FE-14 门禁）
