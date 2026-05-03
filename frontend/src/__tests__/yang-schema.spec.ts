import { describe, it, expect } from 'vitest'
import {
  VLAN_SCHEMA,
  INTERFACES_SCHEMA,
  SYSTEM_SCHEMA,
  SCHEMA_REGISTRY,
  validateField,
  getDefaultValue,
  kebabToCamel,
  camelToKebab,
  convertKeysToCamel,
  convertKeysToKebab,
  type YangNode,
} from '../types/yang-schema'

describe('YANG Schema 注册表', () => {
  it('应包含所有主要 Schema', () => {
    expect(SCHEMA_REGISTRY['/vlans']).toBeDefined()
    expect(SCHEMA_REGISTRY['/ifm:ifm/ifm:interfaces']).toBeDefined()
    expect(SCHEMA_REGISTRY['/system:system']).toBeDefined()
    expect(SCHEMA_REGISTRY['/interfaces']).toBeDefined()
  })

  it('所有 Schema 应该有正确的结构', () => {
    const schemas = [VLAN_SCHEMA, INTERFACES_SCHEMA, SYSTEM_SCHEMA]
    schemas.forEach(schema => {
      expect(schema.path).toBeDefined()
      expect(schema.name).toBeDefined()
      expect(schema.type).toBe('container')
      expect(schema.config).toBe(true)
    })
  })
})

describe('VLAN_SCHEMA 完整性验证', () => {
  it('应包含 vlan list 节点', () => {
    const vlanNode = VLAN_SCHEMA.children?.find(c => c.name === 'vlans')
    expect(vlanNode).toBeDefined()
    expect(vlanNode?.type).toBe('list')
    expect(vlanNode?.key).toBe('id')
    expect(vlanNode?.config).toBe(true)
  })

  it('应包含所有基础属性字段', () => {
    const vlanNode = VLAN_SCHEMA.children?.find(c => c.name === 'vlans')
    const fieldNames = vlanNode?.children?.map(c => c.name) || []

    const requiredFields = ['id', 'name', 'description', 'type', 'admin-status']
    requiredFields.forEach(field => {
      expect(fieldNames).toContain(field)
    })
  })

  it('应包含所有流量控制字段', () => {
    const vlanNode = VLAN_SCHEMA.children?.find(c => c.name === 'vlans')
    const fieldNames = vlanNode?.children?.map(c => c.name) || []

    expect(fieldNames).toContain('broadcast-discard')
    expect(fieldNames).toContain('unknown-multicast-discard')
  })

  it('应包含所有 MAC 学习字段', () => {
    const vlanNode = VLAN_SCHEMA.children?.find(c => c.name === 'vlans')
    const fieldNames = vlanNode?.children?.map(c => c.name) || []

    expect(fieldNames).toContain('mac-learning')
    expect(fieldNames).toContain('mac-aging-time')
  })

  it('应包含统计功能字段', () => {
    const vlanNode = VLAN_SCHEMA.children?.find(c => c.name === 'vlans')
    const fieldNames = vlanNode?.children?.map(c => c.name) || []

    expect(fieldNames).toContain('statistic-enable')
    expect(fieldNames).toContain('statistic-discard')
  })

  it('应包含关联 VLAN 字段 (leafref)', () => {
    const vlanNode = VLAN_SCHEMA.children?.find(c => c.name === 'vlans')
    const superVlanField = vlanNode?.children?.find(c => c.name === 'super-vlan')

    expect(superVlanField).toBeDefined()
    expect(superVlanField?.type).toBe('leafref')
  })

  it('应包含 unknown-unicast-discard 嵌套容器', () => {
    const vlanNode = VLAN_SCHEMA.children?.find(c => c.name === 'vlans')
    const unknownUnicastContainer = vlanNode?.children?.find(
      c => c.name === 'unknown-unicast-discard'
    )

    expect(unknownUnicastContainer).toBeDefined()
    expect(unknownUnicastContainer?.type).toBe('container')
    expect(unknownUnicastContainer?.config).toBe(true)

    const childNames = unknownUnicastContainer?.children?.map(c => c.name) || []
    expect(childNames).toContain('discard')
    expect(childNames).toContain('mac-learning-enable')
  })

  it('应包含 suppression 嵌套容器', () => {
    const vlanNode = VLAN_SCHEMA.children?.find(c => c.name === 'vlans')
    const suppressionContainer = vlanNode?.children?.find(
      c => c.name === 'suppression'
    )

    expect(suppressionContainer).toBeDefined()
    expect(suppressionContainer?.type).toBe('container')
    expect(suppressionContainer?.config).toBe(true)

    const childNames = suppressionContainer?.children?.map(c => c.name) || []
    expect(childNames).toContain('inbound')
    expect(childNames).toContain('outbound')
  })

  it('应包含端口列表字段', () => {
    const vlanNode = VLAN_SCHEMA.children?.find(c => c.name === 'vlans')
    const fieldNames = vlanNode?.children?.map(c => c.name) || []

    expect(fieldNames).toContain('tagged-ports')
    expect(fieldNames).toContain('untagged-ports')
  })

  it('应包含 instances 嵌套容器', () => {
    const instancesContainer = VLAN_SCHEMA.children?.find(
      c => c.name === 'instances'
    )

    expect(instancesContainer).toBeDefined()
    expect(instancesContainer?.type).toBe('container')
    expect(instancesContainer?.config).toBe(true)

    const instanceList = instancesContainer?.children?.find(
      c => c.name === 'instance'
    )
    expect(instanceList).toBeDefined()
    expect(instanceList?.type).toBe('list')
    expect(instanceList?.key).toBe('id')

    const instanceFields = instanceList?.children?.map(c => c.name) || []
    expect(instanceFields).toContain('id')
    expect(instanceFields).toContain('vlan-list')
  })

  it('枚举字段应包含完整的选项', () => {
    const vlanNode = VLAN_SCHEMA.children?.find(c => c.name === 'vlans')
    const typeField = vlanNode?.children?.find(c => c.name === 'type')

    expect(typeField).toBeDefined()
    expect(typeField?.enumOptions).toHaveLength(6) // common, super, sub, principal, separate, group
    expect(typeField?.default).toBe(1) // common
  })

  it('数值字段应包含正确的范围限制', () => {
    const vlanNode = VLAN_SCHEMA.children?.find(c => c.name === 'vlans')
    const idField = vlanNode?.children?.find(c => c.name === 'id')
    const agingTimeField = vlanNode?.children?.find(
      c => c.name === 'mac-aging-time'
    )

    expect(idField?.range).toEqual({ min: 1, max: 4094 })
    expect(agingTimeField?.range).toEqual({ min: 0, max: 1000000 })
  })

  it('字符串字段应包含正确的长度限制', () => {
    const vlanNode = VLAN_SCHEMA.children?.find(c => c.name === 'vlans')
    const nameField = vlanNode?.children?.find(c => c.name === 'name')
    const descField = vlanNode?.children?.find(
      c => c.name === 'description'
    )

    expect(nameField?.length).toEqual({ min: 1, max: 31 })
    expect(descField?.length).toEqual({ min: 1, max: 80 })
  })
})

