// 设备表单校验（纯函数）——提交闸门的唯一事实源。
//
// 为什么不直接用 el-form 的 validate() 作闸门：对话框内表单的模板 ref 在
// teleport 场景下可能指向与页面上不同的实例（F2 测试实测：ref 实例 ≠ 活实例，
// validate() 因字段列表为空而恒 resolve，空表单被放行）。纯函数与渲染无关，
// 生产与测试行为一致，且可 F1 单测。el-form 的 rules 只负责行内错误提示。

/** 设备表单的原始输入（端口为字符串，来自输入框）。 */
export interface DeviceFormInput {
  ip: string
  port: string
  username: string
  password: string
  vendor: string
  role: string
}

/** 字段名 → 错误消息 key（i18n 键，调用方翻译）。 */
export type DeviceFormErrors = Partial<Record<keyof DeviceFormInput, string>>

const IPV4_RE = /^(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}$/
// 角色与后端 BR-14 同口径：≤32 位 [A-Za-z0-9_-]
const ROLE_RE = /^[A-Za-z0-9_-]{1,32}$/

/** 校验设备表单，返回错误映射（空对象 = 通过）。 */
export function validateDeviceForm(form: DeviceFormInput): DeviceFormErrors {
  const errors: DeviceFormErrors = {}

  const ip = form.ip.trim()
  if (!ip) errors.ip = 'devices.ruleIpRequired'
  else if (!IPV4_RE.test(ip)) errors.ip = 'devices.ruleIpInvalid'

  const port = form.port.trim()
  if (port) {
    const n = Number(port)
    if (!Number.isInteger(n) || n < 1 || n > 65535) errors.port = 'devices.rulePortInvalid'
  }

  if (!form.username.trim()) errors.username = 'devices.ruleUserRequired'
  if (!form.password) errors.password = 'devices.rulePassRequired'

  const role = form.role.trim()
  if (role && !ROLE_RE.test(role)) errors.role = 'devices.ruleRoleInvalid'

  return errors
}

/** 表单 → 后端 AddDeviceRequest 载荷（空值不发，由后端兜底缺省）。 */
export function toAddDevicePayload(form: DeviceFormInput) {
  const port = form.port.trim()
  const vendor = form.vendor.trim()
  const role = form.role.trim()
  return {
    ip: form.ip.trim(),
    port: port ? Number(port) : undefined,
    username: form.username.trim(),
    password: form.password,
    vendor: vendor || undefined,
    role: role || undefined,
  }
}
