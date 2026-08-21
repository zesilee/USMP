import { FormItemShell, icons } from '../../ui'
import type { Field } from '../../utils/crdSchemaParser'
import { fieldValidation } from '../../form/antdRules'
import type { useConfigForm } from '../../hooks/useConfigForm'
import FieldRenderer from './FieldRenderer'
import './SchemaForm.scss'

// SchemaForm（FE-02/FE-22 骨架）：字段面 → FormItemShell（label + 受控校验
// 展示，适配层自绘外壳，组 5 接线自 antd Form.Item 迁入）+ FieldRenderer 控件。
// 单一数据源 = useConfigForm（外壳零校验行为、零表单 store——双源同步是 bug
// 温床，见 form/antdRules 架构注）。
// 可见性由 form.visibleFields（when 引擎）过滤——隐藏字段不渲染不校验不入
// payload；choice 走 choiceScope/onChoiceUpdate（成员扁平同级契约）；key 叶带
// 钥匙标识（FE-22，R12 真实图标）。提交门禁权威在 form.blocked（§9）。

export interface SchemaFormProps {
  fields: Field[]
  form: ReturnType<typeof useConfigForm>
  keyField?: string
  /** 编辑态禁用判定（isKey/operationExclude update 等，FE-11/22）。 */
  fieldDisabled?: (f: Field) => boolean
  /** label 尾部扩展（字段级清除钮等，FE-22）。 */
  labelExtra?: (f: Field) => React.ReactNode
}

const SCALAR = new Set(['string', 'number', 'boolean', 'enum'])

// NCE 编辑面对齐：呈现序=主键→*name→*description→其余保持 YANG 定义序
// （仅呈现层重排，派生逻辑/黄金零改动；payload 键序不受呈现序影响）。
function presentRank(f: Field): number {
  if (f.isKey) return 0
  const leaf = (f.path.split('/').pop() ?? f.path).toLowerCase()
  if (leaf === 'name' || leaf.endsWith('-name')) return 1
  if (leaf === 'description' || leaf === 'desc' || leaf.endsWith('-description')) return 2
  return 3
}

export default function SchemaForm({ fields, form, keyField = '', fieldDisabled, labelExtra }: SchemaFormProps) {
  const visible = new Set(form.visibleFields.map((f) => f.path))
  const shown = fields
    .filter((f) => visible.has(f.path))
    .map((f, i) => [f, i] as const)
    .sort((a, b) => presentRank(a[0]) - presentRank(b[0]) || a[1] - b[1])
    .map(([f]) => f)
  return (
    <div className="schema-form">
      <div className="schema-form--grid">
        {shown.map((field) => {
          const v = fieldValidation(field, fields, form.formData)
          const requiredMark = !!(
            (field.required && !field.dynamicDefault) ||
            (keyField && form.keyOf(field) === keyField)
          )
          return (
            <FormItemShell
              key={field.path}
              label={
                <span className="fi-label">
                  {field.isKey && <icons.KeyIcon className="key-icon" data-test="key-icon" />}
                  <span>{field.label}</span>
                </span>
              }
              labelExtra={labelExtra?.(field)}
              required={requiredMark}
              error={v.status === 'error' ? v.help : undefined}
              className={SCALAR.has(field.type) ? undefined : 'fi-span-full'}
            >
              {field.type === 'choice' ? (
                <FieldRenderer
                  field={field}
                  value={form.choiceScope(field)}
                  onChange={(next) => form.onChoiceUpdate(field, next)}
                />
              ) : (
                <FieldRenderer
                  field={field}
                  value={form.formData[form.keyOf(field)]}
                  onChange={(val) => form.setField(form.keyOf(field), val)}
                  disabled={fieldDisabled?.(field)}
                />
              )}
            </FormItemShell>
          )
        })}
      </div>
    </div>
  )
}
