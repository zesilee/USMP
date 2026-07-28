<template>
  <div class="rpc-execute-tab">
    <div class="rpc-head">
      <span class="rpc-name">{{ rpc.label || rpc.name }}</span>
      <el-tag v-if="rpc.highRisk" type="danger" size="small" effect="dark" data-test="rpc-highrisk">
        {{ t('console.rpc.highRisk') }}
      </el-tag>
    </div>
    <p class="rpc-tip">{{ t('console.rpc.tip') }}</p>

    <el-form label-position="top" class="rpc-form">
      <el-form-item v-for="f in resolvedInputs" :key="f.path" :label="labelOf(f)" :required="!!f.required">
        <FieldRenderer :field="f" :model-value="values[keyOf(f)]"
          @update:model-value="values[keyOf(f)] = $event" />
      </el-form-item>
      <div v-if="!resolvedInputs.length" class="rpc-empty">{{ t('console.rpc.noInput') }}</div>
    </el-form>

    <div class="actions">
      <el-button type="primary" :disabled="!device || !submittable || running"
        :loading="running" data-test="rpc-execute" @click="execute">
        {{ t('console.rpc.execute') }}
      </el-button>
    </div>

    <el-alert v-if="result" :type="resultType" :title="result" :closable="true" show-icon
      data-test="rpc-result" @close="result = ''" />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessageBox } from 'element-plus'
import { executeRpc, getConfig } from '../../api'
import type { Field } from '../../utils/crdSchemaParser'
import type { ConsoleTab, RpcDef } from '../../utils/moduleConsole'
import { leafName } from '../../utils/moduleConsole'
import { extractRows } from '../../composables/useDeviceConfig'
import { parseLeafref } from '../../utils/leafref'
import FieldRenderer from './FieldRenderer.vue'

const props = defineProps<{
  tab: ConsoleTab
  module: string
  device: string
}>()

const { t } = useI18n()

const rpc = computed<RpcDef>(() => props.tab.rpc || { name: '', label: '', input: [] })
const values = reactive<Record<string, any>>({})
const running = ref(false)
const result = ref('')
const resultType = ref<'success' | 'error'>('success')

// leafref 输入的目标值下拉选项（键=输入叶名，FE-19）。设备/rpc 变化时按需拉取。
const leafrefOptions = reactive<Record<string, { label: string; value: string }[]>>({})

function keyOf(f: Field): string {
  return leafName(f)
}
function labelOf(f: Field): string {
  return f.label || leafName(f)
}

// 渲染用的输入字段：leafref 且已解析出目标值 → 注入 options（FieldRenderer 渲成
// 下拉）；否则原样（文本框）。设备离线/拉取失败时 options 为空 → 降级文本输入（R08）。
const resolvedInputs = computed<Field[]>(() =>
  rpc.value.input.map((f) => {
    const opts = leafrefOptions[keyOf(f)]
    return f.leafRef && opts?.length ? { ...f, options: opts } : f
  }),
)

// 拉取每个 leafref 输入的目标列表 → 下拉选项。解析路径经 getConfig+extractRows，
// 失败静默降级（保留文本输入，不打扰执行主流程）。
async function loadLeafrefOptions() {
  for (const k of Object.keys(leafrefOptions)) delete leafrefOptions[k]
  if (!props.device) return
  for (const f of rpc.value.input) {
    const target = parseLeafref(f.leafRef)
    if (!target) continue
    try {
      const res = await getConfig(props.device, target.fetchPath)
      const rows = extractRows(res.data?.data, target.listKey, target.keyField)
      const opts = rows
        .map((r) => String(r[target.keyField] ?? ''))
        .filter((v) => v !== '')
        .map((v) => ({ label: v, value: v }))
      if (opts.length) leafrefOptions[keyOf(f)] = opts
    } catch {
      /* 拉取失败降级文本输入（R08） */
    }
  }
}

watch(() => [props.device, rpc.value.name], loadLeafrefOptions, { immediate: true })

// 所有 mandatory input 有值才可执行（§9：校验拦截，不下发）。
const submittable = computed(() =>
  rpc.value.input.every((f) => {
    if (!f.required) return true
    const v = values[keyOf(f)]
    return v !== undefined && v !== null && String(v).trim() !== ''
  }),
)

function collectInputs(): Record<string, string> {
  const out: Record<string, string> = {}
  for (const f of rpc.value.input) {
    const v = values[keyOf(f)]
    if (v !== undefined && v !== null && String(v) !== '') out[keyOf(f)] = String(v)
  }
  return out
}

// 执行：先二次确认（高危升级警示），确认后调执行 API，回显结果/错误。
async function execute() {
  if (!props.device || !submittable.value || running.value) return

  const inputs = collectInputs()
  const detail = Object.entries(inputs).map(([k, v]) => `${k} = ${v}`).join('，') || t('console.rpc.noInput')
  const confirmMsg = t('console.rpc.confirmBody', {
    rpc: rpc.value.label || rpc.value.name,
    device: props.device,
    inputs: detail,
  })
  try {
    await ElMessageBox.confirm(confirmMsg, t('console.rpc.confirmTitle'), {
      type: rpc.value.highRisk ? 'error' : 'warning',
      confirmButtonText: t('console.rpc.confirmOk'),
      cancelButtonText: t('common.cancel'),
      confirmButtonClass: rpc.value.highRisk ? 'el-button--danger' : undefined,
    })
  } catch {
    return // 取消 → 不下发
  }

  running.value = true
  result.value = ''
  try {
    const res = await executeRpc(props.device, props.module, rpc.value.name, inputs)
    const body: any = res.data
    if (body?.success === false) {
      resultType.value = 'error'
      result.value = body?.message || t('console.rpc.failed')
    } else {
      resultType.value = 'success'
      const reply = body?.data?.reply
      result.value = reply && String(reply).trim()
        ? t('console.rpc.doneReply', { reply: String(reply).slice(0, 300) })
        : t('console.rpc.done')
    }
  } catch (e: any) {
    resultType.value = 'error'
    result.value = e?.response?.data?.message || e?.message || t('console.rpc.failed')
  } finally {
    running.value = false
  }
}
</script>

<style scoped>
.rpc-execute-tab {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-width: 560px;
}
.rpc-head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.rpc-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--ink-1, #1f2d3d);
}
.rpc-tip {
  font-size: 11.5px;
  color: var(--ink-3, #93a2b1);
  margin: 0;
}
.rpc-empty {
  color: var(--ink-3, #93a2b1);
  font-size: 13px;
}
.actions {
  display: flex;
  gap: 12px;
}
</style>
