import { i18n } from '../i18n'

export interface CaseDef {
  name: string
  label: string
  fields: Field[]
}

export interface Field {
  path: string
  type: 'string' | 'number' | 'boolean' | 'enum' | 'group' | 'list' | 'leaf-list' | 'choice'
  label: string
  placeholder?: string
  required?: boolean
  pattern?: string
  readonly?: boolean
  hidden?: boolean
  minimum?: number
  maximum?: number
  options?: { label: string; value: string | number }[]
  group?: string
  default?: any
  fields?: Field[]
  // cases 承载 YANG `choice` 的互斥分支（type==='choice' 时非空）；成员字段 path 扁平、
  // 与其它字段同级（sibling），前端据此渲染 Tabs/RadioGroup（FE-08）。
  cases?: CaseDef[]
  // when 携带 YANG `when` XPath 表达式（后端从 schema 透出），驱动数据驱动的条件显隐（FE-07）。
  when?: string
  // must 携带 YANG `must` 约束（XPath 表达式 + 可选提示），驱动跨字段校验（FE-07）。
  must?: { expr: string; message?: string }[]
  // supportFilter 标记厂商 support-filter 扩展：该叶可作查询条件（高级搜索，FE-11）。
  supportFilter?: boolean
  // operationExclude 承载厂商 operation-exclude 扩展（小写归一）：叶级=create-only
  // 标识字段（编辑态禁用）；list/group 级=整节点排除对应操作（FE-11）。
  operationExclude?: string[]
  // presence 标记 YANG presence 容器（type==='group'）：存在即开关（FE-12）。
  presence?: boolean
  // isKey 标记 list key 叶：通用控制台据此派生 keyField（FE-11）。
  isKey?: boolean
  // dynamicDefault 标记厂商 dynamic-default 扩展：值由系统动态缺省——空值=「设备
  // 自行决定」而非缺配置，展示自动分配占位、不强制必填、不入 payload（FE-15）。
  dynamicDefault?: boolean
  // units 携带 YANG units：输入控件展示单位后缀（FE-15）。
  units?: string
}

interface OpenAPIProperty {
  type: string
  description?: string
  enum?: (string | number)[]
  minimum?: number
  maximum?: number
  pattern?: string
  default?: any
  properties?: Record<string, OpenAPIProperty>
  required?: string[]
  'x-custom-label'?: string
  'x-custom-group'?: string
  'x-custom-placeholder'?: string
  'x-custom-readonly'?: boolean
  'x-custom-hidden'?: boolean
}

/**
 * Parses CRD OpenAPI v3 Schema to Field definitions for dynamic forms
 */
export function parseCRDSchemaToFields(schema: any): Field[] {
  if (!schema?.properties?.spec?.properties) {
    return []
  }

  const specProps = schema.properties.spec.properties as Record<string, OpenAPIProperty>
  const specRequired = schema.properties.spec.required as string[] || []

  const fields: Field[] = []

  for (const [path, prop] of Object.entries(specProps)) {
    // Skip hidden fields
    if (prop['x-custom-hidden']) continue

    const field: Field = {
      path,
      type: mapK8sTypeToFieldType(prop),
      label: prop['x-custom-label'] || prop.description || path,
      placeholder: prop['x-custom-placeholder'],
      required: specRequired.includes(path),
      readonly: prop['x-custom-readonly'] || false,
      pattern: prop.pattern,
      minimum: prop.minimum,
      maximum: prop.maximum,
      options: prop.enum?.map(v => ({ label: String(v), value: v })),
      group: prop['x-custom-group'] || i18n.global.t('nav.otherGroup'),
      default: prop.default,
    }

    // Handle nested object properties
    if (prop.type === 'object' && prop.properties) {
      field.fields = parseNestedProperties(prop.properties, [])
    }

    fields.push(field)
  }

  return fields
}

/**
 * Recursively parse nested object properties
 */
function parseNestedProperties(
  properties: Record<string, OpenAPIProperty>,
  required: string[] = []
): Field[] {
  const fields: Field[] = []

  for (const [path, prop] of Object.entries(properties)) {
    if (prop['x-custom-hidden']) continue

    fields.push({
      path,
      type: mapK8sTypeToFieldType(prop),
      label: prop['x-custom-label'] || prop.description || path,
      placeholder: prop['x-custom-placeholder'],
      required: required.includes(path),
      readonly: prop['x-custom-readonly'] || false,
      pattern: prop.pattern,
      minimum: prop.minimum,
      maximum: prop.maximum,
      options: prop.enum?.map(v => ({ label: String(v), value: v })),
      group: prop['x-custom-group'],
      default: prop.default,
    })
  }

  return fields
}

/**
 * Maps OpenAPI types to Field types
 */
function mapK8sTypeToFieldType(prop: OpenAPIProperty): Field['type'] {
  if (prop.enum) return 'enum'
  if (prop.type === 'boolean') return 'boolean'
  if (prop.type === 'integer' || prop.type === 'number') return 'number'
  if (prop.type === 'object' && prop.properties) return 'group'
  return 'string'
}

/**
 * Groups fields by their group property for collapsible sections
 */
export function groupFieldsByGroup(fields: Field[]): Map<string, Field[]> {
  const groups = new Map<string, Field[]>()

  for (const field of fields) {
    const groupName = field.group || i18n.global.t('nav.otherGroup')
    if (!groups.has(groupName)) {
      groups.set(groupName, [])
    }
    groups.get(groupName)!.push(field)
  }

  return groups
}

/**
 * Gets default values from the schema for initial form state
 */
export function getDefaultValues(fields: Field[]): Record<string, any> {
  const values: Record<string, any> = {}

  for (const field of fields) {
    if (field.default !== undefined) {
      values[field.path] = field.default
    } else if (field.type === 'boolean') {
      values[field.path] = false
    }
  }

  return values
}
