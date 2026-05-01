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

/** 华为 VLAN 类型枚举 */
const VLAN_TYPE_OPTIONS = [
  { name: 'Common', value: 1, description: '普通 VLAN' },
  { name: 'Super', value: 2, description: '超级 VLAN' },
  { name: 'Sub', value: 3, description: '子 VLAN' },
  { name: 'Principal', value: 4, description: '主 VLAN (MUX)' },
  { name: 'Separate', value: 5, description: '隔离 VLAN (MUX)' },
  { name: 'Group', value: 6, description: '组 VLAN (MUX)' }
]

/** 启用状态枚举 */
const ENABLE_STATUS_OPTIONS = [
  { name: 'Disable', value: 0, description: '禁用' },
  { name: 'Enable', value: 1, description: '启用' }
]

/** 管理状态枚举 */
const ADMIN_STATUS_OPTIONS = [
  { name: 'Down', value: 0, description: '禁用' },
  { name: 'Up', value: 1, description: '启用' }
]

/** 华为 VLAN 模型 - 完整 config=true 属性 */
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
        // ========== 基础属性 ==========
        {
          path: '/vlans/vlan/id',
          name: 'id',
          type: 'uint',
          description: 'VLAN ID',
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
          length: { min: 1, max: 31 }
        },
        {
          path: '/vlans/vlan/description',
          name: 'description',
          type: 'string',
          description: 'VLAN 描述',
          config: true,
          length: { min: 1, max: 80 }
        },
        {
          path: '/vlans/vlan/type',
          name: 'type',
          type: 'enum',
          description: 'VLAN 类型',
          config: true,
          enumOptions: VLAN_TYPE_OPTIONS,
          default: 1 // common
        },
        {
          path: '/vlans/vlan/admin-status',
          name: 'admin-status',
          type: 'enum',
          description: '管理状态',
          config: true,
          enumOptions: ADMIN_STATUS_OPTIONS,
          default: 1 // up
        },

        // ========== 流量控制 ==========
        {
          path: '/vlans/vlan/broadcast-discard',
          name: 'broadcast-discard',
          type: 'enum',
          description: '丢弃广播包',
          config: true,
          enumOptions: ENABLE_STATUS_OPTIONS,
          default: 0 // disable
        },
        {
          path: '/vlans/vlan/unknown-multicast-discard',
          name: 'unknown-multicast-discard',
          type: 'enum',
          description: '丢弃未知组播包',
          config: true,
          enumOptions: ENABLE_STATUS_OPTIONS,
          default: 0 // disable
        },

        // ========== MAC 学习 ==========
        {
          path: '/vlans/vlan/mac-learning',
          name: 'mac-learning',
          type: 'enum',
          description: 'MAC 地址学习',
          config: true,
          enumOptions: ENABLE_STATUS_OPTIONS,
          default: 1 // enable
        },
        {
          path: '/vlans/vlan/mac-aging-time',
          name: 'mac-aging-time',
          type: 'uint',
          description: 'MAC 老化时间 (秒)，0 表示不老化',
          config: true,
          range: { min: 0, max: 1000000 }
        },

        // ========== 统计功能 ==========
        {
          path: '/vlans/vlan/statistic-enable',
          name: 'statistic-enable',
          type: 'enum',
          description: 'VLAN 统计收集',
          config: true,
          enumOptions: ENABLE_STATUS_OPTIONS,
          default: 0 // disable
        },
        {
          path: '/vlans/vlan/statistic-discard',
          name: 'statistic-discard',
          type: 'enum',
          description: 'BUM 丢弃统计 (需先启用统计)',
          config: true,
          enumOptions: ENABLE_STATUS_OPTIONS,
          default: 0 // disable
        },

        // ========== 关联 VLAN ID (leafref) ==========
        {
          path: '/vlans/vlan/super-vlan',
          name: 'super-vlan',
          type: 'leafref',
          description: '超级 VLAN ID (仅 Sub VLAN 生效)',
          config: true,
          range: { min: 1, max: 4094 }
        },

        // ========== 嵌套容器 - 未知单播丢弃 ==========
        {
          path: '/vlans/vlan/unknown-unicast-discard',
          name: 'unknown-unicast-discard',
          type: 'container',
          description: '未知单播丢弃配置',
          config: true,
          children: [
            {
              path: '/vlans/vlan/unknown-unicast-discard/discard',
              name: 'discard',
              type: 'enum',
              description: '丢弃未知单播包',
              config: true,
              enumOptions: ENABLE_STATUS_OPTIONS,
              default: 0 // disable
            },
            {
              path: '/vlans/vlan/unknown-unicast-discard/mac-learning-enable',
              name: 'mac-learning-enable',
              type: 'enum',
              description: '未知单播 MAC 学习 (需先启用丢弃)',
              config: true,
              enumOptions: ENABLE_STATUS_OPTIONS,
              default: 0 // disable
            }
          ]
        },

        // ========== 嵌套容器 - 流量抑制 ==========
        {
          path: '/vlans/vlan/suppression',
          name: 'suppression',
          type: 'container',
          description: '流量抑制配置',
          config: true,
          children: [
            {
              path: '/vlans/vlan/suppression/inbound',
              name: 'inbound',
              type: 'enum',
              description: '入方向抑制',
              config: true,
              enumOptions: ENABLE_STATUS_OPTIONS,
              default: 0 // disable
            },
            {
              path: '/vlans/vlan/suppression/outbound',
              name: 'outbound',
              type: 'enum',
              description: '出方向抑制',
              config: true,
              enumOptions: ENABLE_STATUS_OPTIONS,
              default: 0 // disable
            }
          ]
        },

        // ========== 只读属性 (config false) ==========
        {
          path: '/vlans/vlan/oper-status',
          name: 'oper-status',
          type: 'enum',
          description: '运行状态',
          config: false,
          enumOptions: [
            { name: 'ACTIVE', value: 1, description: '运行中' },
            { name: 'INACTIVE', value: 0, description: '未激活' }
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
    },

    // ========== VLAN Instances ==========
    {
      path: '/vlans/instances',
      name: 'instances',
      type: 'container',
      description: 'VLAN 实例配置',
      config: true,
      children: [
        {
          path: '/vlans/instances/instance',
          name: 'instance',
          type: 'list',
          description: 'VLAN 实例',
          key: 'id',
          config: true,
          children: [
            {
              path: '/vlans/instances/instance/id',
              name: 'id',
              type: 'uint',
              description: '实例 ID',
              config: true,
              mandatory: true,
              range: { min: 1, max: 4094 }
            },
            {
              path: '/vlans/instances/instance/vlan-list',
              name: 'vlan-list',
              type: 'string',
              description: 'VLAN 范围 (如 1-10,20,30)',
              config: true,
              mandatory: true
            }
          ]
        }
      ]
    }
  ]
}

