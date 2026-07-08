import type { Field } from '../../src/utils/crdSchemaParser'

// IFM 形嵌套 schema fixture（呈现元数据取自真实 huawei-ifm 契约）：
// 根下 global（含 presence 容器 + must）、damp、interfaces（group→interface list）、
// auto-recovery-times（group→list）。供通用控制台 F2 用例共享。
const ifLeaf = (name: string, extra: Partial<Field> = {}): Field => ({
  path: `/ifm/interfaces/interface/${name}`,
  type: 'string',
  label: name,
  ...extra,
})

export const ifmNestedSchema = {
  module: 'ifm',
  title: 'ifm',
  vendor: 'huawei',
  fields: [
    {
      path: '/ifm/global',
      type: 'group',
      label: 'global',
      fields: [
        {
          path: '/ifm/global/statistic-interval',
          type: 'number',
          label: 'statistic-interval',
          minimum: 10,
          maximum: 600,
          default: 300,
          must: [{ expr: '(../statistic-interval) mod 10 = 0', message: '统计间隔须为 10 的倍数' }],
        },
        {
          path: '/ifm/global/ipv4-ignore-primary-sub',
          type: 'boolean',
          label: 'ipv4-ignore-primary-sub',
        },
        {
          path: '/ifm/global/ipv4-conflict-enable',
          type: 'group',
          label: 'ipv4-conflict-enable',
          presence: true,
          must: [{ expr: "../ipv4-ignore-primary-sub='false'", message: '需先关闭 ignore-primary-sub' }],
          fields: [],
        },
      ],
    },
    {
      path: '/ifm/damp',
      type: 'group',
      label: 'damp',
      fields: [{ path: '/ifm/damp/level', type: 'enum', label: 'level' }],
    },
    {
      path: '/ifm/interfaces',
      type: 'group',
      label: 'interfaces',
      fields: [
        {
          path: '/ifm/interfaces/interface',
          type: 'list',
          label: 'interface',
          fields: [
            ifLeaf('name', { isKey: true, required: true }),
            ifLeaf('class', {
              type: 'enum',
              supportFilter: true,
              operationExclude: ['update', 'delete'],
              options: [
                { label: 'main-interface', value: 'main-interface' },
                { label: 'sub-interface', value: 'sub-interface' },
              ],
            }),
            ifLeaf('type', {
              type: 'enum',
              supportFilter: true,
              operationExclude: ['update', 'delete'],
              options: [
                { label: '200GE', value: '200GE' },
                { label: 'Vlanif', value: 'Vlanif' },
              ],
            }),
            ifLeaf('parent-name', { when: "../class='sub-interface'", operationExclude: ['update', 'delete'] }),
            ifLeaf('number', { operationExclude: ['update', 'delete'] }),
            ifLeaf('admin-status', {
              type: 'enum',
              options: [
                { label: 'up', value: 'up' },
                { label: 'down', value: 'down' },
              ],
            }),
            ifLeaf('link-protocol', { type: 'enum', options: [{ label: 'ethernet', value: 'ethernet' }] }),
            ifLeaf('router-type', { type: 'enum', operationExclude: ['update', 'delete'] }),
            ifLeaf('description'),
          ],
        },
      ],
    },
    {
      path: '/ifm/auto-recovery-times',
      type: 'group',
      label: 'auto-recovery-times',
      fields: [
        {
          path: '/ifm/auto-recovery-times/auto-recovery-time',
          type: 'list',
          label: 'auto-recovery-time',
          fields: [
            {
              path: '/ifm/auto-recovery-times/auto-recovery-time/error-down-type',
              type: 'enum',
              label: 'error-down-type',
              isKey: true,
            },
            {
              path: '/ifm/auto-recovery-times/auto-recovery-time/time-value',
              type: 'number',
              label: 'time-value',
            },
          ],
        },
      ],
    },
  ] as Field[],
}

// 模拟数据（用户契约）：3 条 main-interface/200GE/up + 2 条 sub-interface/Vlanif/down。
export const seedRows = [
  { name: '200GE0/1/0', class: 'main-interface', type: '200GE', number: '0/1/0', 'admin-status': 'up', 'link-protocol': 'ethernet' },
  { name: '200GE0/1/1', class: 'main-interface', type: '200GE', number: '0/1/1', 'admin-status': 'up', 'link-protocol': 'ethernet' },
  { name: '200GE0/1/2', class: 'main-interface', type: '200GE', number: '0/1/2', 'admin-status': 'up', 'link-protocol': 'ethernet' },
  { name: '200GE0/1/0.1', class: 'sub-interface', type: 'Vlanif', 'parent-name': '200GE0/1/0', number: '0/1/0.1', 'admin-status': 'down' },
  { name: '200GE0/1/1.1', class: 'sub-interface', type: 'Vlanif', 'parent-name': '200GE0/1/1', number: '0/1/1.1', 'admin-status': 'down' },
]
