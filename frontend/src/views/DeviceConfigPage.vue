<template>
  <div class="device-config">
    <div class="page-header">
      <h2>{{ title }}</h2>
      <div class="header-actions">
        <el-select v-model="selectedDevice" placeholder="选择设备" style="width: 220px" @change="reload">
          <el-option v-for="d in store.devices" :key="d.id" :label="d.ip" :value="d.ip" />
        </el-select>
        <el-button type="primary" :icon="Plus" :disabled="!selectedDevice" @click="openAdd">
          {{ addLabel }}
        </el-button>
      </div>
    </div>

    <el-alert v-if="cfg.error.value" :title="cfg.error.value" type="warning" :closable="false" show-icon
      style="margin-bottom: 16px" />

    <div class="cfg">
      <SchemaTree class="cfg-tree" :fields="cfg.schemaFields.value" :key-field="options.keyField"
        :module-label="options.module" :item-counts="itemCounts" />

      <el-table :data="cfg.items.value" stripe v-loading="cfg.loading.value" class="config-table">
        <el-table-column v-for="col in columns" :key="col.prop" :prop="col.prop" :label="col.label"
          :width="col.width" :min-width="col.width ? undefined : 160" />
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" size="small" link @click="openEdit(row)">编辑</el-button>
          </template>
        </el-table-column>
        <template #empty>
          <span>{{ selectedDevice ? '暂无配置（点击新增）' : '请先选择设备' }}</span>
        </template>
      </el-table>
    </div>

    <el-drawer v-model="drawerVisible" :title="editing ? '编辑' : addLabel" size="560px"
      :close-on-click-modal="!flowActive" :close-on-press-escape="!flowActive" @closed="onDrawerClosed">
      <!-- idle：模型驱动表单 + 实时差异预览 -->
      <template v-if="!flowActive">
        <el-form ref="formRef" :model="formData" :rules="rules" label-position="top" class="config-form">
          <el-form-item v-for="field in cfg.fields.value" :key="field.path" :label="field.label"
            :prop="keyOf(field)">
            <FieldRenderer :field="field" :model-value="formData[keyOf(field)]"
              @update:model-value="formData[keyOf(field)] = $event" />
          </el-form-item>
        </el-form>
        <DiffPreview :diff="diff" />
        <div class="form-tip">字段与约束由 YANG 模型生成，校验通过才会下发，下发即触发对账。</div>
      </template>
      <!-- 下发中/后：真实对账三步进度 -->
      <ReconcileSteps v-else :progress="submitFlow.progress.value" :timed-out="submitFlow.timedOut.value" />

      <template #footer>
        <template v-if="!flowActive">
          <el-button @click="drawerVisible = false">取消</el-button>
          <el-button type="primary" :disabled="!submittable" @click="submit">下发并对账</el-button>
        </template>
        <el-button v-else type="primary" :disabled="!flowDone" @click="drawerVisible = false">
          {{ flowDone ? '关闭' : '对账中…' }}
        </el-button>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { type FormInstance, type FormRules } from 'element-plus'
import { useDeviceStore } from '../stores/device'
import { useDeviceConfig, type DeviceConfigOptions } from '../composables/useDeviceConfig'
import { useConfigSubmit } from '../composables/useConfigSubmit'
import { computeDiff, missingRequired } from '../utils/configDiff'
import type { Field } from '../utils/crdSchemaParser'
import FieldRenderer from '../components/config/FieldRenderer.vue'
import SchemaTree from '../components/config/SchemaTree.vue'
import DiffPreview from '../components/config/DiffPreview.vue'
import ReconcileSteps from '../components/config/ReconcileSteps.vue'

const props = defineProps<{
  title: string
  addLabel: string
  options: DeviceConfigOptions
  columns: { prop: string; label: string; width?: number }[]
}>()

const store = useDeviceStore()
const cfg = useDeviceConfig(props.options)
const submitFlow = useConfigSubmit({ configPath: props.options.configPath, listKey: props.options.listKey })

const selectedDevice = ref('')
const drawerVisible = ref(false)
const editing = ref(false)
const formData = reactive<Record<string, any>>({})
const original = ref<Record<string, any>>({}) // 已回填的实际态基线（新增时为空），供实时差异比对
const formRef = ref<FormInstance>()

