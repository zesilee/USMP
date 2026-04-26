// YANG Schema 类型定义 - 模型驱动 UI 的核心

/** YANG 节点类型 */
export type YangType =
  | 'boolean'
  | 'string'
  | 'int'
  | 'uint'
  | 'enum'
  | 'list'
  | 'container'
  | 'leafref'
  | 'empty'

/** 枚举选项 */
export interface YangEnumOption {
  name: string
  value: string | number
  description?: string
}

/** YANG 节点元数据 */
export interface YangNode {
  /** YANG 路径，如 '/vlans/vlan' */
  path: string
  /** 节点名称 */
  name: string
  /** 节点类型 */
  type: YangType
  /** 描述 */
  description?: string
  /** 是否可配置 */
  config?: boolean
  /** 是否必填 */
  mandatory?: boolean
  /** 默认值 */
  default?: any
  /** 枚举选项 (当 type = enum 时) */
  enumOptions?: YangEnumOption[]
  /** 数字范围 (当 type = int/uint 时) */
  range?: { min?: number; max?: number }
  /** 字符串长度限制 */
  length?: { min?: number; max?: number }
  /** 子节点 (当 type = container/list 时) */
  children?: YangNode[]
  /** list 的主键字段 */
  key?: string
}

/** 表单字段值类型 */
export type FieldValue =
  | string
  | number
  | boolean
  | null
  | undefined
  | Record<string, any>
  | any[]

/** 表单数据结构 */
export interface FormData {
  [field: string]: FieldValue
}

/** 验证结果 */
export interface ValidationResult {
  valid: boolean
  errors: ValidationError[]
}

/** 验证错误 */
export interface ValidationError {
  field: string
  message: string
}

/** 配置变更 */
export interface ConfigChange {
  path: string
  oldValue: FieldValue
  newValue: FieldValue
}

// ============== 预置的 YANG 模型 ==============

/** OpenConfig VLAN 模型 */
export const VLAN_SCHEMA: YangNode = {
  path: '/vlans',
  name: 'vlans',
  type: 'container',
  description: 'VLAN 配置管理',
  config: true,
  children: [
    {
      path: '/vlans/vlan',
      name: 'vlans',
      type: 'list',
      description: 'VLAN 列表',
      key: 'id',
      config: true,
      children: [
        {
          path: '/vlans/vlan/id',
          name: 'id',
          type: 'uint',
          description: 'VLAN ID (1-4094)',
          config: true,
          mandatory: true,
          range: { min: 1, max: 4094 }
        },
        {
          path: '/vlans/vlan/name',
          name: 'name',
          type: 'string',
          description: 'VLAN 名称',
          config: true,
          length: { min: 1, max: 32 }
        },
        {
          path: '/vlans/vlan/admin-status',
          name: 'admin-status',
          type: 'enum',
          description: '管理状态',
          config: true,
          enumOptions: [
            { name: 'UP', value: 'UP', description: '启用' },
            { name: 'DOWN', value: 'DOWN', description: '禁用' }
          ],
          default: 'UP'
        },
        {
          path: '/vlans/vlan/oper-status',
          name: 'oper-status',
          type: 'enum',
          description: '运行状态',
          config: false,
          enumOptions: [
            { name: 'ACTIVE', value: 'ACTIVE', description: '运行中' },
            { name: 'INACTIVE', value: 'INACTIVE', description: '未激活' },
            { name: 'SUSPENDED', value: 'SUSPENDED', description: '已暂停' }
          ]
        },
        {
          path: '/vlans/vlan/tagged-ports',
          name: 'tagged-ports',
          type: 'list',
          description: 'Tagged 端口列表',
          config: true,
          children: [
            {
              path: '/vlans/vlan/tagged-ports/port',
              name: 'port',
              type: 'string',
              description: '端口名称',
              config: true
            }
          ]
        },
        {
          path: '/vlans/vlan/untagged-ports',
          name: 'untagged-ports',
          type: 'list',
          description: 'Untagged 端口列表',
          config: true,
          children: [
            {
              path: '/vlans/vlan/untagged-ports/port',
              name: 'port',
              type: 'string',
              description: '端口名称',
              config: true
            }
          ]
        }
      ]
    }
  ]
}

/** OpenConfig Interfaces 模型 (简化版) */
export const INTERFACES_SCHEMA: YangNode = {
  path: '/interfaces',
  name: 'interfaces',
  type: 'container',
  description: '接口配置容器',
  config: true,
  children: [
    {
      path: '/interfaces/interface',
      name: 'interface',
      type: 'list',
      description: '接口列表',
      key: 'name',
      config: true,
      children: [
        {
          path: '/interfaces/interface/name',
          name: 'name',
          type: 'string',
          description: '接口名称',
          config: true,
          mandatory: true
        },
        {
          path: '/interfaces/interface/description',
          name: 'description',
          type: 'string',
          description: '接口描述',
          config: true
        },
        {
          path: '/interfaces/interface/enabled',
          name: 'enabled',
          type: 'boolean',
          description: '是否启用',
          config: true,
          default: true
        }
      ]
    }
  ]
}

/** Schema 注册表 */
export const SCHEMA_REGISTRY: Record<string, YangNode> = {
  '/vlans': VLAN_SCHEMA,
  '/interfaces': INTERFACES_SCHEMA
}

// ============== 工具函数 ==============

/** 根据路径获取 Schema 节点 */
export function getSchemaByPath(path: string): YangNode | undefined {
  return SCHEMA_REGISTRY[path]
}

/** 验证字段值 */
export function validateField(node: YangNode, value: FieldValue): ValidationResult {
  const errors: ValidationError[] = []

  // 必填检查
  if (node.mandatory && (value === undefined || value === null || value === '')) {
    errors.push({ field: node.name, message: `${node.description || node.name} 为必填项` })
  }

  // 类型检查
  if (value !== undefined && value !== null && value !== '') {
    switch (node.type) {
      case 'uint':
      case 'int':
        const num = Number(value)
        if (isNaN(num)) {
          errors.push({ field: node.name, message: '必须是数字' })
        } else if (node.range) {
          if (node.range.min !== undefined && num < node.range.min) {
            errors.push({ field: node.name, message: `最小值为 ${node.range.min}` })
          }
          if (node.range.max !== undefined && num > node.range.max) {
            errors.push({ field: node.name, message: `最大值为 ${node.range.max}` })
          }
        }
        break

      case 'string':
        if (node.length) {
          const str = String(value)
          if (node.length.min !== undefined && str.length < node.length.min) {
            errors.push({ field: node.name, message: `最少 ${node.length.min} 个字符` })
          }
          if (node.length.max !== undefined && str.length > node.length.max) {
            errors.push({ field: node.name, message: `最多 ${node.length.max} 个字符` })
          }
        }
        break

      case 'enum':
        if (node.enumOptions) {
          const validValues = node.enumOptions.map(o => o.value)
          if (!validValues.includes(value as string | number)) {
            errors.push({ field: node.name, message: '无效的枚举值' })
          }
        }
        break
    }
  }

  return { valid: errors.length === 0, errors }
}

/** 获取字段默认值 */
export function getDefaultValue(node: YangNode): FieldValue {
  if (node.default !== undefined) return node.default

  switch (node.type) {
    case 'boolean': return false
    case 'uint':
    case 'int': return undefined
    case 'string': return ''
    case 'enum': return node.enumOptions?.[0]?.value
    case 'list': return []
    case 'container': return {}
    default: return undefined
  }
}
