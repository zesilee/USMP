<template>
  <div class="item-detail-pane" data-test="item-detail-pane">
    <!-- 条目面包屑 + 关闭（FE-21）：`<list 标签> > <主键值|创建>` -->
    <div class="detail-header">
      <el-breadcrumb separator=">" data-test="detail-breadcrumb" class="detail-crumb">
        <el-breadcrumb-item>{{ tab.label }}</el-breadcrumb-item>
        <el-breadcrumb-item>{{ crumbLeaf }}</el-breadcrumb-item>
      </el-breadcrumb>
      <el-button data-test="detail-close" size="small" @click="emit('close')">{{ t('common.close') }}</el-button>
    </div>

    <template v-if="!flowActive">
      <!-- 二级 Tab（deriveDetailTabs 驱动）：仅存在嵌套子节点时渲染 Tab 头（FE-21 退化边界） -->
      <el-tabs v-if="detailTabs.length > 1" v-model="activeDetail" class="detail-tabs">
        <el-tab-pane v-for="dt in detailTabs" :key="dt.name" :label="dt.label" :name="dt.name" />
      </el-tabs>

      <el-form ref="formRef" :model="form.formData" :rules="form.rules.value" label-position="top" class="config-form">
        <!-- NCE 三列栅格（FE-22）：标量流式补位，choice/leaf-list/嵌套容器整行 -->
        <div class="config-form--grid">
          <el-form-item v-for="field in activeFields" :key="field.path" :prop="form.keyOf(field)"
            :class="{ 'fi-span-full': spansFull(field) }">
            <template #label>
              <span class="fi-label">
                <!-- key 叶钥匙标识（FE-22，R12 真实图标） -->
                <el-icon v-if="field.isKey" class="key-icon" data-test="key-icon"><Key /></el-icon>
                <span>{{ field.label }}</span>
                <!-- 字段级清除（FE-22）：本地清空=本次不下发该叶，非设备侧删除 -->
                <el-tooltip v-if="clearableField(field)" :content="t('console.clearFieldTip')" placement="top">
                  <el-icon class="clear-icon" :data-test="`clear-${form.keyOf(field)}`" @click.stop="clearField(field)"><Delete /></el-icon>
                </el-tooltip>
              </span>
            </template>
            <FieldRenderer v-if="field.type === 'choice'" :field="field" :model-value="form.choiceScope(field)"
              @update:model-value="form.onChoiceUpdate(field, $event)" />
            <FieldRenderer v-else :field="field" :disabled="isFieldDisabled(field)"
              :model-value="form.formData[form.keyOf(field)]"
              @update:model-value="form.formData[form.keyOf(field)] = $event" />
          </el-form-item>
        </div>
        <el-empty v-if="!activeFields.length" :description="t('console.emptyGroup')" />
      </el-form>

      <DiffPreview :diff="form.diff.value" />
      <div class="detail-footer">
        <el-button @click="emit('close')">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" data-test="detail-submit" :disabled="!form.submittable.value" @click="submit">
          {{ t('console.pushAndReconcile') }}
        </el-button>
      </div>
    </template>

    <!-- 下发/对账进度（编排复用，即时下发语义不变） -->
    <template v-else>
      <ReconcileSteps :progress="submitFlow.progress.value" :timed-out="submitFlow.timedOut.value" />
      <div class="detail-footer">
        <el-button type="primary" :disabled="!flowDone" @click="submitFlow.reset()">
          {{ flowDone ? t('console.backToForm') : t('console.reconciling') }}
        </el-button>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Key, Delete } from '@element-plus/icons-vue'
import type { FormInstance } from 'element-plus'
import { useConfigForm } from '../../composables/useConfigForm'
import { useConfigSubmit } from '../../composables/useConfigSubmit'
import type { Field } from '../../utils/crdSchemaParser'
import {
  deriveDetailTabs,
  deriveKeyField,
  configPathFor,
  leafName,
  type ConsoleTab,
} from '../../utils/moduleConsole'
import FieldRenderer from './FieldRenderer.vue'
import DiffPreview from './DiffPreview.vue'
import ReconcileSteps from './ReconcileSteps.vue'

const props = defineProps<{
  tab: ConsoleTab
  rootName: string
  device: string
  mode: 'edit' | 'create'
  row: Record<string, any> | null
  /** POST 包裹键（回读命中容器名时跟随实际键，与列表侧同源） */
  postKey: string
}>()

const emit = defineEmits<{ close: []; saved: [key: string] }>()
const { t } = useI18n()

