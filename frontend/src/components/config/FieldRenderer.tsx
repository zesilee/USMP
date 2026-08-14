import { useState } from 'react'
import {
  Input,
  InputNumber,
  Radio,
  Segmented,
  Select,
  Switch,
  Tabs,
  Button,
  icons,
} from '../../ui'
import { i18n } from '../../i18n'
import type { Field } from '../../utils/crdSchemaParser'
import { keyOf } from '../../form/configForm'
import './FieldRenderer.scss'

// FieldRenderer（FE-01/R05 核心）：YANG 类型 → 控件的递归分派器。语义自旧 Vue 版
// 逐分支平移：8 种 Field.type 全覆盖、受控单向（value 进 / onChange 出）、嵌套层
// （group/list/choice）读子键改副本整体上抛。全部控件经 src/ui 适配层（FA-01）。
// 类型→控件结论（FE-01 spec 锚点，换库不变）：boolean→「打开/关闭」radio、
// 必填短枚举(≤3)→分段控件、其余 enum→下拉、leafref→禁降级文本框的可搜索下拉、
// presence group→开关+子表单、list/leaf-list→可增删行。

export interface FieldRendererProps {
  field: Field
  value: any
  onChange: (next: any) => void
  /** 外部禁用（编辑态 create-only 标识字段、readonly state 叶等，FE-11/14）。 */
  disabled?: boolean
}

const t = (k: string, p?: Record<string, unknown>) => i18n.global.t(k, p)

// ===== 占位提示（FE-15/FE-22）：显式 placeholder > dynamicDefault「系统自动分配」>
// 约束元数据合成（数值 range / 字符串 length）+ 默认值段 =====
function constraintPlaceholder(f: Field): string | undefined {
  let base: string | undefined
  if (f.type === 'number') {
    const { minimum: min, maximum: max } = f
    if (min != null && max != null) base = t('console.rangeBoth', { min, max })
    else if (min != null) base = t('console.rangeMin', { min })
    else if (max != null) base = t('console.rangeMax', { max })
  } else if (f.type === 'string') {
    const { minLength: min, maxLength: max } = f
    if (min != null && max != null) base = t('console.lengthBoth', { min, max })
    else if (min != null) base = t('console.lengthMin', { min })
    else if (max != null) base = t('console.lengthMax', { max })
  }
  // 默认值段：default 原样字符串化不本地化（设备语义值）；boolean 无占位语义。
  const dv = f.default == null || f.type === 'boolean' ? undefined : String(f.default)
  if (base && dv !== undefined) return base + t('console.defaultSuffix', { v: dv })
  if (base) return base
  if (dv !== undefined) return t('console.defaultOnly', { v: dv })
  return undefined
}

function placeholderOf(f: Field): string | undefined {
  return f.placeholder || (f.dynamicDefault ? t('console.autoAssigned') : undefined) || constraintPlaceholder(f)
}

// ===== 子字段公共渲染（group/list/choice 行内）=====
function SubFields({
  fields,
  scope,
  onField,
  disabled,
}: {
  fields: Field[]
  scope: Record<string, any>
  onField: (key: string, v: any) => void
  disabled?: boolean
}) {
  return (
    <div className="group-fields">
      {fields.map((sub) => (
        <div key={sub.path} className="sub-field">
          <label className="field-label">{sub.label}</label>
          <FieldRenderer
            field={sub}
            disabled={disabled}
            value={scope[keyOf(sub)]}
            onChange={(v) => onField(keyOf(sub), v)}
          />
        </div>
      ))}
    </div>
  )
}

