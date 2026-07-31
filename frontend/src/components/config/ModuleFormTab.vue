<template>
  <div class="module-form-tab">
    <el-alert v-if="error" :title="error" type="warning" :closable="false" show-icon />

    <el-form ref="formRef" :model="form.formData" :rules="form.rules.value" label-position="top" class="config-form">
      <!-- 嵌套 group → 二级 Tab（FE-02 NCE 形态）：>1 分组渲染 Tab 头；面板常驻
           （隐藏非销毁）保住跨 Tab 校验/diff 与状态。 -->
      <el-tabs v-if="innerTabs.length > 1" v-model="activeInner" class="inner-tabs">
        <el-tab-pane v-for="tt in innerTabs" :key="tt.name" :label="tt.label" :name="tt.name">
          <div class="config-form--grid">
            <el-form-item v-for="field in paneFields(tt)" :key="field.path" :prop="form.keyOf(field)"
              :error="presenceMustError(field)" :class="{ 'fi-span-full': spansFull(field) }">
              <template #label>
                <span class="fi-label">
                  <el-icon v-if="field.isKey" class="key-icon" data-test="key-icon"><Key /></el-icon>
                  <span>{{ labelOf(field) }}</span>
                  <el-tooltip v-if="clearableField(field)" :content="t('console.clearFieldTip')" placement="top">
                    <el-icon class="clear-icon" :data-test="`clear-${form.keyOf(field)}`" @click.stop="clearField(field)"><Delete /></el-icon>
                  </el-tooltip>
                </span>
              </template>
              <FieldRenderer v-if="field.type === 'choice'" :field="field" :model-value="form.choiceScope(field)"
                @update:model-value="form.onChoiceUpdate(field, $event)" />
              <FieldRenderer v-else :field="field" :disabled="presenceBlocked(field) || !!field.readonly"
                :model-value="form.formData[form.keyOf(field)]"
                @update:model-value="form.formData[form.keyOf(field)] = $event" />
            </el-form-item>
          </div>
        </el-tab-pane>
      </el-tabs>
      <div v-else class="config-form--grid">
        <el-form-item v-for="field in paneFields(innerTabs[0])" :key="field.path" :prop="form.keyOf(field)"
          :error="presenceMustError(field)" :class="{ 'fi-span-full': spansFull(field) }">
          <template #label>
            <span class="fi-label">
              <el-icon v-if="field.isKey" class="key-icon" data-test="key-icon"><Key /></el-icon>
              <span>{{ labelOf(field) }}</span>
              <el-tooltip v-if="clearableField(field)" :content="t('console.clearFieldTip')" placement="top">
                <el-icon class="clear-icon" :data-test="`clear-${form.keyOf(field)}`" @click.stop="clearField(field)"><Delete /></el-icon>
              </el-tooltip>
            </span>
          </template>
          <FieldRenderer v-if="field.type === 'choice'" :field="field" :model-value="form.choiceScope(field)"
            @update:model-value="form.onChoiceUpdate(field, $event)" />
          <FieldRenderer v-else :field="field" :disabled="presenceBlocked(field) || !!field.readonly"
            :model-value="form.formData[form.keyOf(field)]"
            @update:model-value="form.formData[form.keyOf(field)] = $event" />
        </el-form-item>
      </div>
      <div v-if="!form.visibleFields.value.length" class="empty-tip">{{ t('console.emptyGroup') }}</div>
    </el-form>

    <!-- 整 Tab readonly（config false state 子树）：只读视图，无下发入口（FE-14） -->
    <div v-if="!tab.readonly" class="actions">
      <el-button type="primary" :disabled="!device || !form.submittable.value" @click="submit">{{ t('console.push') }}</el-button>
      <span class="form-tip">{{ t('console.formTipPresence') }}</span>
    </div>
    <div v-else class="actions">
      <span class="form-tip">{{ t('console.readonlyTip') }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Key, Delete } from '@element-plus/icons-vue'
import { ElMessage, type FormInstance } from 'element-plus'
import { getConfig, setConfig } from '../../api'
import { ownershipRejectionOf, confirmOwnershipOverride } from '../../composables/ownershipGate'
import { useConfigForm } from '../../composables/useConfigForm'
import { evalPredicate } from '../../utils/xpathEval'
import type { Field } from '../../utils/crdSchemaParser'
import type { ConsoleTab } from '../../utils/moduleConsole'
import { configPathFor, deriveDetailTabs, leafName } from '../../utils/moduleConsole'
import FieldRenderer from './FieldRenderer.vue'

const props = defineProps<{
  tab: ConsoleTab
  rootName: string
  device: string
}>()

const { t } = useI18n()

// 「基本属性」合成 Tab（path 为空）挂在模块根路径下。
const configPath = computed(() =>
  configPathFor(props.rootName, props.tab.field.path || `/${props.rootName}`),
)
// readonly 叶保留渲染（禁用态回显 state 值），payload/校验排除由 useConfigForm 处理（FE-14）。
const fields = computed<Field[]>(() => props.tab.field.fields || [])
const form = useConfigForm(fields)
const formRef = ref<FormInstance>()
const error = ref('')

function labelOf(f: Field): string {
  return f.label || leafName(f)
}

