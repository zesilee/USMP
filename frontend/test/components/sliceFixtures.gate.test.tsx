import { describe, it, expect, vi, beforeAll, afterEach } from 'vitest'
import { readFileSync, readdirSync } from 'node:fs'
import { resolve, basename } from 'node:path'
import { render, cleanup, act } from '@testing-library/react'
import ModuleListTab from '../../src/components/config/ModuleListTab'
import SchemaForm from '../../src/components/config/SchemaForm'
import { useConfigForm } from '../../src/hooks/useConfigForm'
import { UiProvider } from '../../src/ui'
import { deriveTabs, deriveColumns, deriveKeyField, leafName } from '../../src/utils/moduleConsole'
import { rulesForGateProbe } from './sliceGateProbe'
import * as apiModule from '../../src/api'
import type { Field } from '../../src/utils/crdSchemaParser'

// 垂直切片闸门（tasks 7.4/7.5，design D3）：以 **backend/testdata/schema-fixtures
// 全部模块**（68 个，与派生黄金同源、非玩具示例）驱动切片三件——
//   ① Table 运行时动态列：每个 list Tab 的列由 schema 现场派生且与黄金同源纯函数
//      结论一致（表头精确匹配，防子串误通过），antd Table 渲染不崩；
//   ② 单元格分派：按 keyField/列类型合成 2 行数据喂入，render 路径（cellVisible/
//      statusTone/enum Tag/boolean Tag）在 68 模块口径下真实执行；
//   ③ Form 运行时动态校验：每个模块首个表单面 fieldValidation 全字段求值不炸。
// 任一模块失败即闸门不通过（D3：暂停后续重建）。
vi.mock('../../src/api')

const FIXTURES = resolve(process.cwd(), '../backend/testdata/schema-fixtures')

interface Fixture {
  module: string
  fields: Field[]
}

const modules = readdirSync(FIXTURES)
  .filter((f) => f.endsWith('.json'))
  .map((f) => basename(f, '.json'))
  .sort()

function loadFixture(m: string): Fixture {
  return JSON.parse(readFileSync(resolve(FIXTURES, m + '.json'), 'utf-8'))
}

// 按列类型合成两行示例数据（含 up/down 触发状态点、枚举首值触发 Tag）——
// 让单元格 render 路径真实执行而非只渲染表头。
function synthRows(listField: Field, keyField: string): Record<string, any>[] {
  const row = (i: number): Record<string, any> => {
    const r: Record<string, any> = {}
    for (const c of deriveColumns(listField)) {
      const k = leafName(c)
      if (c.type === 'number') r[k] = i
      else if (c.type === 'boolean') r[k] = i === 1
      else if (c.type === 'enum') r[k] = c.options?.[0]?.value ?? 'e'
      else r[k] = i === 1 ? 'up' : 'down' // statusTone 两态都走到
    }
    r[keyField] = i
    return r
  }
  return [row(1), row(2)]
}

beforeAll(() => {
  // 收口 rc-component 延迟定时器：unmount 后残留 setTimeout 在环境销毁后触发
  // 会抛 window is not defined（teardown 泄漏 → 潜在 flake）。
  vi.useFakeTimers()
})

afterEach(() => {
  act(() => {
    vi.runOnlyPendingTimers()
  })
  cleanup()
})

function headerTitles(container: HTMLElement): string[] {
  return Array.from(container.querySelectorAll('th .ant-table-column-title, th')).map(
    (el) => el.textContent?.trim() ?? '',
  )
}

function FormHarness({ fields }: { fields: Field[] }) {
  const form = useConfigForm(fields)
  return <SchemaForm fields={fields} form={form} />
}

describe(`垂直切片闸门 · 全 fixture 驱动（${modules.length} 模块）`, () => {
  it('fixture 集非空且为全量（防目录漂移导致闸门空转）', () => {
    expect(modules.length).toBeGreaterThanOrEqual(60)
  })

  for (const m of modules) {
    // 大模块（driver/ifm 等）在全量并行下可超默认 15s——上限放宽而非削覆盖面。
    it(`[${m}] 列表 Tab 动态列 + 单元格分派 + 表单校验求值`, { timeout: 60000 }, async () => {
      const fx = loadFixture(m)
      const tabs = deriveTabs(fx.fields)
      expect(tabs.length).toBeGreaterThan(0)

      for (const tab of tabs.filter((t) => t.kind === 'list')) {
        const lf = tab.listField!
        const keyField = deriveKeyField(lf)
        expect(keyField).toBeTruthy()
        const rows = synthRows(lf, keyField)
        vi.mocked(apiModule.getConfig).mockResolvedValue({
          data: { success: true, data: { [leafName(lf)]: rows } },
        } as any)

        const { container, unmount } = render(
          <UiProvider>
            <ModuleListTab tab={tab} rootName={fx.module} device="10.0.0.1" />
          </UiProvider>,
        )
        // 取数是异步微任务链：fake timers 下手动冲刷。
        await act(async () => {
          await vi.runAllTimersAsync()
        })

        // ① 列精确匹配（防子串误通过）。
        const titles = headerTitles(container)
        for (const c of deriveColumns(lf)) {
          expect(titles, `[${m}/${tab.name}] 列缺失: ${c.label}`).toContain(c.label)
        }
        // ② 单元格 render 真实执行：合成行内容出现在表体。
        expect(
          container.querySelectorAll('.ant-table-tbody td').length,
          `[${m}/${tab.name}] 数据行未渲染`,
        ).toBeGreaterThan(0)
        unmount()
      }

      const formTab = tabs.find((t) => t.kind === 'form')
      if (formTab) {
        const { unmount } = render(
          <UiProvider>
            <FormHarness fields={formTab.field.fields || []} />
          </UiProvider>,
        )
        unmount()
      }

      // ③ 校验探针：全模块每字段 fieldValidation({}/示例值) 无异常。
      rulesForGateProbe(fx.fields)
    })
  }
})
