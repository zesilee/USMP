import { describe, it, expect } from 'vitest'
import type { Field } from '../../src/utils/crdSchemaParser'
import {
  deriveTabs,
  deriveRpcTabs,
  deriveDetailTabs,
  deriveColumns,
  deriveAllColumns,
  deriveKeyField,
  filterableFields,
  filterRows,
  cellVisible,
  configPathFor,
  statusTone,
} from '../../src/utils/moduleConsole'

// IFM 形嵌套 schema（缩减版）：根下 global(group) + interfaces(group→interface list)。
// 列表叶携带真实 IFM 呈现元数据：isKey/supportFilter/operationExclude/when。
const leaf = (name: string, extra: Partial<Field> = {}): Field => ({
  path: `/ifm/interfaces/interface/${name}`,
  type: 'string',
  label: name,
  ...extra,
})

const interfaceList: Field = {
  path: '/ifm/interfaces/interface',
  type: 'list',
  label: 'interface',
  fields: [
    leaf('name', { isKey: true }),
    leaf('description'),
    leaf('class', {
      type: 'enum',
      supportFilter: true,
      operationExclude: ['update', 'delete'],
      options: [
        { label: 'main-interface', value: 'main-interface' },
        { label: 'sub-interface', value: 'sub-interface' },
      ],
    }),
    leaf('type', { type: 'enum', supportFilter: true, operationExclude: ['update', 'delete'] }),
    leaf('parent-name', { when: "../class='sub-interface'", operationExclude: ['update', 'delete'] }),
    leaf('number', { operationExclude: ['update', 'delete'] }),
    leaf('admin-status', { type: 'enum' }),
    leaf('link-protocol', { type: 'enum' }),
    leaf('router-type', { type: 'enum', operationExclude: ['update', 'delete'] }),
    leaf('mtu', { type: 'number' }),
    leaf('vrf-name'),
    { path: '/ifm/interfaces/interface/damp', type: 'group', label: 'damp', fields: [] },
  ],
}

const rootFields: Field[] = [
  {
    path: '/ifm/global',
    type: 'group',
    label: 'global',
    fields: [
      { path: '/ifm/global/statistic-interval', type: 'number', label: 'statistic-interval' },
    ],
  },
  { path: '/ifm/interfaces', type: 'group', label: 'interfaces', fields: [interfaceList] },
  { path: '/ifm/damp', type: 'group', label: 'damp', fields: [leaf('x')] },
]

const rows = [
  { name: '200GE0/1/0', class: 'main-interface', type: '200GE', 'admin-status': 'up' },
  { name: '200GE0/1/1', class: 'main-interface', type: '200GE', 'admin-status': 'up' },
  { name: '200GE0/1/2', class: 'main-interface', type: '200GE', 'admin-status': 'up' },
  { name: '200GE0/1/0.1', class: 'sub-interface', type: 'Vlanif', 'parent-name': '200GE0/1/0', 'admin-status': 'down' },
  { name: '200GE0/1/1.1', class: 'sub-interface', type: 'Vlanif', 'parent-name': '200GE0/1/1', 'admin-status': 'down' },
]

describe('deriveTabs · 模块根子节点→Tab（零模块硬编码）', () => {
  it('group 包裹单 list → 列表 Tab；普通 group → 表单 Tab', () => {
    const tabs = deriveTabs(rootFields)
    const byName = Object.fromEntries(tabs.map((t) => [t.name, t]))
    expect(byName['interfaces'].kind).toBe('list')
    expect(byName['interfaces'].listField?.path).toBe('/ifm/interfaces/interface')
    expect(byName['global'].kind).toBe('form')
    expect(byName['damp'].kind).toBe('form')
  })

  it('散落根叶子聚合为「基本属性」表单 Tab 且排最前', () => {
    const tabs = deriveTabs([
      { path: '/m/enabled', type: 'boolean', label: 'enabled' },
      ...rootFields,
    ])
    expect(tabs[0].name).toBe('__basic__')
    expect(tabs[0].kind).toBe('form')
    expect(tabs[0].field.fields?.map((f) => f.label)).toEqual(['enabled'])
  })

  it('裸 list 根子节点直接成列表 Tab', () => {
    const tabs = deriveTabs([interfaceList])
    expect(tabs[0].kind).toBe('list')
    expect(tabs[0].listField?.path).toBe(interfaceList.path)
  })

  it('空/无子节点输入返回空数组（降级不崩）', () => {
    expect(deriveTabs([])).toEqual([])
    expect(deriveTabs(undefined as any)).toEqual([])
  })
})

