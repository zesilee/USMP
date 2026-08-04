import { describe, it, expect } from 'vitest'
import { validateDeviceForm, toAddDevicePayload, type DeviceFormInput } from '../../src/utils/deviceForm'

const base: DeviceFormInput = { ip: '7.225.21.14', port: '830', username: 'admin', password: 'secret', vendor: '', role: '' }
const form = (over: Partial<DeviceFormInput> = {}): DeviceFormInput => ({ ...base, ...over })

describe('validateDeviceForm', () => {
  it('合法表单无错误', () => {
    expect(validateDeviceForm(form())).toEqual({})
  })

  it.each([
    ['IP 必填', { ip: '' }, 'ip', 'devices.ruleIpRequired'],
    ['IP 纯空格视为空', { ip: '   ' }, 'ip', 'devices.ruleIpRequired'],
    ['IP 非法格式', { ip: 'not-an-ip' }, 'ip', 'devices.ruleIpInvalid'],
    ['IP 段越界', { ip: '999.1.1.1' }, 'ip', 'devices.ruleIpInvalid'],
    ['IP 段数不足', { ip: '1.2.3' }, 'ip', 'devices.ruleIpInvalid'],
    ['端口非数字', { port: 'abc' }, 'port', 'devices.rulePortInvalid'],
    ['端口为 0', { port: '0' }, 'port', 'devices.rulePortInvalid'],
    ['端口越界', { port: '70000' }, 'port', 'devices.rulePortInvalid'],
    ['端口带小数', { port: '830.5' }, 'port', 'devices.rulePortInvalid'],
    ['用户名必填', { username: '' }, 'username', 'devices.ruleUserRequired'],
    ['密码必填', { password: '' }, 'password', 'devices.rulePassRequired'],
    ['角色非法字符', { role: 'core gw' }, 'role', 'devices.ruleRoleInvalid'],
    ['角色超长', { role: 'x'.repeat(33) }, 'role', 'devices.ruleRoleInvalid'],
  ])('%s', (_name, over, field, key) => {
    expect(validateDeviceForm(form(over as Partial<DeviceFormInput>))[field as 'ip']).toBe(key)
  })

  it.each([
    ['端口留空（后端缺省 830）', { port: '' }],
    ['端口边界 1', { port: '1' }],
    ['端口边界 65535', { port: '65535' }],
    ['厂商留空（后端缺省 huawei）', { vendor: '' }],
    ['角色留空', { role: '' }],
    ['角色合法', { role: 'DC-GW_1' }],
  ])('%s 视为通过', (_name, over) => {
    expect(validateDeviceForm(form(over as Partial<DeviceFormInput>))).toEqual({})
  })

  it('多字段同时非法各自报错', () => {
    const errs = validateDeviceForm(form({ ip: '', username: '', password: '' }))
    expect(Object.keys(errs).sort()).toEqual(['ip', 'password', 'username'])
  })
})

describe('toAddDevicePayload', () => {
  it('去空白并省略空可选字段', () => {
    expect(toAddDevicePayload(form({ ip: ' 7.225.21.14 ', username: ' admin ', port: '', vendor: '', role: '' }))).toEqual({
      ip: '7.225.21.14',
      port: undefined,
      username: 'admin',
      password: 'secret',
      vendor: undefined,
      role: undefined,
    })
  })

  it('端口转数字、可选字段透传', () => {
    expect(toAddDevicePayload(form({ port: '830', vendor: 'huawei', role: 'DCGW' }))).toMatchObject({
      port: 830,
      vendor: 'huawei',
      role: 'DCGW',
    })
  })

  it('密码不做 trim（尾随空格可能是真密码的一部分）', () => {
    expect(toAddDevicePayload(form({ password: ' pa ss ' })).password).toBe(' pa ss ')
  })
})
