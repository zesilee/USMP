<template>
  <div class="module-form-tab">
    <el-alert v-if="error" :title="error" type="warning" :closable="false" show-icon />

    <el-form ref="formRef" :model="form.formData" :rules="form.rules.value" label-position="top" class="config-form">
      <el-form-item v-for="field in form.visibleFields.value" :key="field.path"
        :label="labelOf(field)" :prop="form.keyOf(field)"
        :error="presenceMustError(field)">
        <FieldRenderer v-if="field.type === 'choice'" :field="field" :model-value="form.choiceScope(field)"
          @update:model-value="form.onChoiceUpdate(field, $event)" />
        <FieldRenderer v-else :field="field" :disabled="presenceBlocked(field) || !!field.readonly"
          :model-value="form.formData[form.keyOf(field)]"
          @update:model-value="form.formData[form.keyOf(field)] = $event" />
      </el-form-item>
      <div v-if="!form.visibleFields.value.length" class="empty-tip">该分组暂无可配置字段。</div>
    </el-form>

    <!-- 整 Tab readonly（config false state 子树）：只读视图，无下发入口（FE-14） -->
    <div v-if="!tab.readonly" class="actions">
      <el-button type="primary" :disabled="!device || !form.submittable.value" @click="submit">下发</el-button>
      <span class="form-tip">字段与约束由 YANG 模型生成；presence 开关关闭即该节点不存在。</span>
    </div>
    <div v-else class="actions">
      <span class="form-tip">该分组为设备状态数据（config false），仅供查看。</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage, type FormInstance } from 'element-plus'
import { getConfig, setConfig } from '../../api'
import { useConfigForm } from '../../composables/useConfigForm'
import { evalPredicate } from '../../utils/xpathEval'
import type { Field } from '../../utils/crdSchemaParser'
import type { ConsoleTab } from '../../utils/moduleConsole'
import { configPathFor, leafName } from '../../utils/moduleConsole'
import FieldRenderer from './FieldRenderer.vue'

const props = defineProps<{
  tab: ConsoleTab
  rootName: string
  device: string
}>()

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
  return f.must?.[0]?.message || `${labelOf(f)} 的启用条件不满足`
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
    error.value = e?.response?.data?.message || e?.message || '读取失败'
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
    const res = await setConfig(props.device, configPath.value, form.visiblePayload())
    // 软归属警告（FE-18/BR-11）：命中业务意图认领路径时非阻断提示，下发照常。
    const warn = (res.data as any)?.data?.ownershipWarning
    if (warn?.message) {
      ElMessage.warning(`${warn.message}（${(warn.intents || []).join('、')}）`)
    } else {
      ElMessage.success('已下发')
    }
    await load()
  } catch (e: any) {
    // 后端不支持写入的路径（如尚无转换器）原样透出错误（§9）。
    error.value = e?.response?.data?.message || e?.message || '下发失败'
  }
}
</script>

<style scoped>
.module-form-tab {
  display: flex;
  flex-direction: column;
  gap: 14px;
  max-width: 640px;
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