const listField = computed<Field>(() => props.tab.listField || props.tab.field)
const keyField = computed(() => deriveKeyField(listField.value))
const detailTabs = computed(() => deriveDetailTabs(listField.value))
const configPath = computed(() => configPathFor(props.rootName, props.tab.field.path))
const itemFields = computed<Field[]>(() => listField.value.fields || [])

const form = useConfigForm(itemFields, keyField)
const submitOpts = reactive({ configPath: '', listKey: '' })
watch([configPath, () => props.postKey, listField], () => {
  submitOpts.configPath = configPath.value
  submitOpts.listKey = props.postKey || leafName(listField.value)
}, { immediate: true })
const submitFlow = useConfigSubmit(submitOpts)

const flowActive = computed(() => submitFlow.phase.value !== 'idle')
const flowDone = computed(() => submitFlow.progress.value.done || submitFlow.timedOut.value)

const activeDetail = ref('__main__')
const formRef = ref<FormInstance>()

// 面包屑叶：编辑=行主键值（缺失回退空串不崩，R08），创建=创建文案。
const crumbLeaf = computed(() =>
  props.mode === 'create' ? t('common.create') : String(props.row?.[keyField.value] ?? ''),
)

// 二级 Tab → 字段面：主 Tab 取非容器子叶、嵌套 Tab 取该单一子节点；均按约束引擎
// 可见性过滤（when 隐藏 = 不渲染，FE-07 语义跨 Tab 一致）。
const activeFields = computed<Field[]>(() => {
  const dt = detailTabs.value.find((d) => d.name === activeDetail.value) || detailTabs.value[0]
  if (!dt) return []
  const visible = new Set(form.visibleFields.value.map((f) => f.path))
  if (dt.name === '__main__') return (dt.field.fields || []).filter((f) => visible.has(f.path))
  return [dt.field].filter((f) => visible.has(f.path))
})

// 门禁（FE-11 编辑态 identity 禁用 + FE-22 key 编辑态只读）：readonly 恒禁用。
function isFieldDisabled(f: Field): boolean {
  if (f.readonly) return true
  return props.mode === 'edit' && (!!f.isKey || !!f.operationExclude?.includes('update'))
}

// NCE 三列栅格（FE-22）：宽控件占整行。
function spansFull(f: Field): boolean {
  return f.type === 'choice' || f.type === 'leaf-list' || f.type === 'list' || f.type === 'group'
}

// 字段级清除（FE-22）：可编辑且有值才出清除钮；choice 成员键分散，不提供整体清除。
function clearableField(f: Field): boolean {
  if (f.type === 'choice' || isFieldDisabled(f)) return false
  return form.formData[form.keyOf(f)] !== undefined
}

function clearField(f: Field) {
  delete form.formData[form.keyOf(f)]
}

// 未提交草稿（FE-21 切行确认判据）：有 diff 且不在下发流中。
const dirty = computed(() => !flowActive.value && form.diff.value.length > 0)

// mode/row 变化即重置表单与流（切行/切建）；行内容以主键为准判定变化。
watch(
  [() => props.mode, () => props.row],
  () => {
    submitFlow.reset()
    form.resetForm(props.row ? { ...props.row } : {})
    formRef.value?.clearValidate()
    activeDetail.value = '__main__'
  },
  { immediate: true },
)

async function submit() {
  if (!props.device) return
  if (formRef.value) {
    try {
      await formRef.value.validate()
    } catch {
      /* 行内提示即可；下方权威门禁拦截 */
    }
  }
  if (form.blocked.value) return
  const keyValue = String(form.formData[keyField.value] ?? '')
  await submitFlow.run(props.device, form.visiblePayload())
  if (submitFlow.phase.value !== 'error') emit('saved', keyValue)
}

defineExpose({ dirty, form, submit, isFieldDisabled })
</script>

<style scoped>
.item-detail-pane {
  display: flex;
  flex-direction: column;
  gap: 12px;
  background: #fff;
  border-radius: 8px;
  border: 1px solid var(--line, #e4e7ed);
  padding: 12px 16px 16px;
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.detail-crumb :deep(.el-breadcrumb__inner) {
  font-weight: 600;
}

.detail-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

/* NCE 三列栅格（FE-22）：when 隐藏字段不渲染即不占位，流式补位 */
.config-form--grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  column-gap: 32px;
}

.fi-span-full {
  grid-column: 1 / -1;
}

@media (max-width: 1280px) {
  .config-form--grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 900px) {
  .config-form--grid {
    grid-template-columns: 1fr;
  }
}

.fi-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.key-icon {
  color: var(--ink-2, #5c6b7a);
}

.clear-icon {
  cursor: pointer;
  color: var(--ink-3, #93a2b1);
}

.clear-icon:hover {
  color: var(--st-off, #c45656);
}
</style>
