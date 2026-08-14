import { confirm } from '../ui'
import { i18n } from '../i18n'

// 归属硬锁门（BR-11 二期）：commit 整体 409 携 intents → 阻断确认 → force 重发。
// 判定走结构化字段（code=409 + data.intents），禁文案匹配。

export interface OwnershipRejection {
  intents: string[]
  message: string
}

export function ownershipRejectionOf(res: unknown): OwnershipRejection | null {
  const env = (res as { data?: { code?: number; message?: string; data?: { intents?: unknown } } })?.data
  if (env?.code !== 409 || !Array.isArray(env?.data?.intents)) return null
  return { intents: env.data!.intents as string[], message: env.message || '' }
}

export async function confirmOwnershipOverride(rej: OwnershipRejection): Promise<boolean> {
  const t = i18n.global.t
  return confirm(t('console.ownership.confirm', { intents: rej.intents.join('、') }), {
    title: t('console.ownership.title'),
    okText: t('console.ownership.forcePush'),
    danger: true,
  })
}
