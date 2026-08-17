// vendor types 机制探针（混合协作模式的 typecheck 地基）：@nce/eview-react 的
// d.ts 经 tsconfig paths 映射到 vendor/eview-types（types-only，无实现 JS）——
// 外网 typecheck 可过；实现仅内网集成点存在。本文件只做类型引用，禁止值引用。
import type TableProps from '@nce/eview-react/Table/interfaces/TableProps'
import type ColumnProps from '@nce/eview-react/Table/interfaces/ColumnProps'
import type { TreeProps } from '@nce/eview-react/Tree/Tree'

export type EviewTableProps = Partial<TableProps>
export type EviewColumnProps = ColumnProps
export type EviewTreeProps = TreeProps