describe('INTERFACES_SCHEMA 完整性验证', () => {
  it('应包含 interface list 节点', () => {
    const interfaceNode = INTERFACES_SCHEMA.children?.find(
      c => c.name === 'interface'
    )
    expect(interfaceNode).toBeDefined()
    expect(interfaceNode?.type).toBe('list')
    expect(interfaceNode?.key).toBe('name')
    expect(interfaceNode?.config).toBe(true)
  })

  it('应包含所有基础属性字段', () => {
    const interfaceNode = INTERFACES_SCHEMA.children?.find(
      c => c.name === 'interface'
    )
    const fieldNames = interfaceNode?.children?.map(c => c.name) || []

    const requiredFields = [
      'name', 'description', 'index', 'number', 'position',
      'parent-name', 'admin-status', 'type', 'class',
      'link-protocol', 'router-type', 'service-type'
    ]
    requiredFields.forEach(field => {
      expect(fieldNames).toContain(field)
    })
  })

  it('应包含所有网络参数字段', () => {
    const interfaceNode = INTERFACES_SCHEMA.children?.find(
      c => c.name === 'interface'
    )
    const fieldNames = interfaceNode?.children?.map(c => c.name) || []

    expect(fieldNames).toContain('mtu')
    expect(fieldNames).toContain('mac-address')
    expect(fieldNames).toContain('bandwidth')
    expect(fieldNames).toContain('bandwidth-kbps')
    expect(fieldNames).toContain('vrf-name')
    expect(fieldNames).toContain('vs-name')
  })

  it('应包含所有布尔功能开关字段', () => {
    const interfaceNode = INTERFACES_SCHEMA.children?.find(
      c => c.name === 'interface'
    )
    const fieldNames = interfaceNode?.children?.map(c => c.name) || []

    const booleanFields = [
      'clear-ip-df', 'is-l2-switch', 'l2-mode-enable',
      'link-up-down-trap-enable', 'statistic-enable',
      'spread-mtu-flag'
    ]
    booleanFields.forEach(field => {
      expect(fieldNames).toContain(field)
      const fieldNode = interfaceNode?.children?.find(c => c.name === field)
      expect(fieldNode?.type).toBe('boolean')
    })
  })

  it('应包含定时器字段', () => {
    const interfaceNode = INTERFACES_SCHEMA.children?.find(
      c => c.name === 'interface'
    )
    const fieldNames = interfaceNode?.children?.map(c => c.name) || []

    expect(fieldNames).toContain('down-delay-time')
    expect(fieldNames).toContain('protocol-up-delay-time')
  })

  it('应包含统计配置字段', () => {
    const interfaceNode = INTERFACES_SCHEMA.children?.find(
      c => c.name === 'interface'
    )
    const fieldNames = interfaceNode?.children?.map(c => c.name) || []

    expect(fieldNames).toContain('statistic-interval')
    expect(fieldNames).toContain('statistic-mode')
  })

  it('应包含 control-flap 嵌套容器', () => {
    const interfaceNode = INTERFACES_SCHEMA.children?.find(
      c => c.name === 'interface'
    )
    const controlFlapContainer = interfaceNode?.children?.find(
      c => c.name === 'control-flap'
    )

    expect(controlFlapContainer).toBeDefined()
    expect(controlFlapContainer?.type).toBe('container')
    expect(controlFlapContainer?.config).toBe(true)

    const childNames = controlFlapContainer?.children?.map(c => c.name) || []
    expect(childNames).toContain('ceiling')
    expect(childNames).toContain('control-flap-count')
    expect(childNames).toContain('decay-ng')
    expect(childNames).toContain('decay-ok')
    expect(childNames).toContain('reuse')
    expect(childNames).toContain('suppress')
  })

  it('应包含 damp 嵌套容器', () => {
    const interfaceNode = INTERFACES_SCHEMA.children?.find(
      c => c.name === 'interface'
    )
    const dampContainer = interfaceNode?.children?.find(
      c => c.name === 'damp'
    )

    expect(dampContainer).toBeDefined()
    expect(dampContainer?.type).toBe('container')
    expect(dampContainer?.config).toBe(true)

    const childNames = dampContainer?.children?.map(c => c.name) || []
    expect(childNames).toContain('auto')
    expect(childNames).toContain('manual')

    // 验证 auto 子容器
    const autoContainer = dampContainer?.children?.find(c => c.name === 'auto')
    expect(autoContainer?.children?.map(c => c.name)).toContain('level')

    // 验证 manual 子容器
    const manualContainer = dampContainer?.children?.find(c => c.name === 'manual')
    const manualFields = manualContainer?.children?.map(c => c.name) || []
    expect(manualFields).toContain('half-life-period')
    expect(manualFields).toContain('max-suppress-time')
    expect(manualFields).toContain('reuse')
    expect(manualFields).toContain('suppress')
  })

  it('所有 config=true 的字段应该正确标记', () => {
    const interfaceNode = INTERFACES_SCHEMA.children?.find(
      c => c.name === 'interface'
    )
    const configFields = interfaceNode?.children?.filter(
      c => c.config !== false
    ) || []

    // 至少应该有这么多字段
    expect(configFields.length).toBeGreaterThan(30)
  })
})

