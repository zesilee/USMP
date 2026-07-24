import { describe, it, expect } from 'vitest'
import { readFileSync, writeFileSync, readdirSync, existsSync, mkdirSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, resolve, basename } from 'node:path'
import { conclusionsFor, serialize, type Conclusions } from './deriveConclusions'
import type { Field } from '../../src/utils/crdSchemaParser'

// 前端控制台派生黄金（F1，GD-01/02/03）。
//
// 对 backend/testdata/schema-fixtures/ 的**全部**模块 fixture 运行既有派生纯函数
// （deriveTabs/deriveKeyField/deriveColumns/filterableFields/deriveSchemaTree），把
// 「模块 → 控制台形态」钉为黄金。任一模块派生结论变化即失败并定位到模块。
//
// 模块集合由 fixture 目录动态发现（GD-01），不硬编码名单——新增 fixture 自动纳入
// 覆盖（缺黄金即失败，直到为其生成）。
//
// 更新黄金（仅派生逻辑变更的**预期**刷新时）：UPDATE_GOLDEN=1 npx vitest run test/golden。
// 刻意不用 vitest snapshot 的 -u：结构化 JSON 落盘、一模块一文件，diff 可审、避免顺手全刷（D6/D7）。

const HERE = dirname(fileURLToPath(import.meta.url))
const FIXTURES_DIR = resolve(HERE, '../../../backend/testdata/schema-fixtures')
const GOLDEN_DIR = resolve(HERE, '__data__')
const UPDATE = process.env.UPDATE_GOLDEN === '1'

interface Fixture {
  module: string
  fields: Field[]
}

function loadFixture(module: string): Fixture {
  const raw = readFileSync(resolve(FIXTURES_DIR, module + '.json'), 'utf-8')
  return JSON.parse(raw) as Fixture
}

function fixtureModules(): string[] {
  return readdirSync(FIXTURES_DIR)
    .filter((f) => f.endsWith('.json'))
    .map((f) => basename(f, '.json'))
    .sort()
}

function goldenPath(module: string): string {
  return resolve(GOLDEN_DIR, module + '.json')
}

const modules = fixtureModules()

describe('console-derivation golden (GD-01)', () => {
  it('fixture 目录非空（否则黄金覆盖形同虚设）', () => {
    expect(modules.length).toBeGreaterThan(0)
  })

  // GD-01：全部 fixture 模块参与派生比对；新增模块缺黄金即失败。
  it.each(modules)('%s: 派生结论匹配黄金', (module) => {
    const fx = loadFixture(module)
    const got = serialize(conclusionsFor(module, fx.fields))

    if (UPDATE) {
      if (!existsSync(GOLDEN_DIR)) mkdirSync(GOLDEN_DIR, { recursive: true })
      writeFileSync(goldenPath(module), got)
      return
    }

    const path = goldenPath(module)
    expect(existsSync(path), `缺少黄金 ${module}.json — 运行 UPDATE_GOLDEN=1 生成后人工审阅`).toBe(true)
    const want = readFileSync(path, 'utf-8')
    expect(got, `模块 ${module} 派生结论与黄金不符`).toBe(want)
  })
})

// GD-02：黄金只含派生结论，不含 schema 原文副本、不含 i18n 本地化标签。
// 用反向断言证明这条边界：schema 的非派生相关变化、i18n 变化都不得震动结论，
// 派生逻辑相关变化必须震动结论。
describe('console-derivation golden 边界 (GD-02)', () => {
  const sample = 'vlan'

  it('注入不影响任何派生的 schema 字段（description/placeholder），结论不变', () => {
    const fx = loadFixture(sample)
    const base = serialize(conclusionsFor(sample, fx.fields))

    // 深拷贝后给每个字段塞入纯呈现性、不参与任何派生的属性。
    const mutated = JSON.parse(JSON.stringify(fx.fields)) as Field[]
    const dust = (fs: Field[]) => {
      for (const f of fs) {
        ;(f as any).description = 'INJECTED_' + f.path
        ;(f as any).placeholder = 'INJECTED_PLACEHOLDER'
        if (f.fields) dust(f.fields)
      }
    }
    dust(mutated)
    const after = serialize(conclusionsFor(sample, mutated))
    expect(after).toBe(base)
  })

  it('本地化标签变化不进结论（结论内无中文标签，只有 raw path/name）', () => {
    const fx = loadFixture(sample)
    const c = conclusionsFor(sample, fx.fields)
    const text = serialize(c)
    // 结论文本内不得出现本地化标签的痕迹：tab/列/树节点标识全为 raw YANG 名。
    // 抽样断言 tab.name 与列 name 均为 ASCII 标识符形态（不含中文）。
    for (const t of c.tabs) {
      if (t.name === '__basic__') continue
      expect(t.name, `tab name 应为 raw YANG 名: ${t.name}`).toMatch(/^[\x00-\x7F]+$/)
    }
    expect(text).not.toMatch(/[一-鿿]/) // 结论文本零中文字符
  })

  it('派生相关变化（把首个 list 主键叶改名）必须震动结论', () => {
    const fx = loadFixture(sample)
    const base = serialize(conclusionsFor(sample, fx.fields))

    const mutated = JSON.parse(JSON.stringify(fx.fields)) as Field[]
    // 找到首个 list 的首个 isKey 叶，改其 path 末段——deriveKeyField/columns/tree 都应变。
    let touched = false
    const walk = (fs: Field[]) => {
      for (const f of fs) {
        if (f.type === 'list' && !touched) {
          for (const c of f.fields || []) {
            if (c.isKey) {
              c.path = c.path.replace(/[^/]+$/, 'renamed_key')
              touched = true
              break
            }
          }
        }
        if (f.fields) walk(f.fields)
      }
    }
    walk(mutated)
    expect(touched, 'vlan fixture 应含至少一个 isKey 叶用于本断言').toBe(true)
    const after = serialize(conclusionsFor(sample, mutated))
    expect(after).not.toBe(base)
  })
})

// GD-03：派生变化可定位到模块——局部变更只影响相关模块的结论，未受影响模块逐字节不变。
describe('console-derivation golden 定位性 (GD-03)', () => {
  it('改一个模块的 fields 不影响另一模块的结论（模块间隔离）', () => {
    const a = loadFixture('vlan')
    const bBefore = serialize(conclusionsFor('ntp', loadFixture('ntp').fields))

    // 篡改 vlan 的 fields 副本，重算 vlan 结论——不得改变 ntp 的结论。
    const mutated = JSON.parse(JSON.stringify(a.fields)) as Field[]
    if (mutated[0]) (mutated[0] as any).readonly = !mutated[0].readonly
    conclusionsFor('vlan', mutated)

    const bAfter = serialize(conclusionsFor('ntp', loadFixture('ntp').fields))
    expect(bAfter).toBe(bBefore)
  })

  it('每模块一份黄金文件（GD-03 分文件存储）', () => {
    if (UPDATE) return // 生成轮次尚未落盘，跳过存在性校验
    for (const m of modules) {
      expect(existsSync(goldenPath(m)), `模块 ${m} 应有独立黄金文件`).toBe(true)
    }
  })
})
