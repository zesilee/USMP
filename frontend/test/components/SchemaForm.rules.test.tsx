import { describe, it, expect } from 'vitest'
import { useEffect } from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import SchemaForm from '../../src/components/config/SchemaForm'
import { useConfigForm } from '../../src/hooks/useConfigForm'
import { UiProvider } from '../../src/ui'
import { fieldValidation, rangeViolations } from '../../src/form/antdRules'
import type { Field } from '../../src/utils/crdSchemaParser'

// SchemaForm + antdRules F2（FE-02/§9，R05 闸门第二件）：校验规则由 YANG 元数据
// **运行时**生成——pattern/range/must 行内即时红、required 计入提交门禁（权威
// 判定在 configForm.isBlocked，行内与门禁双防线）、when 隐藏字段不渲染。
const fields: Field[] = [
  { path: '/m/id', type: 'number', label: 'id', isKey: true, minimum: 1, maximum: 4094 },
  { path: '/m/name', type: 'string', label: 'name', pattern: '[A-Za-z0-9_-]+' },
  { path: '/m/desc', type: 'string', label: 'desc', required: true },
  { path: '/m/vid', type: 'number', label: 'vid', dynamicDefault: true, required: true },
  {
    path: '/m/reuse', type: 'number', label: 'reuse',
    must: [{ expr: '(../suppress>../reuse)', message: 'reuse must < suppress' }],
  },
  { path: '/m/suppress', type: 'number', label: 'suppress' },
  { path: '/m/sub', type: 'string', label: 'sub', when: "../mode='sub'" },
  { path: '/m/mode', type: 'string', label: 'mode' },
]

function Harness({ seed }: { seed?: Record<string, any> }) {
  const form = useConfigForm(fields, 'id')
  const { resetForm } = form
  useEffect(() => {
    if (seed) resetForm(seed)
    // 一次性种子（生产路径同为 resetForm，详情区随行/建切换调用）
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])
  return (
    <UiProvider>
      <SchemaForm fields={fields} form={form} keyField="id" />
      <output data-test="blocked">{String(form.blocked)}</output>
    </UiProvider>
  )
}

describe('SchemaForm · 运行时动态校验（R05 闸门）', () => {
  it('pattern 违例：输入非法值行内即时红，改合法即消（FE-02 场景）', async () => {
    render(<Harness />)
    const name = screen.getAllByRole('textbox')[0]
    fireEvent.change(name, { target: { value: 'bad name!' } })
    expect(await screen.findByText(/name/, { selector: '.ant-form-item-explain-error' })).toBeInTheDocument()
    fireEvent.change(name, { target: { value: 'ok_name' } })
    // 错误提示走 antd 离场动画，等待其真正卸载（非 -leave 态残影）。
    await waitFor(() =>
      expect(
        document.querySelector('.ant-form-item-explain-error:not([class*="-leave"])'),
      ).toBeNull(),
    )
  })

  it('must 跨字段违例行内展示（引擎对整表单求值）', async () => {
    render(<Harness seed={{ suppress: 100, reuse: 200 }} />)
    expect(await screen.findByText('reuse must < suppress')).toBeInTheDocument()
  })

  it('when=false 字段不渲染（引擎驱动显隐）', () => {
    render(<Harness seed={{ mode: 'main' }} />)
    expect(screen.queryByText('sub', { selector: 'label *' })).toBeNull()
  })

  it('required 缺失计入提交门禁 blocked；必填标记渲染、dynamicDefault 豁免', () => {
    render(<Harness seed={{ id: 1 }} />)
    expect(screen.getByTestId?.bind(screen) ? document.querySelector('[data-test="blocked"]')!.textContent : '').toBe('true')
    // desc 必填有标记；vid dynamicDefault 豁免无标记
    const requiredLabels = Array.from(document.querySelectorAll('.ant-form-item-required')).map(
      (el) => el.textContent,
    )
    expect(requiredLabels.join()).toContain('desc')
    expect(requiredLabels.join()).not.toContain('vid')
  })
})

describe('SchemaForm · choice 接线（choiceScope/onChoiceUpdate）', () => {
  const cf: Field[] = [
    {
      path: '/m/mode', type: 'choice', label: 'mode',
      cases: [
        { name: 'a', label: 'A', fields: [{ path: '/m/speed', type: 'number', label: 'speed' }] },
        { name: 'b', label: 'B', fields: [{ path: '/m/auto', type: 'boolean', label: 'auto' }] },
      ],
    },
  ]
  function ChoiceHarness() {
    const form = useConfigForm(cf)
    return (
      <UiProvider>
        <SchemaForm fields={cf} form={form} />
        <output data-test="scope">{JSON.stringify(form.formData)}</output>
      </UiProvider>
    )
  }

  it('choice 成员编辑经 onChoiceUpdate 落入 formData；切分支清空旧成员', async () => {
    render(<ChoiceHarness />)
    fireEvent.change(screen.getByRole('spinbutton'), { target: { value: '100' } })
    fireEvent.blur(screen.getByRole('spinbutton'))
    await waitFor(() =>
      expect(JSON.parse(document.querySelector('[data-test="scope"]')!.textContent!)).toEqual({ speed: 100 }),
    )
    fireEvent.click(screen.getByRole('radio', { name: 'B' }))
    await waitFor(() => {
      const scope = JSON.parse(document.querySelector('[data-test="scope"]')!.textContent!)
      expect('speed' in scope).toBe(false)
    })
  })
})

describe('antdRules 纯函数面', () => {
  it('range 违例判定（number 越界/NaN）', () => {
    expect(rangeViolations(fields, { id: 5000 }).map((f) => f.label)).toContain('id')
    expect(rangeViolations(fields, { id: 'abc' }).map((f) => f.label)).toContain('id')
    expect(rangeViolations(fields, { id: 100 })).toEqual([])
  })

  it('fieldValidation：readonly 恒空、无值 pattern 不判、must 命中带消息', () => {
    const ro: Field = { path: '/m/x', type: 'string', label: 'x', readonly: true, pattern: 'a+' }
    expect(fieldValidation(ro, [ro], { x: 'zzz' })).toEqual({})
    expect(fieldValidation(fields[1], fields, {})).toEqual({})
    const v = fieldValidation(fields[4], fields, { suppress: 1, reuse: 2 })
    expect(v.status).toBe('error')
    expect(v.help).toBe('reuse must < suppress')
  })
})
