import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import SchemaForm from '../../src/components/config/SchemaForm'
import { useConfigForm } from '../../src/hooks/useConfigForm'
import { UiProvider } from '../../src/ui'
import type { Field } from '../../src/utils/crdSchemaParser'

// NCE 编辑面对齐：字段呈现序=主键→*name→*description→其余按 YANG 定义序。
// 排序仅呈现层（SchemaForm），派生逻辑零改动（GD-01 黄金不受影响）。
const fields: Field[] = [
  { path: '/m/mac-aging', type: 'number', label: 'MAC老化时间' },
  { path: '/m/description', type: 'string', label: 'VLAN描述' },
  { path: '/m/type', type: 'enum', label: 'VLAN类型', options: [{ label: 'common', value: 'common' }] },
  { path: '/m/name', type: 'string', label: 'VLAN名称' },
  { path: '/m/id', type: 'number', label: 'VLAN标识', isKey: true },
]

function Harness() {
  const form = useConfigForm(fields, 'id')
  return (
    <UiProvider>
      <SchemaForm fields={fields} form={form} keyField="id" />
    </UiProvider>
  )
}

describe('SchemaForm · 字段呈现序（NCE 对齐）', () => {
  it('主键最前、name/description 次之、其余保持 YANG 定义序', () => {
    render(<Harness />)
    const labels = Array.from(document.querySelectorAll('.fis-label')).map((el) =>
      el.textContent?.replace(/[*：:]/g, '').trim(),
    )
    const idx = (s: string) => labels.findIndex((l) => l?.includes(s))
    expect(idx('VLAN标识')).toBeLessThan(idx('VLAN名称'))
    expect(idx('VLAN名称')).toBeLessThan(idx('VLAN描述'))
    expect(idx('VLAN描述')).toBeLessThan(idx('MAC老化时间'))
    // 非前置字段之间保持原始相对序（mac-aging 定义早于 type）。
    expect(idx('MAC老化时间')).toBeLessThan(idx('VLAN类型'))
  })
})
