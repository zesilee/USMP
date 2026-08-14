// UI 组件适配层（frontend-ui-adapter spec：FA-01~04，design D2）。
//
// 契约：业务代码（components/views/hooks/stores）SHALL 只从本目录导入界面控件，
// SHALL NOT 直接 import 组件库包名（守护测试 test/ui/adapter-guard 拦截）。
// 导出面逐项显式声明、只覆盖项目实际使用的控件（FA-02 禁整包透传）；新增控件
// 先在此补导出再由业务引用。当前实现 = antd 6；将来换库（如 EviewUI）只改本目录。
//
// 薄转发边界（FA-02）：仅做重导出 / 已知 API 差异抹平（feedback 的 Promise 化）/
// 图标与主题令牌收口，SHALL NOT 承载业务语义。

// ===== 布局与导航 =====
export { Menu, Breadcrumb, Tabs, Dropdown, Badge } from 'antd'

// ===== 数据展示 =====
export { Table, Tag, Tooltip, Popover, Empty, Alert, Tree, Pagination } from 'antd'
export type { TableProps, TableColumnType } from 'antd'
export type { TreeDataNode } from 'antd'

// ===== 表单与录入 =====
export {
  Form,
  Input,
  InputNumber,
  Select,
  Radio,
  Checkbox,
  Switch,
  Segmented,
  Button,
} from 'antd'
export type { FormInstance, FormRule } from 'antd'

// ===== 反馈与弹层 =====
export { Modal, Drawer, Spin } from 'antd'
export { toast, confirm } from './feedback'
export type { ConfirmOptions } from './feedback'

// ===== 根装配 =====
export { UiProvider } from './provider'

// ===== 图标（语义名，R12）=====
export * as icons from './icons'

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