describe('SYSTEM_SCHEMA 完整性验证', () => {
  it('应包含 system-info 容器节点', () => {
    const systemInfoContainer = SYSTEM_SCHEMA.children?.find(
      c => c.name === 'system-info'
    )
    expect(systemInfoContainer).toBeDefined()
    expect(systemInfoContainer?.type).toBe('container')
    expect(systemInfoContainer?.config).toBe(true)
  })

  it('应包含所有系统配置字段', () => {
    const systemInfoContainer = SYSTEM_SCHEMA.children?.find(
      c => c.name === 'system-info'
    )
    const fieldNames = systemInfoContainer?.children?.map(c => c.name) || []

    expect(fieldNames).toContain('sys-name')
    expect(fieldNames).toContain('sys-contact')
    expect(fieldNames).toContain('sys-location')
  })

  it('只读字段应正确标记为 config=false', () => {
    const systemInfoContainer = SYSTEM_SCHEMA.children?.find(
      c => c.name === 'system-info'
    )
    const readOnlyFields = systemInfoContainer?.children?.filter(
      c => c.config === false
    ) || []

    const readOnlyNames = readOnlyFields.map(f => f.name)
    expect(readOnlyNames).toContain('sys-desc')
    expect(readOnlyNames).toContain('product-name')
    expect(readOnlyNames).toContain('product-version')
    expect(readOnlyNames).toContain('esn')
    expect(readOnlyNames).toContain('sys-uptime')
  })
})

