import { describe, it, expect } from 'vitest'
import {
  parseCRDSchemaToFields,
  groupFieldsByGroup,
  getDefaultValues,
} from '../../src/utils/crdSchemaParser'

describe('crdSchemaParser', () => {
  const mockSchema = {
    properties: {
      spec: {
        properties: {
          deviceID: {
            type: 'string',
            description: 'Target device ID',
            'x-custom-label': '设备 ID',
            'x-custom-placeholder': '例如: 192.168.1.1:830',
            'x-custom-group': '基本信息',
          },
          vlanID: {
            type: 'integer',
            minimum: 1,
            maximum: 4094,
            'x-custom-label': 'VLAN ID',
            'x-custom-group': '基本信息',
          },
          adminStatus: {
            type: 'string',
            enum: ['Up', 'Down'],
            'x-custom-label': '管理状态',
            'x-custom-group': '高级设置',
          },
          enabled: {
            type: 'boolean',
            default: true,
            'x-custom-label': '启用',
          },
        },
        required: ['deviceID', 'vlanID'],
      },
    },
  }

  describe('parseCRDSchemaToFields', () => {
    it('should parse schema to fields array', () => {
      const fields = parseCRDSchemaToFields(mockSchema)
      expect(Array.isArray(fields)).toBe(true)
      expect(fields.length).toBe(4)
    })

    it('should extract custom label from x-custom-label extension', () => {
      const fields = parseCRDSchemaToFields(mockSchema)
      const deviceIdField = fields.find(f => f.path === 'deviceID')
      expect(deviceIdField?.label).toBe('设备 ID')
    })

    it('should extract placeholder from x-custom-placeholder', () => {
      const fields = parseCRDSchemaToFields(mockSchema)
      const deviceIdField = fields.find(f => f.path === 'deviceID')
      expect(deviceIdField?.placeholder).toBe('例如: 192.168.1.1:830')
    })

    it('should map enum type correctly', () => {
      const fields = parseCRDSchemaToFields(mockSchema)
      const statusField = fields.find(f => f.path === 'adminStatus')
      expect(statusField?.type).toBe('enum')
      expect(statusField?.options).toEqual([
        { label: 'Up', value: 'Up' },
        { label: 'Down', value: 'Down' },
      ])
    })

    it('should map boolean type correctly', () => {
      const fields = parseCRDSchemaToFields(mockSchema)
      const enabledField = fields.find(f => f.path === 'enabled')
      expect(enabledField?.type).toBe('boolean')
    })

    it('should map integer type to number', () => {
      const fields = parseCRDSchemaToFields(mockSchema)
      const vlanField = fields.find(f => f.path === 'vlanID')
      expect(vlanField?.type).toBe('number')
      expect(vlanField?.minimum).toBe(1)
      expect(vlanField?.maximum).toBe(4094)
    })

    it('should mark required fields correctly', () => {
      const fields = parseCRDSchemaToFields(mockSchema)
      const deviceIdField = fields.find(f => f.path === 'deviceID')
      expect(deviceIdField?.required).toBe(true)
    })

    it('should extract group information', () => {
      const fields = parseCRDSchemaToFields(mockSchema)
      const deviceIdField = fields.find(f => f.path === 'deviceID')
      expect(deviceIdField?.group).toBe('基本信息')
    })

    it('should return empty array for invalid schema', () => {
      const fields = parseCRDSchemaToFields(null)
      expect(fields).toEqual([])

      const fields2 = parseCRDSchemaToFields({})
      expect(fields2).toEqual([])

      const fields3 = parseCRDSchemaToFields({ properties: {} })
      expect(fields3).toEqual([])
    })

    it('should skip hidden fields with x-custom-hidden', () => {
      const hiddenSchema = {
        properties: {
          spec: {
            properties: {
              visibleField: {
                type: 'string',
                'x-custom-label': '可见字段',
              },
              hiddenField: {
                type: 'string',
                'x-custom-hidden': true,
              },
            },
          },
        },
      }

      const fields = parseCRDSchemaToFields(hiddenSchema)
      expect(fields.length).toBe(1)
      expect(fields[0].path).toBe('visibleField')
    })
  })

  describe('groupFieldsByGroup', () => {
    it('should group fields by their group property', () => {
      const fields = [
        { path: 'deviceID', type: 'string' as const, label: '设备 ID', group: '基本信息' },
        { path: 'vlanID', type: 'number' as const, label: 'VLAN ID', group: '基本信息' },
        { path: 'adminStatus', type: 'enum' as const, label: '管理状态', group: '高级设置', options: [] },
      ]

      const groups = groupFieldsByGroup(fields)
      expect(groups.size).toBe(2)
      expect(groups.get('基本信息')?.length).toBe(2)
      expect(groups.get('高级设置')?.length).toBe(1)
    })

    it('should put ungrouped fields into "其他"', () => {
      const fields = [
        { path: 'field1', type: 'string' as const, label: '字段1' },
        { path: 'field2', type: 'number' as const, label: '字段2', group: '基本信息' },
      ]

      const groups = groupFieldsByGroup(fields)
      expect(groups.get('其他')?.length).toBe(1)
    })
  })

  describe('getDefaultValues', () => {
    it('should extract default values from fields', () => {
      const fields = [
        { path: 'enabled', type: 'boolean' as const, label: '启用', default: true },
        { path: 'name', type: 'string' as const, label: '名称', default: 'default' },
      ]

      const values = getDefaultValues(fields)
      expect(values.enabled).toBe(true)
      expect(values.name).toBe('default')
    })

    it('should return false for boolean fields without default', () => {
      const fields = [
        { path: 'enabled', type: 'boolean' as const, label: '启用' },
      ]

      const values = getDefaultValues(fields)
      expect(values.enabled).toBe(false)
    })

    it('should skip non-boolean fields without default', () => {
      const fields = [
        { path: 'name', type: 'string' as const, label: '名称' },
      ]

      const values = getDefaultValues(fields)
      expect(values.name).toBeUndefined()
    })
  })
})
