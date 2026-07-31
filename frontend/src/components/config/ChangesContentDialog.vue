<template>
  <el-dialog
    :model-value="visible"
    :title="t('console.batch.changes')"
    width="72%"
    @update:model-value="emit('update:visible', $event)"
  >
    <div data-test="changes-legend" class="legend">
      <span class="legend-item added">{{ t('console.batch.legendAdd', { n: summary.creates }) }}</span>
      <span class="legend-item modified">{{ t('console.batch.legendModify', { n: summary.updates }) }}</span>
      <span class="legend-item removed">{{ t('console.batch.legendDelete', { n: summary.deletes }) }}</span>
    </div>
    <el-table
      v-if="rows.length"
      data-test="changes-table"
      :data="rows"
      row-key="id"
      default-expand-all
      :tree-props="{ children: 'children' }"
      size="small"
    >
      <el-table-column :label="t('console.batch.colAttr')" prop="attr" min-width="260" />
      <el-table-column :label="t('console.batch.colBefore')" min-width="200">
        <template #default="{ row }">
          <span :class="beforeClass(row)">{{ row.before }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="t('console.batch.colAfter')" min-width="200">
        <template #default="{ row }">
          <span :class="afterClass(row)">{{ row.after }}</span>
        </template>
      </el-table-column>
    </el-table>
    <el-empty v-else :description="t('console.batch.empty')" />
  </el-dialog>
</template>

<script setup lang="ts">
// 变更内容弹窗（FE-23）：纯前端渲染当前设备变更集——树形三列（属性/变更前/
// 变更后）+ 增/改/删图例计数（绿/黄/红），对齐 NCE 截图形态。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useChangesetStore, type ChangesetEntry } from '../../stores/changeset'

const props = defineProps<{ visible: boolean; device: string }>()
const emit = defineEmits<{ (e: 'update:visible', v: boolean): void }>()

const { t } = useI18n()
const changeset = useChangesetStore()

const summary = computed(() => changeset.summaryFor(props.device))

interface Row {
  id: string
  attr: string
  before?: unknown
  after?: unknown
  kind?: 'add' | 'modify' | 'remove'
  entryOp?: ChangesetEntry['op']
  children?: Row[]
}

const show = (v: unknown) => (v === undefined || v === null ? '' : String(v))

// 条目 → 树行：update 逐字段对比基线（等值跳过、cleared 叶为删除行）；
// create 全字段为新值；delete 展示基线旧值（无基线仅键定位）。
function fieldRows(e: ChangesetEntry, idx: number): Row[] {
  const base = (e.baseline ?? {}) as Record<string, unknown>
  const payload = (e.payload ?? {}) as Record<string, unknown>
  const rows: Row[] = []
  if (e.op === 'delete') {
    const keys = Object.keys(base)
    if (keys.length === 0) {
      rows.push({ id: `e${idx}-key`, attr: e.keyValue ?? '', before: e.keyValue, kind: 'remove' })
    }
    for (const k of keys) {
      rows.push({ id: `e${idx}-${k}`, attr: k, before: show(base[k]), kind: 'remove' })
    }
    return rows
  }
  for (const [k, v] of Object.entries(payload)) {
    const was = base[k]
    if (e.op === 'create' || was === undefined || was === null || was === '') {
      if (show(v) !== '') rows.push({ id: `e${idx}-${k}`, attr: k, after: show(v), kind: 'add' })
      continue
    }
    if (show(v) !== show(was)) {
      rows.push({ id: `e${idx}-${k}`, attr: k, before: show(was), after: show(v), kind: 'modify' })
    }
  }
  for (const k of e.cleared ?? []) {
    const was = base[k]
    if (was !== undefined && was !== null && show(was) !== '') {
      rows.push({ id: `e${idx}-clr-${k}`, attr: k, before: show(was), kind: 'remove' })
    }
  }
  return rows
}

const rows = computed<Row[]>(() =>
  changeset.entriesFor(props.device).map((e, i) => ({
    id: `e${i}`,
    attr: e.label ?? `${e.path}${e.keyValue ? ` [${e.keyValue}]` : ''}`,
    entryOp: e.op,
    children: fieldRows(e, i),
  })),
)

const beforeClass = (row: Row) => (row.kind === 'remove' ? 'cell-removed' : row.kind === 'modify' ? 'cell-modified' : '')
const afterClass = (row: Row) => (row.kind === 'add' ? 'cell-added' : row.kind === 'modify' ? 'cell-modified' : '')
</script>

<style scoped>
.legend {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-bottom: 8px;
  font-size: 12px;
}
.legend-item::before {
  content: '';
  display: inline-block;
  width: 10px;
  height: 10px;
  margin-right: 4px;
  vertical-align: -1px;
}
.legend-item.added::before {
  background: var(--el-color-success);
}
.legend-item.modified::before {
  background: var(--el-color-warning);
}
.legend-item.removed::before {
  background: var(--el-color-danger);
}
.cell-added {
  color: var(--el-color-success);
}
.cell-modified {
  color: var(--el-color-warning);
}
.cell-removed {
  color: var(--el-color-danger);
  text-decoration: line-through;
}
</style>