/** 华为 IFM 接口管理模型 */
// Interface 常用枚举选项
const PORT_STATUS_OPTIONS = [
  { name: 'Down', value: 1, description: '关闭' },
  { name: 'Up', value: 2, description: '启用' }
]

const PORT_TYPE_OPTIONS = [
  { name: 'Ethernet', value: 1, description: '以太网接口' },
  { name: 'GigabitEthernet', value: 3, description: '千兆以太网接口' },
  { name: '100GE', value: 21, description: '100G 以太网接口' },
  { name: '40GE', value: 24, description: '40G 以太网接口' },
  { name: 'Eth-Trunk', value: 5, description: '链路聚合接口' },
  { name: 'Vlanif', value: 16, description: 'VLAN 接口' },
  { name: 'LoopBack', value: 20, description: '环回接口' },
  { name: 'Tunnel', value: 15, description: '隧道接口' }
]

const LINK_PROTOCOL_OPTIONS = [
  { name: 'Ethernet', value: 0, description: '以太网协议' },
  { name: 'PPP', value: 1, description: 'PPP 协议' },
  { name: 'HDLC', value: 2, description: 'HDLC 协议' }
]

const ROUTER_TYPE_OPTIONS = [
  { name: 'PtoP', value: 0, description: '点到点' },
  { name: 'NBMA', value: 1, description: '非广播多点接入' },
  { name: 'P2MP', value: 2, description: '点到多点' },
  { name: 'Broadcast', value: 3, description: '广播模式' }
]