export default function FieldRenderer({ field, value, onChange, disabled }: FieldRendererProps) {
  const off = field.readonly || disabled

  // ===== choice 激活分支（显式选择 > 有数据的 case > 首个）=====
  const [activeCaseName, setActiveCaseName] = useState<string | null>(null)
  if (field.type === 'choice') {
    const scope: Record<string, any> = (value as Record<string, any>) || {}
    const cases = field.cases || []
    const caseKeys = (c: { fields?: Field[] }) => (c.fields || []).map(keyOf)
    const caseHasData = (c: { fields?: Field[] }) =>
      caseKeys(c).some((k) => scope[k] !== undefined && scope[k] !== null && scope[k] !== '')
    const active =
      (activeCaseName && cases.some((c) => c.name === activeCaseName) && activeCaseName) ||
      cases.find(caseHasData)?.name ||
      cases[0]?.name ||
      ''
    // 切换 case：清空其它非激活 case 的成员键（YANG choice 互斥语义），整体上抛。
    const switchCase = (name: string) => {
      setActiveCaseName(name)
      const next: Record<string, any> = { ...scope }
      for (const c of cases) {
        if (c.name === name) continue
        for (const k of caseKeys(c)) delete next[k]
      }
      onChange(next)
    }
    const updateMember = (key: string, v: any) => onChange({ ...scope, [key]: v })
    const LEAF = ['string', 'number', 'boolean', 'enum']
    const usesRadio = cases.every((c) => c.fields?.length === 1 && LEAF.includes(c.fields[0].type))
    const activeFields = cases.find((c) => c.name === active)?.fields || []
    return (
      <div className="field-renderer field-choice">
        {usesRadio ? (
          <>
            <Radio.Group value={active} onChange={(e) => switchCase(String(e.target.value))} disabled={off}>
              {cases.map((c) => (
                <Radio key={c.name} value={c.name}>
                  {c.label}
                </Radio>
              ))}
            </Radio.Group>
            {activeFields.length > 0 && (
              <div className="choice-active-fields">
                <SubFields fields={activeFields} scope={scope} onField={updateMember} disabled={disabled} />
              </div>
            )}
          </>
        ) : (
          <Tabs
            activeKey={active}
            onChange={(k) => switchCase(k)}
            items={cases.map((c) => ({
              key: c.name,
              label: c.label,
              children: (
                <SubFields fields={c.fields || []} scope={scope} onField={updateMember} disabled={disabled} />
              ),
            }))}
          />
        )}
      </div>
    )
  }

  // ===== string：leafref/options → 可搜索下拉（禁降级文本框，选项空保持空态下拉）=====
  if (field.type === 'string' && (field.options?.length || field.leafRef)) {
    return (
      <div className="field-renderer">
        <Select
          value={value ?? undefined}
          onChange={(v) => onChange(v)}
          placeholder={placeholderOf(field)}
          disabled={off}
          allowClear
          showSearch
          style={{ width: '100%' }}
          data-test="leafref-select"
          options={(field.options || []).map((o) => ({ label: o.label, value: o.value }))}
          onClear={() => onChange(undefined)}
        />
      </div>
    )
  }

  if (field.type === 'string') {
    return (
      <div className="field-renderer field-scalar">
        <Input
          value={value ?? ''}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholderOf(field)}
          disabled={off}
        />
        {field.units && <span className="field-units">{field.units}</span>}
      </div>
    )
  }

  if (field.type === 'number') {
    return (
      <div className="field-renderer field-scalar">
        <InputNumber
          value={value ?? null}
          onChange={(v) => onChange(v ?? undefined)}
          placeholder={placeholderOf(field)}
          disabled={off}
          min={field.minimum}
          max={field.maximum}
          controls
          style={{ width: '100%' }}
        />
        {field.units && <span className="field-units">{field.units}</span>}
      </div>
    )
  }

  // boolean →「打开/关闭」radio（FE-01 NCE 形态）：值仍 true/false；可选 boolean
  // 未选 = 两项均不选中（键不入 payload 由表单层保证）。
  if (field.type === 'boolean') {
    return (
      <div className="field-renderer">
        <Radio.Group
          value={value ?? undefined}
          onChange={(e) => onChange(e.target.value)}
          disabled={off}
        >
          <Radio value={true}>{t('console.boolOn')}</Radio>
          <Radio value={false}>{t('console.boolOff')}</Radio>
        </Radio.Group>
      </div>
    )
  }

  if (field.type === 'enum') {
    const n = field.options?.length ?? 0
    // 必填且 ≤3 选项 → 分段控件（segmented 无清空能力，可选枚举须保留
    // 「清空=键不入 payload」语义，故仅必填走此分支）；零选项降级下拉（R08）。
    if (field.required && n > 0 && n <= 3) {
      return (
        <div className="field-renderer">
          <Segmented
            value={value ?? (field.options![0].value as any)}
            onChange={(v) => onChange(v)}
            options={(field.options || []).map((o) => ({ label: o.label, value: o.value as any }))}
            disabled={off}
          />
        </div>
      )
    }
    return (
      <div className="field-renderer">
        <Select
          value={value ?? undefined}
          onChange={(v) => onChange(v)}
          placeholder={placeholderOf(field)}
          disabled={off}
          allowClear
          style={{ width: '100%' }}
          options={(field.options || []).map((o) => ({ label: o.label, value: o.value }))}
          onClear={() => onChange(undefined)}
        />
      </div>
    )
  }

  // presence group：存在即开关。关闭 → onChange(undefined)（键不入 payload，
  // 节点不存在 FE-12/FE-27）；开启 → 保留/新建对象并展开子表单。
  if (field.type === 'group' && field.presence) {
    const on = value != null
    const kids = field.fields || []
    return (
      <div className="field-renderer field-presence">
        <Switch
          checked={on}
          disabled={off}
          onChange={(next) => onChange(next ? { ...(value || {}) } : undefined)}
        />
        {on && kids.length > 0 && (
          <div className="field-group presence-fields">
            <SubFields
              fields={kids}
              scope={(value as Record<string, any>) || {}}
              onField={(k, v) => onChange({ ...((value as Record<string, any>) || {}), [k]: v })}
              disabled={disabled}
            />
          </div>
        )}
      </div>
    )
  }

  if (field.type === 'group') {
    return (
      <div className="field-renderer field-group">
        <SubFields
          fields={field.fields || []}
          scope={(value as Record<string, any>) || {}}
          onField={(k, v) => onChange({ ...((value as Record<string, any>) || {}), [k]: v })}
          disabled={disabled}
        />
      </div>
    )
  }

  // leaf-list：可增删的标量数组（元素为字符串/数字/枚举值）。
  if (field.type === 'leaf-list') {
    const items: any[] = Array.isArray(value) ? value : []
    const update = (idx: number, v: any) => onChange(items.map((x, i) => (i === idx ? v : x)))
    return (
      <div className="field-renderer field-list">
        {items.map((item, idx) => (
          <div key={idx} className="list-row leaf-list-row">
            {field.options?.length ? (
              <Select
                value={item ?? undefined}
                onChange={(v) => update(idx, v)}
                disabled={off}
                allowClear
                style={{ width: '100%' }}
                options={field.options.map((o) => ({ label: o.label, value: o.value }))}
              />
            ) : (
              <Input
                value={item ?? ''}
                onChange={(e) => update(idx, e.target.value)}
                placeholder={field.placeholder}
                disabled={off}
              />
            )}
            <Button
              type="link"
              danger
              size="small"
              icon={<icons.DeleteIcon />}
              onClick={() => onChange(items.filter((_, i) => i !== idx))}
            >
              {t('common.delete')}
            </Button>
          </div>
        ))}
        <Button type="primary" size="small" ghost icon={<icons.PlusIcon />} onClick={() => onChange([...items, ''])}>
          {t('console.addItem', { label: field.label })}
        </Button>
      </div>
    )
  }

  // list：可增删行的嵌套子表单。
  if (field.type === 'list') {
    const rows: Record<string, any>[] = Array.isArray(value) ? value : []
    const kids = field.fields || []
    const updateRow = (idx: number, key: string, v: any) =>
      onChange(rows.map((r, i) => (i === idx ? { ...r, [key]: v } : r)))
    return (
      <div className="field-renderer field-list">
        {rows.map((row, idx) => (
          <div key={idx} className="list-row">
            <div className="list-row-fields">
              <SubFields
                fields={kids}
                scope={row || {}}
                onField={(k, v) => updateRow(idx, k, v)}
                disabled={disabled}
              />
            </div>
            <Button
              type="link"
              danger
              size="small"
              icon={<icons.DeleteIcon />}
              onClick={() => onChange(rows.filter((_, i) => i !== idx))}
            >
              {t('common.delete')}
            </Button>
          </div>
        ))}
        <Button type="primary" size="small" ghost icon={<icons.PlusIcon />} onClick={() => onChange([...rows, {}])}>
          {t('console.addItem', { label: field.label })}
        </Button>
      </div>
    )
  }

  // 未知类型降级文本框（R08 不崩）。
  return (
    <div className="field-renderer">
      <Input value={value ?? ''} onChange={(e) => onChange(e.target.value)} disabled={off} />
    </div>
  )
}
