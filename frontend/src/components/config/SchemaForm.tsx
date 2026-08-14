import { Form, icons } from '../../ui'
import type { Field } from '../../utils/crdSchemaParser'
import { fieldValidation } from '../../form/antdRules'
import type { useConfigForm } from '../../hooks/useConfigForm'
import FieldRenderer from './FieldRenderer'
import './SchemaForm.scss'

// SchemaForm（FE-02/FE-22 骨架）：字段面 → Form.Item（label + 受控校验展示）+
// FieldRenderer 控件。单一数据源 = useConfigForm（Form.Item 不给 name、不接
// antd Form store——双源同步是 bug 温床，见 form/antdRules 架构注）。
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

export default function SchemaForm({ fields, form, keyField = '', fieldDisabled, labelExtra }: SchemaFormProps) {
  const visible = new Set(form.visibleFields.map((f) => f.path))
  const shown = fields.filter((f) => visible.has(f.path))
  return (
    <Form layout="vertical" className="schema-form" component="div">
      <div className="schema-form--grid">
        {shown.map((field) => {
          const v = fieldValidation(field, fields, form.formData)
          const requiredMark = !!(
            (field.required && !field.dynamicDefault) ||
            (keyField && form.keyOf(field) === keyField)
          )
          return (
            <Form.Item
              key={field.path}
              label={
                <span className="fi-label">
                  {field.isKey && <icons.KeyIcon className="key-icon" data-test="key-icon" />}
                  <span>{field.label}</span>
                  {labelExtra?.(field)}
                </span>
              }
              required={requiredMark}
              validateStatus={v.status}
              help={v.help}
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
            </Form.Item>
          )
        })}
      </div>
    </Form>
  )
}