// 抽屉编排态：idle 显示表单+差异预览；flowActive 显示对账进度。
const flowActive = computed(() => submitFlow.phase.value !== 'idle')
const flowDone = computed(() => submitFlow.progress.value.done || submitFlow.timedOut.value)

// 实时差异（表单期望值 ↔ 已回填实际态）；下发按钮 = 有改动 && 无缺失必填。
const diff = computed(() => computeDiff(formData, original.value, cfg.fields.value))
const submittable = computed(
  () => diff.value.length > 0 && missingRequired(cfg.fields.value, formData, props.options.keyField).length === 0,
)

function keyOf(f: Field): string {
  return f.path.split('/').filter(Boolean).pop() || f.path
}

// 架构树上目标 list 的数量 pill：把当前已配置行数挂到该 list 节点 path 上。
const itemCounts = computed<Record<string, number>>(() =>
  cfg.itemListPath.value ? { [cfg.itemListPath.value]: cfg.items.value.length } : {},
)

// 由 schema 生成校验规则：主键(keyField)与 required 叶子必填；数值字段带 min/max 时校验范围。
// 服务端仍有权威兜底(如 VLAN ID 1-4094)，此处提前拦截、行内提示。
const rules = computed<FormRules>(() => {
  const r: FormRules = {}
  for (const f of cfg.fields.value) {
    const key = keyOf(f)
    const list: any[] = []
    if (f.required || key === props.options.keyField) {
      list.push({ required: true, message: `${f.label} 必填`, trigger: ['change', 'blur'] })
    }
    if (f.type === 'number' && (f.minimum != null || f.maximum != null)) {
      list.push({ type: 'number', min: f.minimum, max: f.maximum, message: `${f.label} 超出范围`, trigger: ['change', 'blur'] })
    }
    if (list.length) r[key] = list
  }
  return r
})

function resetForm(seed: Record<string, any> = {}) {
  Object.keys(formData).forEach((k) => delete formData[k])
  Object.assign(formData, seed)
}

function openAdd() {
  editing.value = false
  submitFlow.reset()
  original.value = {} // 新增：基线空 → 填入即“新增”
  resetForm()
  formRef.value?.clearValidate()
  drawerVisible.value = true
}

function openEdit(row: Record<string, any>) {
  editing.value = true
  submitFlow.reset()
  original.value = { ...row } // 编辑：基线 = 已回填实际态
  resetForm({ ...row })
  drawerVisible.value = true
}

async function submit() {
  if (!selectedDevice.value) return
  // 表单校验不通过则不提交（§9：不提交、行内提示 YANG 约束）
  if (formRef.value) {
    try {
      await formRef.value.validate()
    } catch {
      return
    }
  }
  // 下发 → 回读 → 轮询对账（真实进度由 ReconcileSteps 展示）
  await submitFlow.run(selectedDevice.value, { ...formData })
  // 下发成功（非 setConfig 失败）则重读列表，反映最新配置
  if (submitFlow.phase.value !== 'error') await cfg.loadItems(selectedDevice.value)
}

// 抽屉关闭后复位编排态，下次打开回到表单
function onDrawerClosed() {
  submitFlow.reset()
}

function reload() {
  if (selectedDevice.value) cfg.loadItems(selectedDevice.value)
}

onMounted(async () => {
  await store.fetchDevices()
  try {
    await cfg.loadSchema()
  } catch {
    /* schema 拉取失败不阻断页面 */
  }
})
</script>

<style scoped>
.device-config {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.page-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
}

.header-actions {
  display: flex;
  gap: 12px;
}

/* 左侧 YANG 架构树 + 右侧配置表格（对齐设计原型 .cfg 双栏） */
.cfg {
  display: grid;
  grid-template-columns: 288px 1fr;
  gap: 16px;
  align-items: start;
}

.cfg-tree {
  position: sticky;
  top: 16px;
}

.config-table {
  background: #fff;
  border-radius: 8px;
}

@media (max-width: 900px) {
  .cfg {
    grid-template-columns: 1fr;
  }
}

.config-form {
  padding: 0 4px;
}

.form-tip {
  margin-top: 14px;
  font-size: 11.5px;
  line-height: 1.6;
  color: var(--ink-3, #93a2b1);
}
</style>