describe('deriveKeyField / deriveColumns · 模型驱动列派生', () => {
  it('keyField 取 isKey 叶；缺失时回退首个标量叶', () => {
    expect(deriveKeyField(interfaceList)).toBe('name')
    const noKey: Field = { ...interfaceList, fields: [leaf('a'), leaf('b')] }
    expect(deriveKeyField(noKey)).toBe('a')
  })

  it('分层取列：key→identity(operationExclude∋update)→when 条件列→enum→其余，封顶', () => {
    const cols = deriveColumns(interfaceList, 9).map((c) => c.label)
    // key 首列
    expect(cols[0]).toBe('name')
    // identity 层（schema 序）：class/type/parent-name/number/router-type
    expect(cols.slice(1, 6)).toEqual(['class', 'type', 'parent-name', 'number', 'router-type'])
    // enum 层：admin-status/link-protocol
    expect(cols.slice(6, 8)).toEqual(['admin-status', 'link-protocol'])
    // 封顶 9：其余标量只进 1 个；group 子节点不入列
    expect(cols).toHaveLength(9)
    expect(cols).not.toContain('damp')
  })

  it('层内去重且 cap 生效', () => {
    const cols = deriveColumns(interfaceList, 3)
    expect(cols).toHaveLength(3)
    expect(new Set(cols.map((c) => c.path)).size).toBe(3)
  })
})

describe('filterableFields / filterRows · support-filter 驱动的高级搜索', () => {
  it('搜索字段集仅取 supportFilter=true 的叶', () => {
    expect(filterableFields(interfaceList).map((f) => f.label)).toEqual(['class', 'type'])
  })

  it('enum 全等过滤 + 空条件跳过', () => {
    const fields = filterableFields(interfaceList)
    expect(filterRows(rows, { class: 'sub-interface' }, fields)).toHaveLength(2)
    expect(filterRows(rows, { class: '' }, fields)).toHaveLength(5)
    expect(filterRows(rows, {}, fields)).toHaveLength(5)
  })

  it('字符串子串过滤（大小写不敏感）', () => {
    const fields = [leaf('name')]
    expect(filterRows(rows, { name: '0/1/0' }, fields)).toHaveLength(2)
    expect(filterRows(rows, { name: 'vlanif' }, fields)).toHaveLength(0)
  })

  it('组合条件为 AND', () => {
    const fields = filterableFields(interfaceList)
    expect(filterRows(rows, { class: 'sub-interface', type: '200GE' }, fields)).toHaveLength(0)
  })
})

describe('cellVisible · 行级 when 单元格', () => {
  const parentCol = interfaceList.fields!.find((f) => f.label === 'parent-name')!

  it('when 以行数据求值：main 行不可见、sub 行可见', () => {
    expect(cellVisible(parentCol, rows[0])).toBe(false)
    expect(cellVisible(parentCol, rows[3])).toBe(true)
  })

  it('无 when 恒可见；求值失败降级可见（R08）', () => {
    expect(cellVisible(leaf('name'), rows[0])).toBe(true)
    expect(cellVisible(leaf('bad', { when: '((' }), rows[0])).toBe(true)
  })
})

describe('configPathFor · 运行时规范路径派生', () => {
  it('逐段加模块根前缀（对齐控制器注册路径）', () => {
    expect(configPathFor('ifm', '/ifm/interfaces')).toBe('ifm:ifm/ifm:interfaces')
    expect(configPathFor('vlan', '/vlan/vlans')).toBe('vlan:vlan/vlan:vlans')
    expect(configPathFor('system', '/system')).toBe('system:system')
  })
})

describe('statusTone · 值驱动状态色（非字段名驱动）', () => {
  it('up→ok，down→bad，其余无色', () => {
    expect(statusTone('up')).toBe('ok')
    expect(statusTone('down')).toBe('bad')
    expect(statusTone('main-interface')).toBe('')
    expect(statusTone(1500)).toBe('')
  })
})

