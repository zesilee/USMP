<template>
  <el-dialog
    :model-value="visible"
    :title="t('console.batch.dryRunTitle')"
    width="76%"
    @update:model-value="emit('update:visible', $event)"
  >
    <div v-if="loading" v-loading="true" class="dryrun-loading" />
    <el-alert v-else-if="error" data-test="dryrun-error" type="error" :title="error" :closable="false" show-icon />
    <el-tabs v-else-if="result">
      <el-tab-pane :label="t('console.batch.tabPayload')">
        <el-alert type="info" :title="t('console.batch.payloadHint')" :closable="false" show-icon class="hint" />
        <div class="device-name">{{ t('console.batch.deviceName') }}{{ result.device }}</div>
        <div v-for="(e, i) in result.entries" :key="i" class="entry-block">
          <el-alert
            v-if="e.unsupported"
            type="warning"
            :title="`${e.path}：${e.unsupported_reason || t('console.batch.unsupported')}`"
            :closable="false"
          />
          <div v-else class="xml-panes">
            <div class="xml-pane">
              <div class="pane-title">{{ t('console.batch.forward') }}</div>
              <XmlViewer :xml="e.forward_xml" />
            </div>
            <div class="xml-pane">
              <div class="pane-title">{{ t('console.batch.rollback') }}</div>
              <XmlViewer :xml="e.rollback_xml" />
            </div>
          </div>
        </div>
      </el-tab-pane>
      <el-tab-pane :label="t('console.batch.tabDiff')">
        <el-alert type="info" :title="t('console.batch.diffHint', { source: baselineLabel })" :closable="false" show-icon class="hint" />
        <el-table data-test="dryrun-diff" :data="diffRows" size="small">
          <el-table-column :label="t('console.batch.colAttr')" prop="path" min-width="300" />
          <el-table-column :label="t('console.batch.colBefore')" min-width="180">
            <template #default="{ row }">
              <span :class="row.type === 'DELETE' ? 'cell-removed' : row.type === 'MODIFY' ? 'cell-modified' : ''">{{ row.old }}</span>
            </template>
          </el-table-column>
          <el-table-column :label="t('console.batch.colAfter')" min-width="180">
            <template #default="{ row }">
              <span :class="row.type === 'ADD' ? 'cell-added' : row.type === 'MODIFY' ? 'cell-modified' : ''">{{ row.new }}</span>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>
  </el-dialog>
</template>

<script setup lang="ts">
// 试运行弹窗（FE-23/CS-01）：打开即调 preview 接口（纯计算不下发）。
// Tab① 待下发设备数据 = 按条目正向/回滚报文双栏；无 XML 通道条目如实降级
// （CS-03，不伪造报文）。Tab② 网元数据差异对比 = diff 树 + 基线来源标注。
// 失败如实报错且不影响变更集（R08/§9）。
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { previewChangeset, type ChangesetPreviewDataDTO } from '../../api'
import XmlViewer from './XmlViewer.vue'
import { useChangesetStore } from '../../stores/changeset'

const props = defineProps<{ visible: boolean; device: string }>()
const emit = defineEmits<{ (e: 'update:visible', v: boolean): void }>()

const { t } = useI18n()
const changeset = useChangesetStore()

const loading = ref(false)
const error = ref('')
const result = ref<ChangesetPreviewDataDTO | null>(null)

watch(
  () => props.visible,
  (open) => {
    if (open) load()
  },
  { immediate: true },
)

async function load() {
  loading.value = true
  error.value = ''
  result.value = null
  try {
    const res = await previewChangeset(changeset.toRequest(props.device))
    const env = res.data as unknown as { code: number; success: boolean; message?: string; data?: ChangesetPreviewDataDTO }
    if (!env.success || !env.data) {
      error.value = env.message || 'preview failed'
      return
    }
    result.value = env.data
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}

const diffRows = computed(() => (result.value ? result.value.entries.flatMap((e) => e.diff ?? []) : []))

// 基线来源标注：多条目取首个非 none 来源（同请求内后端按锚点 memo，来源一致为常态）。
const baselineLabel = computed(() => {
  const src = result.value?.entries.find((e) => e.baseline_source !== 'none')?.baseline_source ?? 'none'
  const key = { desired: 'baselineDesired', cache: 'baselineCache', device: 'baselineDevice', none: 'baselineNone' }[src]
  return t(`console.batch.${key}`)
})
</script>

<style scoped>
.hint {
  margin-bottom: 10px;
}
.dryrun-loading {
  min-height: 160px;
}
.device-name {
  font-weight: 600;
  color: var(--el-color-primary);
  margin-bottom: 8px;
}
.entry-block {
  margin-bottom: 12px;
}
.xml-panes {
  display: flex;
  gap: 12px;
}
.xml-pane {
  flex: 1;
  border: 1px solid var(--el-border-color);
  border-radius: 2px;
  overflow: hidden;
}
.pane-title {
  font-weight: 600;
  padding: 6px 10px;
  border-bottom: 1px solid var(--el-border-color);
  background: var(--el-fill-color-light);
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
