import { defineStore } from 'pinia'
import { reactive } from 'vue'

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

const keyOf = (path: string, keyValue?: string) => `${path}::${keyValue ?? ''}`

/**
 * 攒批变更集 store（FE-23）：按设备隔离的待提交变更草稿。
 * 会话态、不持久化（刷新即丢，离开确认与提示条兜底——设计 D1 诚实边界）。
 */
export const useChangesetStore = defineStore('changeset', () => {
  /** deviceIp → 条目列表（保持入集顺序，提交按此序下发）。 */
  const byDevice = reactive<Record<string, ChangesetEntry[]>>({})

  const listOf = (device: string): ChangesetEntry[] => byDevice[device] ?? []

  const entriesFor = (device: string): ChangesetEntry[] => listOf(device)

  const countFor = (device: string): number => listOf(device).length

  const entryFor = (device: string, path: string, keyValue?: string): ChangesetEntry | undefined =>
    listOf(device).find((e) => keyOf(e.path, e.keyValue) === keyOf(path, keyValue))

  const isPendingDelete = (device: string, path: string, keyValue?: string): boolean =>
    entryFor(device, path, keyValue)?.op === 'delete'

  /**
   * 写入/合并 create|update 条目：同 (path,keyValue) 一份变更项——payload/cleared/
   * label 取最新，op 与 baseline 保持首次（FE-21 合并语义：diff 基线锚定首次编辑前）。
   */
  function upsert(device: string, entry: ChangesetEntry) {
    const list = byDevice[device] ?? (byDevice[device] = [])
    const idx = list.findIndex((e) => keyOf(e.path, e.keyValue) === keyOf(entry.path, entry.keyValue))
    if (idx < 0) {
      list.push({ ...entry })
      return
    }
    const prev = list[idx]
    list[idx] = {
      ...entry,
      op: prev.op === 'create' ? 'create' : entry.op,
      baseline: prev.baseline ?? entry.baseline ?? null,
    }
  }

  /**
   * 标记删除（FE-16）：待创建条目直接移除（不产生删除报文）；既有 update 条目
   * 被删除取代（一键一份变更项）；否则新增删除项。
   */
  function markDelete(device: string, mark: DeleteMark) {
    const list = byDevice[device] ?? (byDevice[device] = [])
    const idx = list.findIndex((e) => keyOf(e.path, e.keyValue) === keyOf(mark.path, mark.keyValue))
    if (idx >= 0 && list[idx].op === 'create') {
      list.splice(idx, 1)
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
    if (idx >= 0) list[idx] = entry
    else list.push(entry)
  }

  /** 取消删除（FE-16「取消删除」按钮）：移除该删除项。 */
  function unmarkDelete(device: string, path: string, keyValue?: string) {
    const list = byDevice[device]
    if (!list) return
    const idx = list.findIndex(
      (e) => e.op === 'delete' && keyOf(e.path, e.keyValue) === keyOf(path, keyValue),
    )
    if (idx >= 0) list.splice(idx, 1)
  }

  /** 移除任意条目（撤销单条变更）。 */
  function remove(device: string, path: string, keyValue?: string) {
    const list = byDevice[device]
    if (!list) return
    const idx = list.findIndex((e) => keyOf(e.path, e.keyValue) === keyOf(path, keyValue))
    if (idx >= 0) list.splice(idx, 1)
  }

  /** 清空指定设备的变更集（FE-23「重置」；仅当前设备，其他设备保留）。 */
  function clear(device: string) {
    delete byDevice[device]
  }

  /** 图例计数：增/改/删（FE-23）。 */
  const summaryFor = (device: string): ChangesetSummary => {
    const out: ChangesetSummary = { creates: 0, updates: 0, deletes: 0 }
    for (const e of listOf(device)) {
      if (e.op === 'create') out.creates++
      else if (e.op === 'update') out.updates++
      else out.deletes++
    }
    return out
  }

  /**
   * 序列化为后端 ChangesetReq（config-changeset 契约）：list 条目 payload 按
   * listKey 包裹为 {listKey:[payload]}；delete 条目携 key；空 cleared 不入。
   */
  const toRequest = (device: string): ChangesetRequest => ({
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
  })

  return {
    byDevice,
    entriesFor,
    countFor,
    entryFor,
    isPendingDelete,
    upsert,
    markDelete,
    unmarkDelete,
    remove,
    clear,
    summaryFor,
    toRequest,
  }
})
