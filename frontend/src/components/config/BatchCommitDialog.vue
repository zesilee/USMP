<template>
  <el-dialog
    :model-value="visible"
    :title="t('console.batch.commit')"
    width="560px"
    :close-on-click-modal="false"
    :show-close="done"
    @update:model-value="onDialogToggle"
  >
    <el-alert
      v-if="flow.error.value"
      data-test="commit-error"
      type="error"
      :title="flow.error.value"
      :closable="false"
      show-icon
    />
    <ReconcileSteps :progress="flow.progress.value" :timed-out="flow.timedOut.value" />
    <div class="dialog-footer">
      <el-button type="primary" data-test="commit-close" :disabled="!done" @click="close">
        {{ done ? t('common.close') : t('console.reconciling') }}
      </el-button>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
// 提交进度弹窗（FE-03 攒批）：打开即执行变更集提交编排（commit→回读→轮询对账），
// ReconcileSteps 复用既有状态机呈现 pushing→reading→终局。失败如实展示且变更集
// 保留（useChangesetSubmit 语义）；成功后 emit committed 供页面刷新列表/表单。
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useChangesetSubmit } from '../../composables/useChangesetSubmit'
import ReconcileSteps from './ReconcileSteps.vue'

const props = defineProps<{ visible: boolean; device: string }>()
const emit = defineEmits<{ (e: 'update:visible', v: boolean): void; (e: 'committed'): void }>()

const { t } = useI18n()
const flow = useChangesetSubmit()

const done = computed(
  () => flow.progress.value.done || flow.timedOut.value || flow.phase.value === 'error' || flow.phase.value === 'idle',
)

watch(
  () => props.visible,
  async (open) => {
    if (!open) return
    flow.reset()
    const committed = await flow.run(props.device)
    if (committed) emit('committed')
    if (flow.phase.value === 'idle') emit('update:visible', false) // 空集/用户取消 force：静默收起
  },
  { immediate: true },
)

function close() {
  emit('update:visible', false)
}

function onDialogToggle(v: boolean) {
  if (!v && !done.value) return // 进行中不允许点遮罩关闭
  emit('update:visible', v)
}
</script>

<style scoped>
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}
</style>
