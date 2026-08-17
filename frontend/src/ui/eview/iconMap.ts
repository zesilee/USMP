// EviewUI 图标映射表（FA-04 / 组 3.4）：23 个语义图标名 → @nce/icon-plus 组件名。
// 依据：icon-plus 1.0.87 d.ts 全量比对（调研矩阵，21 直接对应 + 2 近似）。
// 实心变体经 props type="filled" 表达（如 WarningFilled），此处只记组件名；
// 桥实现按本表 import { IconPlus<名> } from '@nce/icon-plus'。
export const ICON_MAP: Record<string, { name: string; filled?: boolean; approx?: boolean }> = {
  ArrowDownIcon: { name: 'IcPublicChevronDown' },
  ArrowUpIcon: { name: 'IcPublicChevronUp' },
  BellIcon: { name: 'IcPublicNotice', approx: true }, // 无纯铃铛，通知铃近似
  ConnectionIcon: { name: 'IcIctApi' },
  DataLineIcon: { name: 'IcIctCurveChart' },
  DeleteIcon: { name: 'IcPublicTrash' },
  DocumentIcon: { name: 'IcPublicGenericFile' },
  ExpandIcon: { name: 'IcPublicMenuExpansion' },
  FoldIcon: { name: 'IcPublicMenuCollapse' },
  KeyIcon: { name: 'IcPublicKey' },
  MonitorIcon: { name: 'IcPublicScreenWifi', approx: true }, // 无纯显示器
  PlusIcon: { name: 'IcPublicPlus' },
  RefreshIcon: { name: 'IcPublicRefreshClockwise' },
  RefreshRightIcon: { name: 'IcPublicRedo' },
  SearchIcon: { name: 'IcPublicSearch' },
  SettingIcon: { name: 'IcPublicSetting' },
  ShareIcon: { name: 'IcPublicShare' },
  ToolsIcon: { name: 'IcPublicWrench' },
  WarningFilledIcon: { name: 'IcPublicWarning', filled: true },
  CheckIcon: { name: 'IcPublicCheckmark' },
  CloseIcon: { name: 'IcPublicClose' },
  PlaceholderIcon: { name: 'IcPublicQuestionmarkCircle' },
}