describe('deriveTabs · readonly 子树降级只读 Tab（FE-14）', () => {
  const roList: Field = {
    path: '/ifm/remote-interfaces/remote-interface',
    type: 'list',
    label: 'remote-interface',
    readonly: true,
    fields: [
      { path: '/ifm/remote-interfaces/remote-interface/index', type: 'string', label: 'index', readonly: true, isKey: true },
    ],
  }
  const roGroup: Field = {
    path: '/ifm/remote-interfaces',
    type: 'group',
    label: 'remote-interfaces',
    readonly: true,
    fields: [roList],
  }

  it('整棵 readonly group 包裹单 list → 只读列表 Tab（不因 readonly 过滤而误判成表单）', () => {
    const tabs = deriveTabs([roGroup])
    expect(tabs).toHaveLength(1)
    expect(tabs[0].kind).toBe('list')
    expect(tabs[0].readonly).toBe(true)
    expect(tabs[0].listField?.path).toBe('/ifm/remote-interfaces/remote-interface')
  })

  it('readonly 裸 list → 只读列表 Tab；可编辑节点 readonly 不置位', () => {
    const tabs = deriveTabs([roList, ...rootFields])
    const byName = Object.fromEntries(tabs.map((t) => [t.name, t]))
    expect(byName['remote-interface'].readonly).toBe(true)
    expect(byName['interfaces'].readonly).toBeFalsy()
    expect(byName['global'].readonly).toBeFalsy()
  })

  it('readonly 普通 group → 只读表单 Tab', () => {
    const roForm: Field = {
      path: '/ifm/ipv4-interface-count',
      type: 'group',
      label: 'ipv4-interface-count',
      readonly: true,
      fields: [
        { path: '/ifm/ipv4-interface-count/protocol-up-count', type: 'number', label: 'protocol-up-count', readonly: true },
        { path: '/ifm/ipv4-interface-count/protocol-down-count', type: 'number', label: 'protocol-down-count', readonly: true },
      ],
    }
    const tabs = deriveTabs([roForm])
    expect(tabs[0].kind).toBe('form')
    expect(tabs[0].readonly).toBe(true)
  })

  it('全 readonly 散落根叶 → 基本属性 Tab 只读；混有可编辑叶则不只读', () => {
    const roLeaf: Field = { path: '/ifm/uptime', type: 'string', label: 'uptime', readonly: true }
    const rwLeaf: Field = { path: '/ifm/name', type: 'string', label: 'name' }
    expect(deriveTabs([roLeaf])[0].readonly).toBe(true)
    expect(deriveTabs([roLeaf, rwLeaf])[0].readonly).toBeFalsy()
  })
})

