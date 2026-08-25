import { useCallback, useEffect, useMemo, useState } from 'react'
import { Breadcrumb, Button, Tabs, Tooltip, icons } from '../../ui'
import { i18n } from '../../i18n'
import { getConfig } from '../../api'
import { useChangesetStore } from '../../stores/changeset'
import { useConfigForm } from '../../hooks/useConfigForm'
import { snapshotBaseline } from '../../form/configForm'
import { mergeReadonlyState } from '../../utils/configState'
import type { Field } from '../../utils/crdSchemaParser'
import {
  deriveDetailTabs,
  deriveKeyField,
  configPathFor,
  leafName,
  type ConsoleTab,
} from '../../utils/moduleConsole'
import SchemaForm from './SchemaForm'
import DiffPreview from './DiffPreview'
import './ItemDetailPane.scss'

// ItemDetailPane（FE-21/FE-22）：列表行详情同屏编辑区——面包屑 + 二级 Tab
// （deriveDetailTabs）+ SchemaForm 字段面 + 字段级清除 + DiffPreview + 「暂存」
// 入变更集（FE-23 攒批：确定不发写请求）。语义自旧 Vue 版逐段平移：变更集回填
// 保持首次 baseline、include_state 单行状态读合并只读值、编辑态 key/identity
// 禁用、竞态防线（回包时行已切换即丢弃）。
const t = (k: string, p?: Record<string, unknown>) => i18n.global.t(k, p)

export interface ItemDetailPaneProps {
  tab: ConsoleTab
  rootName: string
  device: string
  mode: 'edit' | 'create'
  row: Record<string, any> | null
  /** POST 包裹键（回读命中容器名时跟随实际键，与列表侧同源）。 */
  postKey?: string
  onClose: () => void
  onStaged: (key: string) => void
  /** 未提交草稿态上报（FE-21 切行守卫数据源）。 */
  onDirtyChange?: (dirty: boolean) => void
}

