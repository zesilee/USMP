<template>
  <div class="module-form-tab">
    <!-- 设备不支持占位态（FE-24）：预标记或运行中学习到 node-unsupported 时整体
         替换表单区——无入集/下发入口；重试走 force_refresh 逃生通道。 -->
    <div v-if="nodeUnsupported" class="node-unsupported" data-test="node-unsupported">
      <el-empty :description="t('console.nodeUnsupported')">
        <el-button type="primary" data-test="node-unsupported-retry" @click="load(true)">
          {{ t('console.nodeUnsupportedRetry') }}
        </el-button>
      </el-empty>
    </div>
    <template v-else>
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
      <el-button type="primary" :disabled="!device || !form.submittable.value" @click="submit">{{ t('console.stageChange') }}</el-button>
      <span class="form-tip">{{ t('console.formTipPresence') }}</span>
    </div>
    <div v-else class="actions">
      <span class="form-tip">{{ t('console.readonlyTip') }}</span>
    </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Key, Delete } from '@element-plus/icons-vue'
import { ElMessage, type FormInstance } from 'element-plus'
import { getConfig } from '../../api'
import { nodeUnsupportedFromEnvelope, nodeUnsupportedFromError } from '../../utils/nodeSupport'
import { useConfigForm } from '../../composables/useConfigForm'
import { useChangesetStore } from '../../stores/changeset'
import { evalPredicate } from '../../utils/xpathEval'
import type { Field } from '../../utils/crdSchemaParser'
import type { ConsoleTab } from '../../utils/moduleConsole'
import { configPathFor, deriveDetailTabs, leafName } from '../../utils/moduleConsole'
import FieldRenderer from './FieldRenderer.vue'

const props = defineProps<{
  tab: ConsoleTab
  rootName: string
  device: string
  /** FE-24：schema unsupported 预标记（CN-05），true 时不发取数、渲染占位态。 */
  unsupported?: boolean
}>()

const { t } = useI18n()

// 「基本属性」合成 Tab（path 为空）挂在模块根路径下。
const configPath = computed(() =>
  configPathFor(props.rootName, props.tab.field.path || `/${props.rootName}`),
)
// readonly 叶保留渲染（禁用态回显 state 值），payload/校验排除由 useConfigForm 处理（FE-14）。
const fields = computed<Field[]>(() => props.tab.field.fields || [])
// 攒批模式（FE-03）：removals 开启——基线有值被清 = 删除意图入 diff。
const form = useConfigForm(fields, undefined, { removals: true })
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

// ===== 设备不支持占位态（FE-24）=====
// 初始 = schema 预标记（CN-05，进控制台零请求直接占位）；运行中收到结构化
// node-unsupported 即时转态。重试（load(true)）走 force_refresh 逃生通道，
// 后端成功即自动清标记，本地同步恢复正常渲染。
const nodeUnsupported = ref(!!props.unsupported)
watch(() => props.unsupported, (v) => {
  const was = nodeUnsupported.value
  nodeUnsupported.value = !!v
  // 预标记解除（设备切换后新 schema 落定）：恢复取数，不留空表单。
  if (was && !v) void load()
})

// ===== 读回填 =====
async function load(force = false) {
  error.value = ''
  if (!props.device) {
    form.resetForm()
    return
  }
  // 预标记/已学习不支持：不打设备（FE-24）；重试显式 force 才放行。
  if (nodeUnsupported.value && !force) return
  try {
    // 只读 Tab（config false state 子树）走 <get> 状态通道：<get-config> 对
    // 此类节点会被真机以 unknown-element 拒绝（FE-14 真机回归）。
    const res = await getConfig(props.device, configPath.value, force, !!props.tab.readonly)
    // 信封为 HTTP 200 统一格式：不支持语义在成功回调里识别（不弹裸错误）。
    if (nodeUnsupportedFromEnvelope(res.data)) {
      nodeUnsupported.value = true
      return
    }
    nodeUnsupported.value = false
    const payload = res.data?.data
    const subtree = payload?.data ?? payload ?? {}
    const seed: Record<string, any> = {}
    for (const f of fields.value) {
      const k = form.keyOf(f)
      if (subtree[k] !== undefined) seed[k] = subtree[k]
    }
    form.resetForm(seed)
  } catch (e: any) {
    // 不支持语义（非 200 兜底形态）同转占位，不显示裸错误（FE-24）。
    if (nodeUnsupportedFromError(e)) {
      nodeUnsupported.value = true
      return
    }
    // 后端暂不支持该路径读时如实降级：空表单 + 告警（§9，不伪装成功）。
    form.resetForm()
    error.value = e?.response?.data?.message || e?.message || t('console.readFailed')
  }
}

watch(() => props.device, () => load(), { immediate: true })

// ===== 确定=入变更集（FE-03 攒批）：不发写请求；归属硬锁与下发经「提交配置」 =====
const changeset = useChangesetStore()

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
  changeset.upsert(props.device, {
    op: 'update',
    path: '/' + configPath.value,
    // 只发改动字段（真机 unknown-element 回归）：容器表单同样被回读值填满，
    // 原样回推会被设备能力裁剪的叶拒绝。
    payload: form.changedPayload(),
    cleared: form.clearedKeys.value,
    baseline: { ...form.original.value },
    label: props.tab.label,
  })
  ElMessage.success(t('console.stagedOk'))
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
