import { useCallback, useEffect, useMemo, useState } from 'react'
import { Alert, Button, Empty, icons, toast } from '../../ui'
import { i18n } from '../../i18n'
import { getConfig } from '../../api'
import { useChangesetStore } from '../../stores/changeset'
import { useConfigForm } from '../../hooks/useConfigForm'
import { nodeUnsupportedFromEnvelope, nodeUnsupportedFromError } from '../../utils/nodeSupport'
import { configPathFor, type ConsoleTab } from '../../utils/moduleConsole'
import type { Field } from '../../utils/crdSchemaParser'
import SchemaForm from './SchemaForm'
import DiffPreview from './DiffPreview'
import './ModuleFormTab.scss'

// ModuleFormTab（FE-10/FE-14/FE-24）：非 list 的表单 Tab——取数回填（只读 Tab 走
// <get> 状态通道：get-config 对 config=false 子树会被真机 unknown-element 拒绝）、
// 节点不支持占位态（预标记/运行中学习，重试 force 逃生）、「确定=入变更集」攒批
// （只发改动字段，真机能力裁剪回归）。语义自旧 Vue 版逐段平移。
const t = (k: string, p?: Record<string, unknown>) => i18n.global.t(k, p)

export interface ModuleFormTabProps {
  tab: ConsoleTab
  rootName: string
  device: string
  /** schema 预标记不支持（CN-05：进控制台零请求直接占位）。 */
  unsupported?: boolean
  onUnsupportedChange?: (unsupported: boolean) => void
}

export default function ModuleFormTab({ tab, rootName, device, unsupported, onUnsupportedChange }: ModuleFormTabProps) {
  const fields = useMemo<Field[]>(() => tab.field.fields || [], [tab.field])
  const configPath = useMemo(() => configPathFor(rootName, tab.field.path), [rootName, tab.field.path])
  const form = useConfigForm(fields, '', { removals: true })
  const changeset = useChangesetStore()

  const [error, setError] = useState('')
  const [nodeUnsupported, setNodeUnsupported] = useState(!!unsupported)
  useEffect(() => setNodeUnsupported(!!unsupported), [unsupported])
  useEffect(() => onUnsupportedChange?.(nodeUnsupported), [nodeUnsupported, onUnsupportedChange])

  const { resetForm, keyOf } = form

  const load = useCallback(
    async (force = false) => {
      setError('')
      if (!device) {
        resetForm()
        return
      }
      if (nodeUnsupported && !force) return // 预标记/已学习：不打设备（FE-24）
      try {
        const res = await getConfig(device, configPath, force, !!tab.readonly)
        if (nodeUnsupportedFromEnvelope(res.data)) {
          setNodeUnsupported(true)
          return
        }
        setNodeUnsupported(false)
        const payload = res.data?.data
        const subtree = payload?.data ?? payload ?? {}
        const seed: Record<string, any> = {}
        for (const f of fields) {
          const k = keyOf(f)
          if (subtree[k] !== undefined) seed[k] = subtree[k]
        }
        resetForm(seed)
      } catch (e: any) {
        if (nodeUnsupportedFromError(e)) {
          setNodeUnsupported(true)
          return
        }
        // 后端暂不支持该路径读时如实降级：空表单 + 告警（§9，不伪装成功）。
        resetForm()
        setError(e?.response?.data?.message || e?.message || t('console.readFailed'))
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [device, configPath, tab.readonly, fields, nodeUnsupported],
  )

  useEffect(() => {
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [device, configPath])

  // ===== 确定=入变更集（FE-03 攒批）：不发写请求 =====
  const submit = () => {
    if (!device || form.blocked) return
    changeset.upsert(device, {
      op: 'update',
      path: '/' + configPath,
      // 只发改动字段（真机 unknown-element 回归）。
      payload: form.changedPayload(),
      cleared: form.clearedKeys,
      baseline: { ...form.original },
      label: tab.label,
    })
    toast(t('console.stagedOk'))
  }

  if (nodeUnsupported) {
    return (
      <div className="module-form-tab" data-test="node-unsupported">
        <Empty description={t('console.nodeUnsupported')}>
          <Button size="small" icon={<icons.RefreshIcon />} onClick={() => void load(true)}>
            {t('common.retry')}
          </Button>
        </Empty>
      </div>
    )
  }

  return (
    <div className="module-form-tab" data-test="module-form-tab">
      {error && <Alert type="warning" showIcon message={error} />}
      <SchemaForm fields={fields} form={form} />
      {!tab.readonly ? (
        <>
          <DiffPreview diff={form.diff} />
          <div className="actions">
            <Button type="primary" disabled={!device || !form.submittable} onClick={submit} data-test="form-stage">
              {t('console.stageChange')}
            </Button>
            <Button onClick={() => void load(true)}>{t('console.fetchSource')}</Button>
          </div>
        </>
      ) : (
        // 整 Tab readonly（config false state 子树）：只读视图，无下发入口（FE-14）。
        <span className="form-tip">{t('console.readonlyTip')}</span>
      )}
    </div>
  )
}