describe('deriveDetailTabs · 详情二级 Tab 派生（FE-21）', () => {
  const dleaf = (name: string, extra: Partial<Field> = {}): Field => ({
    path: `/ifm/interfaces/interface/${name}`,
    type: 'string',
    label: name,
    ...extra,
  })
  const dampGroup: Field = {
    path: '/ifm/interfaces/interface/damp',
    type: 'group',
    label: 'damp',
    fields: [dleaf('suppress', { type: 'number' })],
  }
  // group 包裹单 list（常见形态）→ 子表格 Tab
  const wrappedList: Field = {
    path: '/ifm/interfaces/interface/sub-ifs',
    type: 'group',
    label: 'sub-ifs',
    fields: [
      {
        path: '/ifm/interfaces/interface/sub-ifs/sub-if',
        type: 'list',
        label: 'sub-if',
        fields: [dleaf('id', { isKey: true })],
      },
    ],
  }
  const bareList: Field = {
    path: '/ifm/interfaces/interface/addrs',
    type: 'list',
    label: 'addrs',
    fields: [dleaf('ip', { isKey: true })],
  }
  const detailList: Field = {
    path: '/ifm/interfaces/interface',
    type: 'list',
    label: 'interface',
    fields: [
      dleaf('name', { isKey: true }),
      dleaf('mtu', { type: 'number' }),
      dleaf('groups', { type: 'leaf-list' }),
      dampGroup,
      wrappedList,
      bareList,
    ],
  }

  it('标量/leaf-list 叶 → 首个主表单 Tab（label=list 标签）', () => {
    const tabs = deriveDetailTabs(detailList)
    expect(tabs[0].name).toBe('__main__')
    expect(tabs[0].kind).toBe('form')
    expect(tabs[0].label).toBe('interface')
    expect(tabs[0].field.fields?.map((f) => f.label)).toEqual(['name', 'mtu', 'groups'])
  })

  it('嵌套 group → 子表单 Tab；裸嵌套 list 与 group 包裹单 list → 子表格 Tab（schema 序）', () => {
    const tabs = deriveDetailTabs(detailList)
    expect(tabs.map((t) => t.name)).toEqual(['__main__', 'damp', 'sub-ifs', 'addrs'])
    const byName = Object.fromEntries(tabs.map((t) => [t.name, t]))
    expect(byName['damp'].kind).toBe('form')
    expect(byName['sub-ifs'].kind).toBe('list')
    expect(byName['sub-ifs'].listField?.path).toBe('/ifm/interfaces/interface/sub-ifs/sub-if')
    expect(byName['addrs'].kind).toBe('list')
    expect(byName['addrs'].listField?.path).toBe(bareList.path)
  })

  it('无嵌套子节点 → 仅单主 Tab（FE-21 退化边界）', () => {
    const flat: Field = { ...detailList, fields: [dleaf('name', { isKey: true }), dleaf('mtu')] }
    const tabs = deriveDetailTabs(flat)
    expect(tabs).toHaveLength(1)
    expect(tabs[0].name).toBe('__main__')
  })

  it('readonly 嵌套子树 → 只读子 Tab；主 Tab 跟随 list 自身 readonly', () => {
    const roGroup: Field = { ...dampGroup, readonly: true }
    const tabs = deriveDetailTabs({ ...detailList, fields: [dleaf('name'), roGroup] })
    const byName = Object.fromEntries(tabs.map((t) => [t.name, t]))
    expect(byName['damp'].readonly).toBe(true)
    expect(byName['__main__'].readonly).toBeFalsy()
    const roTabs = deriveDetailTabs({ ...detailList, readonly: true, fields: [dleaf('name', { readonly: true })] })
    expect(roTabs[0].readonly).toBe(true)
  })

  it('空/undefined fields → 单主 Tab 空表单（降级不崩，R08）', () => {
    expect(deriveDetailTabs({ ...detailList, fields: [] })).toHaveLength(1)
    expect(deriveDetailTabs({ ...detailList, fields: undefined })).toHaveLength(1)
  })
})

describe('deriveAllColumns · 可用列全集（FE-11 列设置）', () => {
  it('全集含全部标量叶（group/list 子节点不入列），层序与默认集一致', () => {
    const all = deriveAllColumns(interfaceList).map((c) => c.label)
    expect(all).toHaveLength(11)
    expect(all).toContain('mtu')
    expect(all).toContain('vrf-name')
    expect(all).not.toContain('damp')
  })

  it('默认集 = 全集前缀（默认 9 列语义不变）', () => {
    const all = deriveAllColumns(interfaceList)
    const defaults = deriveColumns(interfaceList, 9)
    expect(defaults.map((c) => c.path)).toEqual(all.slice(0, 9).map((c) => c.path))
  })
})

describe('deriveRpcTabs（FE-19：rpc 与容器平级派生）', () => {
  const rpcs = [
    {
      name: 'reset-if-counters-by-name',
      label: '按接口名清除统计',
      highRisk: false,
      input: [{ path: 'if-name', type: 'string' as const, label: 'if-name', required: true, leafRef: '/ifm/interfaces/interface/name' }],
    },
    { name: 'restart-if', label: '重启接口', highRisk: true, input: [] as Field[] },
  ]

  it('每个 rpc 派生一个 kind=rpc 的 Tab，携带 rpc 定义', () => {
    const tabs = deriveRpcTabs(rpcs)
    expect(tabs).toHaveLength(2)
    expect(tabs[0].kind).toBe('rpc')
    expect(tabs[0].label).toBe('按接口名清除统计')
    expect(tabs[0].rpc?.name).toBe('reset-if-counters-by-name')
    expect(tabs[0].rpc?.input[0].required).toBe(true)
    // name 前缀避免与容器 Tab 撞名
    expect(tabs[0].name.startsWith('__rpc__')).toBe(true)
  })

  it('高危标记透传', () => {
    const tabs = deriveRpcTabs(rpcs)
    expect(tabs[1].rpc?.highRisk).toBe(true)
    expect(tabs[0].rpc?.highRisk).toBe(false)
  })

  it('空/undefined 输入 → 空数组', () => {
    expect(deriveRpcTabs([])).toEqual([])
    expect(deriveRpcTabs(undefined)).toEqual([])
  })
})