const SERVICE_TYPE_OPTIONS = [
  { name: 'Unknown', value: 0, description: '未知' },
  { name: 'L3', value: 1, description: '三层接口' },
  { name: 'L2', value: 2, description: '二层接口' },
  { name: 'L3Main', value: 3, description: '三层主接口' }
]

const CLASS_TYPE_OPTIONS = [
  { name: 'Main', value: 1, description: '主接口' },
  { name: 'Sub', value: 2, description: '子接口' },
  { name: 'Tunnel', value: 3, description: '隧道接口' }
]

const STATISTIC_MODE_OPTIONS = [
  { name: 'Interface', value: 1, description: '接口统计' },
  { name: 'SubInterface', value: 2, description: '子接口统计' },
  { name: 'All', value: 3, description: '全部统计' }
]

export const INTERFACES_SCHEMA: YangNode = {
  path: '/ifm:ifm/ifm:interfaces',
  name: 'interfaces',
  type: 'container',
  description: '接口配置管理',
  config: true,
  children: [
    {
      path: '/ifm:ifm/ifm:interfaces/interface',
      name: 'interface',
      type: 'list',
      description: '接口列表',
      key: 'name',
      config: true,
      children: [
        // ===== 基础属性 =====
        {
          path: '/ifm:ifm/ifm:interfaces/interface/name',
          name: 'name',
          type: 'string',
          description: '接口名称',
          config: true,
          mandatory: true
        },
        {
          path: '/ifm:ifm/ifm:interfaces/interface/description',
          name: 'description',
          type: 'string',
          description: '接口描述',
          config: true
        },
        {
          path: '/ifm:ifm/ifm:interfaces/interface/index',
          name: 'index',
          type: 'uint',
          description: '接口索引',
          config: true
        },
        {
          path: '/ifm:ifm/ifm:interfaces/interface/number',
          name: 'number',
          type: 'string',
          description: '接口编号',
          config: true
        },
        {
          path: '/ifm:ifm/ifm:interfaces/interface/position',
          name: 'position',
          type: 'string',
          description: '接口位置',
          config: true
        },
        {
          path: '/ifm:ifm/ifm:interfaces/interface/parent-name',
          name: 'parent-name',
          type: 'string',
          description: '父接口名称',
          config: true
        },

        // ===== 状态和类型 =====
        {
          path: '/ifm:ifm/ifm:interfaces/interface/admin-status',
          name: 'admin-status',
          type: 'enum',
          description: '管理状态',
          config: true,
          enumOptions: PORT_STATUS_OPTIONS,
          default: 2
        },
        {
          path: '/ifm:ifm/ifm:interfaces/interface/type',
          name: 'type',
          type: 'enum',
          description: '接口类型',
          config: true,
          enumOptions: PORT_TYPE_OPTIONS,
          default: 1
        },
        {
          path: '/ifm:ifm/ifm:interfaces/interface/class',
          name: 'class',
          type: 'enum',
          description: '接口分类',
          config: true,
          enumOptions: CLASS_TYPE_OPTIONS,
          default: 1
        },
        {
          path: '/ifm:ifm/ifm:interfaces/interface/link-protocol',
          name: 'link-protocol',
          type: 'enum',
          description: '链路协议类型',
          config: true,
          enumOptions: LINK_PROTOCOL_OPTIONS,
          default: 0
        },
        {
          path: '/ifm:ifm/ifm:interfaces/interface/router-type',
          name: 'router-type',
          type: 'enum',
          description: '路由类型',
          config: true,
          enumOptions: ROUTER_TYPE_OPTIONS,
          default: 3
        },
        {
          path: '/ifm:ifm/ifm:interfaces/interface/service-type',
          name: 'service-type',
          type: 'enum',
          description: '服务类型',
          config: true,
          enumOptions: SERVICE_TYPE_OPTIONS,
          default: 1
        },

        // ===== 网络参数 =====
        {
          path: '/ifm:ifm/ifm:interfaces/interface/mtu',
          name: 'mtu',
          type: 'uint',
          description: 'MTU (最大传输单元)',
          config: true,
          range: { min: 64, max: 9216 },
          default: 1500
        },
        {
          path: '/ifm:ifm/ifm:interfaces/interface/mac-address',
          name: 'mac-address',
          type: 'string',
          description: 'MAC 地址',
          config: true
        },
        {
          path: '/ifm:ifm/ifm:interfaces/interface/bandwidth',
          name: 'bandwidth',
          type: 'uint',
          description: '带宽 (Mbps)',
          config: true,
          range: { min: 1, max: 1000000 }
        },
        {
          path: '/ifm:ifm/ifm:interfaces/interface/bandwidth-kbps',
          name: 'bandwidth-kbps',
          type: 'uint',
          description: '带宽 (Kbps)',
          config: true,
          range: { min: 1, max: 100000000 }
        },
        {
          path: '/ifm:ifm/ifm:interfaces/interface/vrf-name',
          name: 'vrf-name',
          type: 'string',
          description: 'VRF 名称',
          config: true
        },
        {
          path: '/ifm:ifm/ifm:interfaces/interface/vs-name',
          name: 'vs-name',
          type: 'string',
          description: 'VS 名称',
          config: true
        },

        // ===== 链路聚合 =====
        {
          path: '/ifm:ifm/ifm:interfaces/interface/aggregation-name',
          name: 'aggregation-name',
          type: 'string',
          description: '聚合接口名称',
          config: true
        },

        // ===== 定时器和延迟 =====
        {
          path: '/ifm:ifm/ifm:interfaces/interface/down-delay-time',
          name: 'down-delay-time',
          type: 'uint',
          description: 'Down 延迟时间 (秒)',
          config: true,
          range: { min: 0, max: 600 },
          default: 0
        },
        {
          path: '/ifm:ifm/ifm:interfaces/interface/protocol-up-delay-time',
          name: 'protocol-up-delay-time',
          type: 'uint',
          description: '协议 Up 延迟时间 (秒)',
          config: true,
          range: { min: 0, max: 600 },
          default: 0
        },

        // ===== 功能开关 =====
        {
          path: '/ifm:ifm/ifm:interfaces/interface/clear-ip-df',
          name: 'clear-ip-df',
          type: 'boolean',
          description: '清除 IP DF 标志',
          config: true,
          default: false
        },
        {
          path: '/ifm:ifm/ifm:interfaces/interface/is-l2-switch',
          name: 'is-l2-switch',
          type: 'boolean',
          description: '是否为二层交换口',
          config: true,
          default: false
        },
        {
          path: '/ifm:ifm/ifm:interfaces/interface/l2-mode-enable',
          name: 'l2-mode-enable',
          type: 'boolean',
          description: '启用二层模式',
          config: true,
          default: false
        },
        {
          path: '/ifm:ifm/ifm:interfaces/interface/link-up-down-trap-enable',
          name: 'link-up-down-trap-enable',
          type: 'boolean',
          description: '启用 Link Up/Down Trap',
          config: true,
          default: true
        },
        {
          path: '/ifm:ifm/ifm:interfaces/interface/statistic-enable',
          name: 'statistic-enable',
          type: 'boolean',
          description: '启用统计',
          config: true,
          default: true
        },
        {
          path: '/ifm:ifm/ifm:interfaces/interface/spread-mtu-flag',
          name: 'spread-mtu-flag',
          type: 'boolean',
          description: '传播 MTU 标志',
          config: true,
          default: false
        },

        // ===== 统计配置 =====
        {
          path: '/ifm:ifm/ifm:interfaces/interface/statistic-interval',
          name: 'statistic-interval',
          type: 'uint',
          description: '统计间隔 (秒)',
          config: true,
          range: { min: 10, max: 600 },
          default: 300
        },
        {
          path: '/ifm:ifm/ifm:interfaces/interface/statistic-mode',
          name: 'statistic-mode',
          type: 'enum',
          description: '统计模式',
          config: true,
          enumOptions: STATISTIC_MODE_OPTIONS,
          default: 1
        },

        // ===== 嵌套容器：ControlFlap =====
        {
          path: '/ifm:ifm/ifm:interfaces/interface/control-flap',
          name: 'control-flap',
          type: 'container',
          description: '接口振荡抑制配置',
          config: true,
          children: [
            {
              path: '/ifm:ifm/ifm:interfaces/interface/control-flap/ceiling',
              name: 'ceiling',
              type: 'uint',
              description: '抑制阈值上限',
              config: true,
              range: { min: 1, max: 20000 }
            },
            {
              path: '/ifm:ifm/ifm:interfaces/interface/control-flap/control-flap-count',
              name: 'control-flap-count',
              type: 'uint',
              description: '振荡次数统计',
              config: true
            },
            {
              path: '/ifm:ifm/ifm:interfaces/interface/control-flap/decay-ng',
              name: 'decay-ng',
              type: 'uint',
              description: '故障状态衰减系数',
              config: true,
              range: { min: 1, max: 900 }
            },
            {
              path: '/ifm:ifm/ifm:interfaces/interface/control-flap/decay-ok',
              name: 'decay-ok',
              type: 'uint',
              description: '正常状态衰减系数',
              config: true,
              range: { min: 1, max: 900 }
            },
            {
              path: '/ifm:ifm/ifm:interfaces/interface/control-flap/reuse',
              name: 'reuse',
              type: 'uint',
              description: '恢复阈值',
              config: true,
              range: { min: 1, max: 20000 }
            },
            {
              path: '/ifm:ifm/ifm:interfaces/interface/control-flap/suppress',
              name: 'suppress',
              type: 'uint',
              description: '抑制启动阈值',
              config: true,
              range: { min: 1, max: 20000 }
            }
          ]
        },

        // ===== 嵌套容器：Damp =====
        {
          path: '/ifm:ifm/ifm:interfaces/interface/damp',
          name: 'damp',
          type: 'container',
          description: '接口 Damp 抑制配置',
          config: true,
          children: [
            {
              path: '/ifm:ifm/ifm:interfaces/interface/damp/tx-off',
              name: 'tx-off',
              type: 'boolean',
              description: '关闭 TX 发送',
              config: true,
              default: false
            },
            {
              path: '/ifm:ifm/ifm:interfaces/interface/damp/auto',
              name: 'auto',
              type: 'container',
              description: '自动 Damp 配置',
              config: true,
              children: [
                {
                  path: '/ifm:ifm/ifm:interfaces/interface/damp/auto/level',
                  name: 'level',
                  type: 'enum',
                  description: 'Damp 等级',
                  config: true,
                  enumOptions: [
                    { name: 'Level 1', value: 1, description: '等级 1' },
                    { name: 'Level 2', value: 2, description: '等级 2' },
                    { name: 'Level 3', value: 3, description: '等级 3' },
                    { name: 'Level 4', value: 4, description: '等级 4' }
                  ]
                }
              ]
            },
            {
              path: '/ifm:ifm/ifm:interfaces/interface/damp/manual',
              name: 'manual',
              type: 'container',
              description: '手动 Damp 配置',
              config: true,
              children: [
                {
                  path: '/ifm:ifm/ifm:interfaces/interface/damp/manual/half-life-period',
                  name: 'half-life-period',
                  type: 'uint',
                  description: '半衰期 (秒)',
                  config: true,
                  range: { min: 1, max: 45 }
                },
                {
                  path: '/ifm:ifm/ifm:interfaces/interface/damp/manual/max-suppress-time',
                  name: 'max-suppress-time',
                  type: 'uint',
                  description: '最大抑制时间 (秒)',
                  config: true,
                  range: { min: 1, max: 255 }
                },
                {
                  path: '/ifm:ifm/ifm:interfaces/interface/damp/manual/reuse',
                  name: 'reuse',
                  type: 'uint',
                  description: '恢复阈值',
                  config: true,
                  range: { min: 1, max: 20000 }
                },
                {
                  path: '/ifm:ifm/ifm:interfaces/interface/damp/manual/suppress',
                  name: 'suppress',
                  type: 'uint',
                  description: '抑制启动阈值',
                  config: true,
                  range: { min: 1, max: 20000 }
                }
              ]
            }
          ]
        }
      ]
    }
  ]
}