export default function ItemDetailPane(props: ItemDetailPaneProps) {
  const { tab, rootName, device, mode, row, onClose, onStaged, onDirtyChange } = props
  const listField = tab.listField || tab.field
  const keyField = useMemo(() => deriveKeyField(listField), [listField])
  const detailTabs = useMemo(() => deriveDetailTabs(listField), [listField])
  // 变更集条目路径：后端 ChangesetReq 按锚点前缀匹配，须带前导斜杠。
  const configPath = useMemo(() => '/' + configPathFor(rootName, tab.field.path), [rootName, tab.field.path])
  const itemFields = useMemo(() => listField.fields || [], [listField])
  const postListKey = props.postKey || leafName(listField)

  // 攒批模式（FE-21/FE-03）：removals 开启——基线有值被清 = 删除意图入 diff。
  const form = useConfigForm(itemFields, keyField, { removals: true })
  const changeset = useChangesetStore()
  const [activeDetail, setActiveDetail] = useState('__main__')

  // 门禁（FE-11 编辑态 identity 禁用 + FE-22 key 编辑态只读）：readonly 恒禁用。
  const isFieldDisabled = useCallback(
    (f: Field): boolean => {
      if (f.readonly) return true
      return mode === 'edit' && (!!f.isKey || !!f.operationExclude?.includes('update'))
    },
    [mode],
  )

  const { resetForm, setOriginal, patchForm, removeField } = form

  // 未提交草稿态上报（FE-21）：diff 非空即 dirty。
  const dirty = form.diff.length > 0
  useEffect(() => onDirtyChange?.(dirty), [dirty, onDirtyChange])

  // mode/row 变化即重置表单（切行/切建）。变更集已有该条目 → 用最新值回填
  // 并保持首次 baseline（FE-21 合并语义）：payload 覆盖行数据、cleared 叶置空。
  useEffect(() => {
    const seed: Record<string, any> = row ? { ...row } : {}
    const pending = row
      ? changeset.entryFor(device, configPath, String(row[keyField] ?? ''))
      : undefined
    if (pending && pending.op !== 'delete') {
      Object.assign(seed, pending.payload ?? {})
      for (const k of pending.cleared ?? []) delete seed[k]
    }
    resetForm(seed)
    if (pending && pending.op !== 'delete') {
      // diff 基线锚定设备实际态（首次快照）：变更集回填值应呈现为「已改动」。
      setOriginal(snapshotBaseline((pending.baseline as Record<string, any>) ?? (row ? { ...row } : {})))
    }
    setActiveDetail('__main__')

    // 按需单行状态读（读通道拆分，真机回归）：列表 config-only 快通道不含
    // config=false 状态，编辑态按谓词只读该行合入只读值；失败静默降级（R08）。
    if (mode !== 'edit' || !device || !row) return
    const keyVal = String(row[keyField] ?? '')
    if (!keyVal) return
    let alive = true
    void (async () => {
      try {
        const res = await getConfig(device, `${configPath}/${postListKey}[${keyField}='${keyVal}']`, false, true)
        const sub = res.data?.data?.data ?? res.data?.data
        const v = sub?.[postListKey] ?? sub?.[leafName(listField)]
        const stateRow = Array.isArray(v)
          ? v[0]
          : v && typeof v === 'object'
            ? (Object.values(v)[0] as Record<string, any>)
            : undefined
        if (!alive) return // 竞态防线：回包时行已切换则丢弃
        patchForm((prev) => {
          const next = { ...prev }
          mergeReadonlyState(itemFields, next, stateRow)
          return next
        })
      } catch {
        /* 状态读失败不打扰编辑（§9 优雅降级） */
      }
    })()
    return () => {
      alive = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mode, row, device, configPath, keyField, postListKey])

  // 字段级清除（FE-22）：可编辑且有值才出清除钮；choice 成员键分散不整体清除。
  const clearableField = (f: Field): boolean => {
    if (f.type === 'choice' || isFieldDisabled(f)) return false
    return form.formData[form.keyOf(f)] !== undefined
  }
  // 清除 tooltip 按基线区分语义：有基线值=删除意图（提交后从设备删除），无=本地置空。
  const clearTipFor = (f: Field): string => {
    const baseVal = form.original[form.keyOf(f)]
    return baseVal !== undefined && baseVal !== null && String(baseVal) !== ''
      ? t('console.clearFieldTipDelete')
      : t('console.clearFieldTip')
  }

  const labelExtra = (f: Field) =>
    clearableField(f) ? (
      <Tooltip title={clearTipFor(f)}>
        <icons.DeleteIcon
          className="clear-icon"
          data-test={`clear-${form.keyOf(f)}`}
          onClick={(e) => {
            e.stopPropagation()
            removeField(form.keyOf(f))
          }}
        />
      </Tooltip>
    ) : null

  // 二级 Tab → 字段面：主 Tab 取非容器子叶、嵌套 Tab 取该单一子节点。
  const activeTab = detailTabs.find((d) => d.name === activeDetail) || detailTabs[0]
  const activeFields: Field[] = activeTab
    ? activeTab.name === '__main__'
      ? activeTab.field.fields || []
      : [activeTab.field]
    : []

  // 确定 = 写入变更集（FE-21 攒批）：不发任何写请求；提交经工具栏「提交配置」。
  const submit = () => {
    if (!device || form.blocked) return
    const keyValue = String(form.formData[keyField] ?? '')
    changeset.upsert(device, {
      op: mode === 'create' ? 'create' : 'update',
      path: configPath,
      listKey: postListKey,
      keyValue,
      // 主键+改动字段（真机 unknown-element 回归）：merge 稀疏语义只需改动集。
      payload: form.changedPayload(),
      cleared: form.clearedKeys,
      baseline: { ...form.original },
      label: `${tab.label} ${keyValue}`,
    })
    onStaged(keyValue)
  }

  const crumbLeaf = mode === 'create' ? t('common.create') : String(row?.[keyField] ?? '')

  return (
    <div className="item-detail-pane" data-test="item-detail-pane">
      <div className="detail-header">
        <Breadcrumb
          data-test="detail-breadcrumb"
          className="detail-crumb"
          separator=">"
          items={[{ title: tab.label }, { title: crumbLeaf }]}
        />
        <Button size="small" data-test="detail-close" onClick={onClose}>
          {t('common.close')}
        </Button>
      </div>

      {detailTabs.length > 1 && (
        <Tabs
          className="detail-tabs"
          activeKey={activeDetail}
          onChange={setActiveDetail}
          items={detailTabs.map((d) => ({ key: d.name, label: d.label }))}
        />
      )}

      <SchemaForm
        fields={activeFields}
        form={form}
        keyField={keyField}
        fieldDisabled={isFieldDisabled}
        labelExtra={labelExtra}
      />

      <DiffPreview diff={form.diff} />

      <div className="detail-footer">
        <Button onClick={onClose}>{t('common.cancel')}</Button>
        <Button type="primary" data-test="detail-submit" disabled={!form.submittable} onClick={submit}>
          {t('console.stageChange')}
        </Button>
      </div>
    </div>
  )
}
