import { describe, it, expect, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useChangesetStore } from '../../src/stores/changeset'

const DEV_A = '10.0.0.1'
const DEV_B = '10.0.0.2'
const VLAN_PATH = '/vlan:vlan/vlan:vlans'

function upsertVlan(
  s: ReturnType<typeof useChangesetStore>,
  device: string,
  op: 'create' | 'update',
  keyValue: string,
  payload: Record<string, unknown>,
  extra: Partial<{ cleared: string[]; baseline: Record<string, unknown> | null; label: string }> = {},
) {
  s.upsert(device, {
    op,
    path: VLAN_PATH,
    listKey: 'vlan',
    keyValue,
    payload,
    cleared: extra.cleared ?? [],
    baseline: extra.baseline ?? null,
    label: extra.label ?? `vlan ${keyValue}`,
  })
}

describe('changeset store · 攒批变更集（FE-23/FE-21/FE-16）', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('初始空态：计数 0、条目空', () => {
    const s = useChangesetStore()
    expect(s.countFor(DEV_A)).toBe(0)
    expect(s.entriesFor(DEV_A)).toEqual([])
  })

  it('upsert 新增条目：计数与条目可查', () => {
    const s = useChangesetStore()
    upsertVlan(s, DEV_A, 'update', '10', { id: 10, description: 'x' })
    expect(s.countFor(DEV_A)).toBe(1)
    expect(s.entryFor(DEV_A, VLAN_PATH, '10')?.payload).toEqual({ id: 10, description: 'x' })
  })

  it('同条目二次 upsert 合并为一份：payload/cleared 取最新，baseline 保持首次快照（FE-21）', () => {
    const s = useChangesetStore()
    upsertVlan(s, DEV_A, 'update', '10', { id: 10, description: 'v1' }, { baseline: { id: 10, description: 'orig' } })
    upsertVlan(s, DEV_A, 'update', '10', { id: 10, description: 'v2' }, { cleared: ['name'], baseline: { id: 10, description: 'v1' } })
    expect(s.countFor(DEV_A)).toBe(1)
    const e = s.entryFor(DEV_A, VLAN_PATH, '10')!
    expect(e.payload).toEqual({ id: 10, description: 'v2' })
    expect(e.cleared).toEqual(['name'])
    expect(e.baseline).toEqual({ id: 10, description: 'orig' }, )
  })

  it('编辑待创建条目仍是 create：op 保持首次（否则提交把新建误报为修改）', () => {
    const s = useChangesetStore()
    upsertVlan(s, DEV_A, 'create', '50', { id: 50, name: 'v1' })
    upsertVlan(s, DEV_A, 'update', '50', { id: 50, name: 'v2' })
    const e = s.entryFor(DEV_A, VLAN_PATH, '50')!
    expect(e.op).toBe('create')
    expect(e.payload).toEqual({ id: 50, name: 'v2' })
  })

  it('markDelete 既有条目：生成删除项并可查待删除态（FE-16）', () => {
    const s = useChangesetStore()
    s.markDelete(DEV_A, { path: VLAN_PATH, listKey: 'vlan', keyValue: '30', label: 'vlan 30' })
    expect(s.countFor(DEV_A)).toBe(1)
    expect(s.isPendingDelete(DEV_A, VLAN_PATH, '30')).toBe(true)
  })

  it('markDelete 覆盖同键 update 条目：删除取代修改（一键一份变更项）', () => {
    const s = useChangesetStore()
    upsertVlan(s, DEV_A, 'update', '10', { id: 10, description: 'x' })
    s.markDelete(DEV_A, { path: VLAN_PATH, listKey: 'vlan', keyValue: '10', label: 'vlan 10' })
    expect(s.countFor(DEV_A)).toBe(1)
    expect(s.entryFor(DEV_A, VLAN_PATH, '10')?.op).toBe('delete')
  })

  it('markDelete 待创建条目 = 直接移除，不产生删除项（FE-16 边界）', () => {
    const s = useChangesetStore()
    upsertVlan(s, DEV_A, 'create', '40', { id: 40, name: 'new' })
    s.markDelete(DEV_A, { path: VLAN_PATH, listKey: 'vlan', keyValue: '40', label: 'vlan 40' })
    expect(s.countFor(DEV_A)).toBe(0)
    expect(s.entryFor(DEV_A, VLAN_PATH, '40')).toBeUndefined()
  })

  it('unmarkDelete 取消删除：删除项移除、计数减一', () => {
    const s = useChangesetStore()
    s.markDelete(DEV_A, { path: VLAN_PATH, listKey: 'vlan', keyValue: '30', label: 'vlan 30' })
    s.unmarkDelete(DEV_A, VLAN_PATH, '30')
    expect(s.countFor(DEV_A)).toBe(0)
    expect(s.isPendingDelete(DEV_A, VLAN_PATH, '30')).toBe(false)
  })

  it('按设备隔离：A 攒 2 条不影响 B；clear 只清当前设备（FE-23）', () => {
    const s = useChangesetStore()
    upsertVlan(s, DEV_A, 'update', '10', { id: 10 })
    upsertVlan(s, DEV_A, 'create', '20', { id: 20 })
    upsertVlan(s, DEV_B, 'update', '10', { id: 10 })
    expect(s.countFor(DEV_A)).toBe(2)
    expect(s.countFor(DEV_B)).toBe(1)
    s.clear(DEV_A)
    expect(s.countFor(DEV_A)).toBe(0)
    expect(s.countFor(DEV_B)).toBe(1)
  })

  it('不同 path 同 keyValue 互不合并（定位=path+keyValue）', () => {
    const s = useChangesetStore()
    upsertVlan(s, DEV_A, 'update', '10', { id: 10 })
    s.upsert(DEV_A, {
      op: 'update',
      path: '/ifm:ifm/ifm:interfaces',
      listKey: 'interface',
      keyValue: '10',
      payload: { name: '10' },
      cleared: [],
      baseline: null,
      label: 'if 10',
    })
    expect(s.countFor(DEV_A)).toBe(2)
  })

  it('toRequest 序列化为后端契约（op/path/payload/key/cleared，listKey 包裹）', () => {
    const s = useChangesetStore()
    upsertVlan(s, DEV_A, 'update', '10', { id: 10, description: 'x' }, { cleared: ['name'] })
    s.markDelete(DEV_A, { path: VLAN_PATH, listKey: 'vlan', keyValue: '30', label: 'vlan 30' })
    const req = s.toRequest(DEV_A)
    expect(req.device).toBe(DEV_A)
    expect(req.entries).toEqual([
      {
        op: 'update',
        path: VLAN_PATH,
        payload: { vlan: [{ id: 10, description: 'x' }] },
        cleared: ['name'],
      },
      { op: 'delete', path: VLAN_PATH, key: '30' },
    ])
  })

  it('summaryFor 图例计数：增/改/删（FE-23 徽标与图例数据源）', () => {
    const s = useChangesetStore()
    upsertVlan(s, DEV_A, 'create', '1', { id: 1 })
    upsertVlan(s, DEV_A, 'update', '2', { id: 2 })
    s.markDelete(DEV_A, { path: VLAN_PATH, listKey: 'vlan', keyValue: '3', label: 'vlan 3' })
    expect(s.summaryFor(DEV_A)).toEqual({ creates: 1, updates: 1, deletes: 1 })
  })

  it('非 list 表单条目（无 keyValue）：payload 原样序列化、同 path 合并', () => {
    const s = useChangesetStore()
    const entry = {
      op: 'update' as const,
      path: '/bgp:bgp',
      payload: { 'base-process': { as: '100' } },
      cleared: [],
      baseline: null,
      label: 'bgp',
    }
    s.upsert(DEV_A, entry)
    s.upsert(DEV_A, { ...entry, payload: { 'base-process': { as: '200' } } })
    expect(s.countFor(DEV_A)).toBe(1)
    const req = s.toRequest(DEV_A)
    expect(req.entries[0]).toEqual({
      op: 'update',
      path: '/bgp:bgp',
      payload: { 'base-process': { as: '200' } },
    })
  })
})