/** Schema 注册表 */
/** 华为 System 系统配置模型 */
export const SYSTEM_SCHEMA: YangNode = {
  path: '/system:system',
  name: 'system',
  type: 'container',
  description: '系统配置管理',
  config: true,
  children: [
    {
      path: '/system:system/system:system-info',
      name: 'system-info',
      type: 'container',
      description: '系统基本信息',
      config: true,
      children: [
        {
          path: '/system:system/system:system-info/sys-name',
          name: 'sys-name',
          type: 'string',
          description: '系统名称',
          config: true,
          length: { min: 1, max: 246 },
          default: 'HUAWEI'
        },
        {
          path: '/system:system/system:system-info/sys-contact',
          name: 'sys-contact',
          type: 'string',
          description: '系统联系信息',
          config: true,
          length: { min: 1, max: 255 }
        },
        {
          path: '/system:system/system:system-info/sys-location',
          name: 'sys-location',
          type: 'string',
          description: '系统位置',
          config: true,
          length: { min: 1, max: 255 }
        },
        {
          path: '/system:system/system:system-info/sys-desc',
          name: 'sys-desc',
          type: 'string',
          description: '系统描述',
          config: false // 只读
        },
        {
          path: '/system:system/system:system-info/product-name',
          name: 'product-name',
          type: 'string',
          description: '产品名称',
          config: false // 只读
        },
        {
          path: '/system:system/system:system-info/product-version',
          name: 'product-version',
          type: 'string',
          description: '产品版本',
          config: false // 只读
        },
        {
          path: '/system:system/system:system-info/esn',
          name: 'esn',
          type: 'string',
          description: '设备序列号',
          config: false // 只读
        },
        {
          path: '/system:system/system:system-info/sys-uptime',
          name: 'sys-uptime',
          type: 'uint',
          description: '系统运行时间 (秒)',
          config: false // 只读
        }
      ]
    }
  ]
}

