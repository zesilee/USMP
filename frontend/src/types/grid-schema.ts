export type GridWidgetType = 'text' | 'number' | 'select' | 'switch' | 'textarea' | 'table'

export interface GridSchema {
  schemaVersion: string
  module: string
  targetPath: string
  capabilitySource: string
  layout: GridLayout
  sections: GridSection[]
  widgets: GridWidget[]
  values: Record<string, any>
}

export interface GridLayout {
  type: string
  columns: number
  gap: string
}

export interface GridSection {
  id: string
  title: string
  description?: string
  widgets: string[]
}

export interface GridWidget {
  id: string
  type: GridWidgetType
  label: string
  help?: string
  rowKey?: string
  grid: WidgetGrid
  columns?: GridColumn[]
  binding?: Record<string, any>
  disabled?: boolean
  disabledReason?: string
}

export interface WidgetGrid {
  span: number
  offset?: number
  order?: number
}

export interface GridColumn {
  id: string
  type: GridWidgetType
  label: string
  placeholder?: string
  readonly?: boolean
  options?: GridOption[]
  validation?: GridValidation
}

export interface GridOption {
  label: string
  value: string | number | boolean
}

export interface GridValidation {
  required?: boolean
  min?: number
  max?: number
  minLength?: number
  maxLength?: number
}

export interface InterfaceGridApplyPayload {
  schemaVersion: string
  values: Record<string, any>
}

export interface InterfaceGridApplyResult {
  schemaVersion: string
  values?: Record<string, any>
  lastSync?: string
}