describe('字段验证函数', () => {
  it('必填字段验证应正确工作', () => {
    const node: YangNode = {
      path: '/test',
      name: 'test',
      type: 'string',
      mandatory: true,
    }

    expect(validateField(node, '').valid).toBe(false)
    expect(validateField(node, null).valid).toBe(false)
    expect(validateField(node, undefined).valid).toBe(false)
    expect(validateField(node, 'value').valid).toBe(true)
  })

  it('数值范围验证应正确工作', () => {
    const node: YangNode = {
      path: '/test',
      name: 'test',
      type: 'uint',
      range: { min: 1, max: 100 },
    }

    expect(validateField(node, 0).valid).toBe(false)
    expect(validateField(node, 101).valid).toBe(false)
    expect(validateField(node, 50).valid).toBe(true)
  })

  it('字符串长度验证应正确工作', () => {
    const node: YangNode = {
      path: '/test',
      name: 'test',
      type: 'string',
      length: { min: 1, max: 10 },
    }

    // 空字符串在非必填时被跳过验证
    expect(validateField(node, '').valid).toBe(true)
    // 但超过最大长度时应失败
    expect(validateField(node, 'a'.repeat(11)).valid).toBe(false)
    // 正常长度应通过
    expect(validateField(node, 'valid').valid).toBe(true)
    // 必填字段为空时应该失败
    const mandatoryNode: YangNode = {
      ...node,
      mandatory: true,
    }
    expect(validateField(mandatoryNode, '').valid).toBe(false)
  })

  it('枚举值验证应正确工作', () => {
    const node: YangNode = {
      path: '/test',
      name: 'test',
      type: 'enum',
      enumOptions: [
        { name: 'A', value: 1 },
        { name: 'B', value: 2 },
      ],
    }

    expect(validateField(node, 0).valid).toBe(false)
    expect(validateField(node, 1).valid).toBe(true)
    expect(validateField(node, 2).valid).toBe(true)
  })
})

describe('默认值函数', () => {
  it('boolean 类型应返回 false 作为默认值', () => {
    const node: YangNode = {
      path: '/test',
      name: 'test',
      type: 'boolean',
    }
    expect(getDefaultValue(node)).toBe(false)
  })

  it('string 类型应返回空字符串作为默认值', () => {
    const node: YangNode = {
      path: '/test',
      name: 'test',
      type: 'string',
    }
    expect(getDefaultValue(node)).toBe('')
  })

  it('自定义默认值应优先返回', () => {
    const node: YangNode = {
      path: '/test',
      name: 'test',
      type: 'uint',
      default: 100,
    }
    expect(getDefaultValue(node)).toBe(100)
  })
})

describe('键名转换工具函数', () => {
  it('kebabToCamel 应正确转换命名风格', () => {
    expect(kebabToCamel('admin-status')).toBe('adminStatus')
    expect(kebabToCamel('vlan-id')).toBe('vlanId')
    expect(kebabToCamel('name')).toBe('name')
  })

  it('camelToKebab 应正确转换命名风格', () => {
    expect(camelToKebab('adminStatus')).toBe('admin-status')
    expect(camelToKebab('vlanId')).toBe('vlan-id')
    expect(camelToKebab('name')).toBe('name')
  })

  it('convertKeysToCamel 应递归转换对象键名', () => {
    const input = {
      'admin-status': 'UP',
      'vlan-id': 100,
      nested: {
        'tagged-ports': ['GigabitEthernet0/0/1'],
      },
    }
    const result = convertKeysToCamel(input)
    expect(result).toHaveProperty('adminStatus')
    expect(result).toHaveProperty('vlanId')
    expect(result.nested).toHaveProperty('taggedPorts')
  })

  it('convertKeysToKebab 应递归转换对象键名', () => {
    const input = {
      adminStatus: 'UP',
      vlanId: 100,
      nested: {
        taggedPorts: ['GigabitEthernet0/0/1'],
      },
    }
    const result = convertKeysToKebab(input)
    expect(result).toHaveProperty('admin-status')
    expect(result).toHaveProperty('vlan-id')
    expect(result.nested).toHaveProperty('tagged-ports')
  })
})