export const SCHEMA_REGISTRY: Record<string, YangNode> = {
  '/vlans': VLAN_SCHEMA,
  '/ifm:ifm/ifm:interfaces': INTERFACES_SCHEMA,
  '/system:system': SYSTEM_SCHEMA,
  // 向后兼容
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

// ============== 键名转换工具 ==============

/** kebab-case 转 camelCase */
export function kebabToCamel(str: string): string {
  return str.replace(/-([a-z])/g, (_, c) => c.toUpperCase())
}

/** camelCase 转 kebab-case */
export function camelToKebab(str: string): string {
  return str.replace(/([a-z])([A-Z])/g, '$1-$2').toLowerCase()
}

/** 递归转换对象的键名 - 方向: kebab -> camel */
export function convertKeysToCamel<T = any>(obj: any): T {
  if (obj === null || obj === undefined) return obj as T
  if (Array.isArray(obj)) return obj.map(item => convertKeysToCamel(item)) as unknown as T
  if (typeof obj !== 'object') return obj as T

  const result: Record<string, any> = {}
  for (const key in obj) {
    if (Object.prototype.hasOwnProperty.call(obj, key)) {
      const newKey = kebabToCamel(key)
      result[newKey] = convertKeysToCamel(obj[key])
    }
  }
  return result as T
}

/** 递归转换对象的键名 - 方向: camel -> kebab */
export function convertKeysToKebab<T = any>(obj: any): T {
  if (obj === null || obj === undefined) return obj as T
  if (Array.isArray(obj)) return obj.map(item => convertKeysToKebab(item)) as unknown as T
  if (typeof obj !== 'object') return obj as T

  const result: Record<string, any> = {}
  for (const key in obj) {
    if (Object.prototype.hasOwnProperty.call(obj, key)) {
      const newKey = camelToKebab(key)
      result[newKey] = convertKeysToKebab(obj[key])
    }
  }
  return result as T
}
