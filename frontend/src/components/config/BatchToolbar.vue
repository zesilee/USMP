<template>
  <div class="batch-toolbar">
    <el-alert
      v-if="showHint"
      data-test="batch-hint"
      class="batch-hint"
      type="info"
      :title="t('console.batch.hint')"
      show-icon
      @close="hintClosed = true"
    />
    <el-badge :value="count" :hidden="count === 0" class="batch-badge">
      <el-button data-test="batch-changes" @click="changesOpen = true">
        {{ t('console.batch.changes') }}
      </el-button>
    </el-badge>
    <el-button data-test="batch-dryrun" :disabled="count === 0" @click="dryRunOpen = true">
      {{ t('console.batch.dryRun') }}
    </el-button>
    <el-button data-test="batch-reset" :disabled="count === 0" @click="onReset">
      {{ t('console.batch.reset') }}
    </el-button>
    <el-button data-test="batch-commit" type="primary" :disabled="count === 0" @click="emit('commit-request')">
      {{ t('console.batch.commit') }}
    </el-button>

    <ChangesContentDialog v-model:visible="changesOpen" :device="device" />
    <DryRunDialog v-model:visible="dryRunOpen" :device="device" />
  </div>
</template>

<script setup lang="ts">
// 攒批工具栏（FE-23，一期预留位落座）：变更内容（徽标=当前设备未提交条目数）/
// 试运行/重置/提交配置。变更集为空时后三者禁用；有变更时展示提示条（可关闭，
// 清空后再攒新变更重新出现）。提交编排由页面层承担（emit commit-request）。
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessageBox } from 'element-plus'
import { useChangesetStore } from '../../stores/changeset'
import ChangesContentDialog from './ChangesContentDialog.vue'
import DryRunDialog from './DryRunDialog.vue'

const props = defineProps<{ device: string }>()
const emit = defineEmits<{ (e: 'reset'): void; (e: 'commit-request'): void }>()

const { t } = useI18n()
const changeset = useChangesetStore()

const count = computed(() => changeset.countFor(props.device))

const changesOpen = ref(false)
const dryRunOpen = ref(false)

// 提示条：有变更即显；用户关闭后保持隐藏，直到变更集清零后再次攒入（0→>0 复位）。
const hintClosed = ref(false)
watch(count, (n, prev) => {
  if (prev === 0 && n > 0) hintClosed.value = false
})
const showHint = computed(() => count.value > 0 && !hintClosed.value)

// 重置（FE-23）：确认后清空当前设备变更集；页面层收 reset 事件恢复表单/标记行。
async function onReset() {
  try {
    await ElMessageBox.confirm(
      t('console.batch.resetConfirm', { count: count.value }),
      t('console.batch.resetConfirmTitle'),
      { type: 'warning' },
    )
  } catch {
    return
  }
  changeset.clear(props.device)
  emit('reset')
}
</script>

<style scoped>
.batch-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
}
.batch-hint {
  padding: 4px 10px;
  width: auto;
}
.batch-badge :deep(.el-badge__content) {
  z-index: 1;
}
</style>