// 嵌套 group → 二级 Tab（FE-02 NCE 形态）：复用 deriveDetailTabs（标量→主 Tab、
// 嵌套 group→子表单 Tab、嵌套 list→子表格 Tab）。单 Tab 退化不渲染 Tab 头。
const innerTabs = computed(() => deriveDetailTabs(props.tab.field))
const activeInner = ref('__main__')
watch(() => props.tab, () => { activeInner.value = '__main__' })

function paneFields(tt?: { name: string; field: Field }): Field[] {
  if (!tt) return []
  const visible = new Set(form.visibleFields.value.map((f) => f.path))
  if (tt.name === '__main__') return (tt.field.fields || []).filter((f) => visible.has(f.path))
  return [tt.field].filter((f) => visible.has(f.path))
}

// NCE 三列栅格（FE-22）：宽控件占整行。
function spansFull(f: Field): boolean {
  return f.type === 'choice' || f.type === 'leaf-list' || f.type === 'list' || f.type === 'group'
}

// 字段级清除（FE-22）：可编辑且有值才出清除钮；choice 成员键分散不提供整体清除。
function clearableField(f: Field): boolean {
  if (f.type === 'choice' || f.readonly || presenceBlocked(f)) return false
  return form.formData[form.keyOf(f)] !== undefined
}

function clearField(f: Field) {
  delete form.formData[form.keyOf(f)]
}

// ===== presence 容器的 must 门禁（FE-12）=====
// must 依赖同级字段（如 ../ipv4-ignore-primary-sub='false'）：不满足 → 开关禁用并
// 强制关闭（节点不可存在）；求值失败降级为可用（R08）。
function presenceMustSatisfied(f: Field): boolean {
  if (!(f.type === 'group' && f.presence) || !f.must?.length) return true
  return f.must.every((m) => {
    const r = evalPredicate(m.expr, form.formData)
    return 'error' in r && r.error !== undefined ? true : !!r.value
  })
}

function presenceBlocked(f: Field): boolean {
  return f.type === 'group' && !!f.presence && !presenceMustSatisfied(f)
}

function presenceMustError(f: Field): string {
  if (!presenceBlocked(f)) return ''
  return f.must?.[0]?.message || t('console.presenceCondNotMet', { label: labelOf(f) })
}

// must 变为不满足时强制关闭 presence（键删除 = 节点不存在）。
watch(
  () => fields.value.filter(presenceBlocked).map(form.keyOf),
  (blockedKeys) => {
    for (const k of blockedKeys) {
      if (form.formData[k] !== undefined) delete form.formData[k]
    }
  },
)

// ===== 读回填 =====
async function load() {
  error.value = ''
  if (!props.device) {
    form.resetForm()
    return
  }
  try {
    const res = await getConfig(props.device, configPath.value)
    const payload = res.data?.data
    const subtree = payload?.data ?? payload ?? {}
    const seed: Record<string, any> = {}
    for (const f of fields.value) {
      const k = form.keyOf(f)
      if (subtree[k] !== undefined) seed[k] = subtree[k]
    }
    form.resetForm(seed)
  } catch (e: any) {
    // 后端暂不支持该路径读时如实降级：空表单 + 告警（§9，不伪装成功）。
    form.resetForm()
    error.value = e?.response?.data?.message || e?.message || t('console.readFailed')
  }
}

watch(() => props.device, load, { immediate: true })

// ===== 下发 =====
async function submit() {
  if (!props.device) return
  if (formRef.value) {
    try {
      await formRef.value.validate()
    } catch {
      /* 行内提示；权威门禁在下方 */
    }
  }
  if (form.blocked.value) return
  try {
    let res = await setConfig(props.device, configPath.value, form.visiblePayload())
    // 归属硬锁（FE-18 二期）：信封 409 → 阻断确认 → 确认后携 force 重发，取消则中止
    // （不置错误态、不刷新）。
    const rej = ownershipRejectionOf(res)
    if (rej) {
      if (!(await confirmOwnershipOverride(rej))) return
      res = await setConfig(props.device, configPath.value, form.visiblePayload(), true)
      // 信封恒 200：force 重发失败按 success 判定，如实透出（§9）。
      if ((res.data as any)?.success === false) {
        error.value = (res.data as any)?.message || t('console.forcePushFailed')
        return
      }
    }
    // 软归属警告（FE-18/BR-11）：force 放行仍附带，非阻断提示，下发照常。
    const warn = (res.data as any)?.data?.ownershipWarning
    if (warn?.message) {
      ElMessage.warning(`${warn.message}（${(warn.intents || []).join('、')}）`)
    } else {
      ElMessage.success(t('console.pushed'))
    }
    await load()
  } catch (e: any) {
    // 后端不支持写入的路径（如尚无转换器）原样透出错误（§9）。
    error.value = e?.response?.data?.message || e?.message || t('console.pushFailed')
  }
}
</script>

<style scoped>
.module-form-tab {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

/* NCE 三列栅格（FE-22）：when 隐藏字段不渲染即不占位 */
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

.config-form {
  padding: 0 4px;
}

.actions {
  display: flex;
  align-items: center;
  gap: 14px;
}

.form-tip {
  font-size: 11.5px;
  color: var(--ink-3, #93a2b1);
}

.empty-tip {
  color: var(--ink-3, #93a2b1);
  font-size: 13px;
}
</style>
