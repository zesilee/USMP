export interface Field {
  path: string
  type: 'string' | 'number' | 'boolean' | 'enum' | 'group'
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
      group: prop['x-custom-group'] || '其他',
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
    const groupName = field.group || '其他'
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
