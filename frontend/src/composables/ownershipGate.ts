import { ElMessageBox } from 'element-plus'

// FE-18 二期：归属硬锁阻断确认流。
// 后端信封恒 HTTP 200——409 归属拒绝以 resolved 响应到达，调用方须按 code 分支：
// 识别 → 弹阻断确认 → 确认后携 force=true 重发，取消则静默中止（不置错误态）。

export interface OwnershipRejection {
  intents: string[]
  message: string
}

// ownershipRejectionOf 识别信封 409 归属拒绝；其他响应（成功/普通错误/畸形体）一律
// 返回 null，不劫持既有错误处理。
export function ownershipRejectionOf(res: unknown): OwnershipRejection | null {
  const env = (res as { data?: { code?: number; message?: string; data?: { intents?: unknown } } })?.data
  if (env?.code !== 409 || !Array.isArray(env?.data?.intents)) return null
  return { intents: env.data!.intents as string[], message: env.message || '' }
}

// confirmOwnershipOverride 弹阻断确认框：列出认领意图 + 覆盖警示。确认返回 true
// （调用方带 force=true 重发），取消返回 false（中止流程，不抛错）。
export async function confirmOwnershipOverride(rej: OwnershipRejection): Promise<boolean> {
  try {
    await ElMessageBox.confirm(
      `该路径由业务配置 ${rej.intents.join('、')} 管理，意图收敛时会覆盖本次手工修改。确定强制下发？`,
      '路径已被业务配置认领',
      { confirmButtonText: '强制下发', cancelButtonText: '取消', type: 'warning' },
    )
    return true
  } catch {
    return false
  }
}
