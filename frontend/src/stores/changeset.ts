import { create } from './createStore'

/** 变更集条目操作类型（与后端 config-changeset 契约同名）。 */
export type ChangesetOp = 'create' | 'update' | 'delete'

/**
 * 变更集单条目（FE-23/FE-21/FE-16）：
 * 定位 = path + keyValue（非 list 表单条目无 keyValue，按 path 定位）；
 * baseline 是首次编辑前的设备实际态快照（diff 展示与回滚参照，合并更新时保持首次值）。
 */
export interface ChangesetEntry {
  op: ChangesetOp
  path: string
  /** list 包裹键（RFC7951 列表键名，如 'vlan'/'interface'）；非 list 表单为空。 */
  listKey?: string
  /** list 条目主键值；非 list 表单为空。 */
  keyValue?: string
  /** 以 path 为根的 RFC7951 子树载荷（list 条目为单条目对象，序列化时包裹）。 */
  payload?: Record<string, unknown>
  /** 字段级清除的叶名（提交时经叶级删除报文生效，FE-22/CS-05）。 */
  cleared?: string[]
  /** 首次编辑前基线快照（同条目多次编辑合并时保持首次）。 */
  baseline?: Record<string, unknown> | null
  /** 面包屑/变更内容展示标签。 */
  label?: string
}

/** markDelete 入参：删除条目的定位与展示标签。 */
export interface DeleteMark {
  path: string
  listKey?: string
  keyValue: string
  label?: string
}

/** 后端 ChangesetReq 序列化形态（api.previewChangeset/commitChangeset 入参）。 */
export interface ChangesetRequest {
  device: string
  entries: Array<{
    op: ChangesetOp
    path: string
    payload?: Record<string, unknown>
    key?: string
    cleared?: string[]
  }>
}

/** 图例计数（FE-23 徽标与变更内容图例数据源）。 */
export interface ChangesetSummary {
  creates: number
  updates: number
  deletes: number
}

const entryKey = (path: string, keyValue?: string) => `${path}::${keyValue ?? ''}`

interface ChangesetState {
  /** deviceIp → 条目列表（保持入集顺序，提交按此序下发）。不可变更新（D6）。 */
  byDevice: Record<string, ChangesetEntry[]>
  entriesFor: (device: string) => ChangesetEntry[]
  countFor: (device: string) => number
  entryFor: (device: string, path: string, keyValue?: string) => ChangesetEntry | undefined
  isPendingDelete: (device: string, path: string, keyValue?: string) => boolean
  upsert: (device: string, entry: ChangesetEntry) => void
  markDelete: (device: string, mark: DeleteMark) => void
  unmarkDelete: (device: string, path: string, keyValue?: string) => void
  remove: (device: string, path: string, keyValue?: string) => void
  clear: (device: string) => void
  summaryFor: (device: string) => ChangesetSummary
  toRequest: (device: string) => ChangesetRequest
}

/**
 * 攒批变更集 store（FE-23）：按设备隔离的待提交变更草稿。
 * 会话态、不持久化（刷新即丢，离开确认与提示条兜底——设计 D1 诚实边界）。
 */
export const useChangesetStore = create<ChangesetState>((set, get) => {
  const listOf = (device: string): ChangesetEntry[] => get().byDevice[device] ?? []

  // 不可变写回（D6）：整设备列表替换；清空设备用解构删键（键不存在=无草稿）。
  const setList = (device: string, list: ChangesetEntry[]) =>
    set((s) => ({ byDevice: { ...s.byDevice, [device]: list } }))

  return {
    byDevice: {},

    entriesFor: (device) => listOf(device),
    countFor: (device) => listOf(device).length,

    entryFor: (device, path, keyValue) =>
      listOf(device).find((e) => entryKey(e.path, e.keyValue) === entryKey(path, keyValue)),

    isPendingDelete: (device, path, keyValue) =>
      get().entryFor(device, path, keyValue)?.op === 'delete',

    /**
     * 写入/合并 create|update 条目：同 (path,keyValue) 一份变更项——payload/cleared/
     * label 取最新，op 与 baseline 保持首次（FE-21 合并语义：diff 基线锚定首次编辑前）。
     */
    upsert: (device, entry) => {
      const list = listOf(device)
      const idx = list.findIndex(
        (e) => entryKey(e.path, e.keyValue) === entryKey(entry.path, entry.keyValue),
      )
      if (idx < 0) {
        setList(device, [...list, { ...entry }])
        return
      }
      const prev = list[idx]
      const merged: ChangesetEntry = {
        ...entry,
        op: prev.op === 'create' ? 'create' : entry.op,
        baseline: prev.baseline ?? entry.baseline ?? null,
      }
      setList(device, list.map((e, i) => (i === idx ? merged : e)))
    },

    /**
     * 标记删除（FE-16）：待创建条目直接移除（不产生删除报文）；既有 update 条目
     * 被删除取代（一键一份变更项）；否则新增删除项。
     */
    markDelete: (device, mark) => {
      const list = listOf(device)
      const idx = list.findIndex(
        (e) => entryKey(e.path, e.keyValue) === entryKey(mark.path, mark.keyValue),
      )
      if (idx >= 0 && list[idx].op === 'create') {
        setList(device, list.filter((_, i) => i !== idx))
        return
      }
      const entry: ChangesetEntry = {
        op: 'delete',
        path: mark.path,
        listKey: mark.listKey,
        keyValue: mark.keyValue,
        label: mark.label,
        baseline: idx >= 0 ? list[idx].baseline : null,
      }
      setList(device, idx >= 0 ? list.map((e, i) => (i === idx ? entry : e)) : [...list, entry])
    },

    /** 取消删除（FE-16「取消删除」按钮）：移除该删除项。 */
    unmarkDelete: (device, path, keyValue) => {
      const list = get().byDevice[device]
      if (!list) return
      setList(
        device,
        list.filter((e) => !(e.op === 'delete' && entryKey(e.path, e.keyValue) === entryKey(path, keyValue))),
      )
    },

    /** 移除任意条目（撤销单条变更）。 */
    remove: (device, path, keyValue) => {
      const list = get().byDevice[device]
      if (!list) return
      setList(device, list.filter((e) => entryKey(e.path, e.keyValue) !== entryKey(path, keyValue)))
    },

    /** 清空指定设备的变更集（FE-23「重置」；仅当前设备，其他设备保留）。 */
    clear: (device) =>
      set((s) => {
        const { [device]: _drop, ...rest } = s.byDevice
        return { byDevice: rest }
      }),

    /** 图例计数：增/改/删（FE-23）。 */
    summaryFor: (device) => {
      const out: ChangesetSummary = { creates: 0, updates: 0, deletes: 0 }
      for (const e of listOf(device)) {
        if (e.op === 'create') out.creates++
        else if (e.op === 'update') out.updates++
        else out.deletes++
      }
      return out
    },

    /**
     * 序列化为后端 ChangesetReq（config-changeset 契约）：list 条目 payload 按
     * listKey 包裹为 {listKey:[payload]}；delete 条目携 key；空 cleared 不入。
     */
    toRequest: (device) => ({
      device,
      entries: listOf(device).map((e) => {
        if (e.op === 'delete') {
          return { op: e.op, path: e.path, key: e.keyValue }
        }
        const payload = e.listKey ? { [e.listKey]: [e.payload ?? {}] } : (e.payload ?? {})
        const out: ChangesetRequest['entries'][number] = { op: e.op, path: e.path, payload }
        if (e.cleared && e.cleared.length > 0) out.cleared = [...e.cleared]
        return out
      }),
    }),
  }
})
