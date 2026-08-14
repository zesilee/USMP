// UI 适配层 · 图标收口（FA-04，R12 禁 emoji 顶替图标）：以旧 Element Plus 图标
// 语义名为契约重导出（重建参考 rebuild-notes/data-test-inventory 同期的 19 个
// 在用图标），业务代码只认语义名——换库只改右侧映射。
// 缺失图标的规范占位 = QuestionCircleOutlined（PlaceholderIcon），SHALL NOT 空白。
export {
  DownOutlined as ArrowDownIcon,
  UpOutlined as ArrowUpIcon,
  BellOutlined as BellIcon,
  ApiOutlined as ConnectionIcon,
  LineChartOutlined as DataLineIcon,
  DeleteOutlined as DeleteIcon,
  FileTextOutlined as DocumentIcon,
  MenuUnfoldOutlined as ExpandIcon,
  MenuFoldOutlined as FoldIcon,
  KeyOutlined as KeyIcon,
  DesktopOutlined as MonitorIcon,
  PlusOutlined as PlusIcon,
  SyncOutlined as RefreshIcon,
  RedoOutlined as RefreshRightIcon,
  SearchOutlined as SearchIcon,
  SettingOutlined as SettingIcon,
  ShareAltOutlined as ShareIcon,
  ToolOutlined as ToolsIcon,
  ExclamationCircleFilled as WarningFilledIcon,
  QuestionCircleOutlined as PlaceholderIcon,
} from '@ant-design/icons'
