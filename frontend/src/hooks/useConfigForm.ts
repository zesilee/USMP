import { useCallback, useState } from 'react'
import type { Field } from '../utils/crdSchemaParser'
import {
  keyOf,
  choiceMemberFields,
  visibleFields,
  flatFields,
  editableFlat,
  mustViolations,
  patternViolations,
  computeFormDiff,
  clearedKeys,
  isBlocked,
  isSubmittable,
  visiblePayload,
  changedPayload,
  snapshotBaseline,
  type ConfigFormOptions,
} from '../form/configForm'

// 模型驱动表单编排 hook（React 外壳）：状态 = formData + original 基线；全部派生
// 结论每渲染直调 form/configForm 纯函数重算（design D5：不做 memo，杜绝依赖数组
// 漏项的陈旧值）。**键存在性即节点存在性（FE-27/design D6）**：删键一律解构真删，
// SHALL NOT `{...prev, [k]: undefined}`（键残留会静默下发多余字段，真机
// unknown-element 拒绝——守护测试与 FE-27 套件双防线）。

export function useConfigForm(fields: Field[], keyField = '', opts?: ConfigFormOptions) {
  const [formData, setFormData] = useState<Record<string, any>>({})
  const [original, setOriginal] = useState<Record<string, any>>({})

  /** 写单字段；value === undefined 即删键（节点不存在，FE-27）。 */
  const setField = useCallback((key: string, value: any) => {
    setFormData((prev) => {
      if (value === undefined) {
        if (!(key in prev)) return prev
        const { [key]: _drop, ...rest } = prev
        return rest
      }
      return { ...prev, [key]: value }
    })
  }, [])

  /** 显式删键（字段级清除等场景语义化入口）。 */
  const removeField = useCallback((key: string) => setField(key, undefined), [setField])

  /** choice 成员 scope：承载成员键的扁平对象（FieldRenderer choice 契约）。 */
  const choiceScope = useCallback(
    (field: Field): Record<string, any> => {
      const o: Record<string, any> = {}
      for (const k of choiceMemberFields(field).map(keyOf)) if (k in formData) o[k] = formData[k]
      return o
    },
    [formData],
  )

  /** choice 整体回写：next 中 undefined/缺失的成员键从 formData 真删（互斥语义）。 */
  const onChoiceUpdate = useCallback((field: Field, next: Record<string, any>) => {
    setFormData((prev) => {
      const out = { ...prev }
      for (const k of choiceMemberFields(field).map(keyOf)) {
        if (next[k] === undefined) delete out[k]
        else out[k] = next[k]
      }
      return out
    })
  }, [])

  /** mode/row 变化时整体重置（切行/切建）：基线深快照独立于 formData。 */
  const resetForm = useCallback((seed: Record<string, any> = {}) => {
    setFormData({ ...seed })
    setOriginal(snapshotBaseline(seed))
  }, [])

  /** 外部合并只读状态等场景：函数式补丁（内部仍须遵守删键解构）。 */
  const patchForm = useCallback((patch: (prev: Record<string, any>) => Record<string, any>) => {
    setFormData((prev) => patch(prev))
  }, [])

  return {
    formData,
    original,
    setField,
    removeField,
    choiceScope,
    onChoiceUpdate,
    resetForm,
    patchForm,
    setOriginal,
    keyOf,
    // ===== 派生结论（每渲染重算，D5）=====
    visibleFields: visibleFields(fields, formData),
    flatFields: flatFields(fields, formData),
    editableFlat: editableFlat(fields, formData),
    mustViolations: mustViolations(fields, formData),
    patternViolations: patternViolations(fields, formData),
    diff: computeFormDiff(fields, formData, original, opts),
    clearedKeys: clearedKeys(fields, formData, original, opts),
    blocked: isBlocked(fields, formData, keyField),
    submittable: isSubmittable(fields, formData, original, keyField, opts),
    visiblePayload: () => visiblePayload(fields, formData),
    changedPayload: () => changedPayload(fields, formData, original, keyField),
  }
}
