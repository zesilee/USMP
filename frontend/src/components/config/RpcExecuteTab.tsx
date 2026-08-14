import { useCallback, useEffect, useMemo, useState } from 'react'
import { Alert, Button, Tag, confirm } from '../../ui'
import { i18n } from '../../i18n'
import { executeRpc, getConfig } from '../../api'
import { extractRows } from '../../utils/extractRows'
import { parseLeafref } from '../../utils/leafref'
import { keyOf } from '../../form/configForm'
import type { Field } from '../../utils/crdSchemaParser'
import type { RpcDef } from '../../utils/moduleConsole'
import FieldRenderer from './FieldRenderer'
import './RpcExecuteTab.scss'

// RpcExecuteTab（FE-19/FE-20）：模型驱动 rpc 执行面板——input 复用 FieldRenderer；
// leafref 输入一律下拉（禁自由文本，目标值经 getConfig+extractRows 拉取，失败=空
// 下拉 R08）；执行前二次确认、高危（ext:execution-warn）升级危险样式；ExecuteRPC
// 有副作用不重试、结果/错误如实回显。语义自旧 Vue 版逐段平移。
const t = (k: string, p?: Record<string, unknown>) => i18n.global.t(k, p)

export interface RpcExecuteTabProps {
  rpc: RpcDef
  module: string
  device: string
}

export default function RpcExecuteTab({ rpc, module, device }: RpcExecuteTabProps) {
  const [values, setValues] = useState<Record<string, any>>({})
  const [leafrefOptions, setLeafrefOptions] = useState<Record<string, { label: string; value: string }[]>>({})
  const [running, setRunning] = useState(false)
  const [result, setResult] = useState('')
  const [resultType, setResultType] = useState<'success' | 'error'>('success')

  // rpc/设备切换：清空输入与结果。
  useEffect(() => {
    setValues({})
    setResult('')
  }, [device, rpc.name])

  // 拉取每个 leafref 输入的目标列表 → 下拉选项（FE-19）。
  useEffect(() => {
    setLeafrefOptions({})
    if (!device) return
    let alive = true
    void (async () => {
      const next: Record<string, { label: string; value: string }[]> = {}
      for (const f of rpc.input) {
        const target = parseLeafref(f.leafRef)
        if (!target) continue
        try {
          const res = await getConfig(device, target.fetchPath)
          const rows = extractRows(res.data?.data, target.listKey, target.keyField)
          const opts = rows
            .map((r) => String(r[target.keyField] ?? ''))
            .filter((v) => v !== '')
            .map((v) => ({ label: v, value: v }))
          if (opts.length) next[keyOf(f)] = opts
        } catch {
          /* 拉取失败=空选项 → 空下拉（R08） */
        }
      }
      if (alive) setLeafrefOptions(next)
    })()
    return () => {
      alive = false
    }
  }, [device, rpc])

  // 渲染用输入字段：leafref 注入 options（FieldRenderer 对 leafRef 恒下拉）。
  const renderFields = useMemo<Field[]>(
    () =>
      rpc.input.map((f) =>
        f.leafRef ? { ...f, options: leafrefOptions[keyOf(f)] ?? [] } : f,
      ),
    [rpc.input, leafrefOptions],
  )

  // 所有 mandatory input 有值才可执行（§9）。
  const submittable = useMemo(
    () =>
      rpc.input.every((f) => {
        if (!f.required) return true
        const v = values[keyOf(f)]
        return v !== undefined && v !== null && String(v).trim() !== ''
      }),
    [rpc.input, values],
  )

  const collectInputs = useCallback((): Record<string, string> => {
    const out: Record<string, string> = {}
    for (const f of rpc.input) {
      const v = values[keyOf(f)]
      if (v !== undefined && v !== null && String(v) !== '') out[keyOf(f)] = String(v)
    }
    return out
  }, [rpc.input, values])

  // 执行：先二次确认（高危升级危险样式），确认后调执行 API（不重试），回显结果。
  const execute = async () => {
    if (!device || !submittable || running) return
    const inputs = collectInputs()
    const detail = Object.entries(inputs).map(([k, v]) => `${k} = ${v}`).join('，') || t('console.rpc.noInput')
    const ok = await confirm(
      t('console.rpc.confirmBody', { rpc: rpc.label || rpc.name, device, inputs: detail }),
      { title: t('console.rpc.confirmTitle'), danger: !!rpc.highRisk, okText: t('console.rpc.confirmOk') },
    )
    if (!ok) return // 取消 → 不下发

    setRunning(true)
    setResult('')
    try {
      const res = await executeRpc(device, module, rpc.name, inputs)
      const body: any = res.data
      if (body?.success === false) {
        setResultType('error')
        setResult(body?.message || t('console.rpc.failed'))
      } else {
        setResultType('success')
        const reply = body?.data?.reply
        setResult(
          reply && String(reply).trim()
            ? t('console.rpc.doneReply', { reply: String(reply).slice(0, 300) })
            : t('console.rpc.done'),
        )
      }
    } catch (e: any) {
      setResultType('error')
      setResult(e?.response?.data?.message || e?.message || t('console.rpc.failed'))
    } finally {
      setRunning(false)
    }
  }

  return (
    <div className="rpc-execute-tab" data-test="rpc-execute-tab">
      <div className="rpc-head">
        <span className="rpc-name">{rpc.label || rpc.name}</span>
        {rpc.highRisk && (
          <Tag color="red" data-test="rpc-highrisk">
            {t('console.rpc.highRisk')}
          </Tag>
        )}
      </div>
      <p className="rpc-tip">{t('console.rpc.tip')}</p>

      {renderFields.length === 0 ? (
        <span className="rpc-empty">{t('console.rpc.noInput')}</span>
      ) : (
        <div className="rpc-inputs">
          {renderFields.map((f) => (
            <div key={f.path} className="sub-field">
              <label className="field-label">
                {f.label}
                {f.required && <span className="req-mark">*</span>}
              </label>
              <FieldRenderer
                field={f}
                value={values[keyOf(f)]}
                onChange={(v) => setValues((prev) => ({ ...prev, [keyOf(f)]: v }))}
              />
            </div>
          ))}
        </div>
      )}

      <div className="actions">
        <Button
          type="primary"
          danger={!!rpc.highRisk}
          loading={running}
          disabled={!device || !submittable}
          onClick={() => void execute()}
          data-test="rpc-execute"
        >
          {t('console.rpc.execute')}
        </Button>
      </div>

      {result && <Alert type={resultType} showIcon message={result} data-test="rpc-result" />}
    </div>
  )
}
