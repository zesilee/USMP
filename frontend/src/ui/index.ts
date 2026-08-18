// UI 组件适配层（frontend-ui-adapter spec：FA-01~06，design D2）。
//
// 契约：业务代码（components/views/hooks/stores）SHALL 只从本目录导入界面控件，
// SHALL NOT 直接 import 组件库包名（守护测试 test/ui/adapter-guard 拦截）。
// 导出面逐项显式声明、只覆盖项目实际使用的控件（FA-02 禁整包透传）；新增控件
// 先在此补导出再由业务引用。
//
// 当前实现 = EviewUI 桥（组 5 接线，对外恒 antd 形态——调用点零改动），经
// @ui-backend 裸别名单点切换：生产/typecheck → ./eview（vite.config 与
// tsconfig paths）；外网 vitest → ./antd-backend 测试镜像（EviewUI 实现包
// 不出内网，见其 README）；EVIEW_REAL=1（内网校准）→ ./eview 全链真身。
//
// 薄转发边界（FA-02）：仅做重导出 / 已知 API 差异抹平（feedback 的 Promise 化）/
// 图标与主题令牌收口，SHALL NOT 承载业务语义。

// ===== 布局与导航 =====
export { Tabs, Menu } from '@ui-backend/components/structure'
export { Breadcrumb, Badge } from '@ui-backend/components/display'
export { Dropdown } from '@ui-backend/components/controls'

// ===== 数据展示 =====
export { Table } from '@ui-backend/components/structure'
export type { TableColumnType } from '@ui-backend/components/structure'
export { Tag, Empty } from '@ui-backend/components/display'
export { Tooltip, Popover, Alert } from '@ui-backend/components/rest'

// ===== 表单与录入 =====
export { Input, InputNumber, Select } from '@ui-backend/components/inputs'
export { Radio, Checkbox, Switch, Segmented } from '@ui-backend/components/controls'
export { Button } from '@ui-backend/components/rest'
export { default as FormItemShell } from './eview/FormItemShell'
export type { FormItemShellProps } from './eview/FormItemShell'

// ===== 反馈与弹层 =====
export { Modal, Drawer } from '@ui-backend/components/display'
export { Spin } from '@ui-backend/components/rest'
export { toast, confirm } from '@ui-backend/feedback'
export type { ConfirmOptions } from '@ui-backend/feedback'

// ===== 根装配 =====
export { UiProvider } from '@ui-backend/provider'

// ===== 图标（语义名，R12）=====
export * as icons from '@ui-backend/icons'

// ===== 主题令牌（FA-04：业务禁硬编码色值）=====
export {
  colorPrimary,
  colorSuccess,
  colorWarning,
  colorError,
  textPrimary,
  textSecondary,
  textTertiary,
  borderColor,
  bgLayout,
  bgElevated,
} from './tokens'
