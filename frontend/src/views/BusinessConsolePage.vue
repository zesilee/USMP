<template>
  <div class="business-console">
    <el-breadcrumb separator="/">
      <el-breadcrumb-item>业务网络配置</el-breadcrumb-item>
      <el-breadcrumb-item>{{ pageTitle }}</el-breadcrumb-item>
    </el-breadcrumb>

    <el-alert v-if="schemaError" :title="schemaError" type="warning" :closable="false" show-icon />
    <el-alert
      v-else-if="listError"
      :title="listError"
      type="warning"
      :closable="false"
      show-icon
      data-test="business-unavailable"
    />

    <template v-if="!schemaError">
      <div class="toolbar">
        <el-button type="primary" data-test="business-create" @click="openCreate">新建业务实例</el-button>
        <span class="tip">意图实例由 YANG 模型驱动；提交后编排为各设备原生配置（跨设备事务下发）。</span>
      </div>

      <!-- 实例列表：平台作用域——每行一个意图实例，收敛状态由 status 聚合。 -->
      <el-table :data="items" v-loading="loading" data-test="business-table">
        <el-table-column prop="name" label="实例名" min-width="140" />
        <el-table-column label="VLAN" width="90">
          <template #default="{ row }">{{ row.spec?.['vlan-id'] ?? '-' }}</template>
        </el-table-column>
        <el-table-column label="设备数" width="90">
          <template #default="{ row }">{{ (row.spec?.devices || []).length }}</template>
        </el-table-column>
        <el-table-column label="收敛状态" min-width="160">
          <template #default="{ row }">
            <el-tag :type="convergeTagType(row)" size="small" :data-test="`converge-${row.name}`">
              {{ convergeText(row) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <el-button size="small" text type="primary" @click="openDetail(row)">详情</el-button>
            <el-button size="small" text type="primary" :data-test="`business-edit-${row.name}`" @click="openEdit(row)">编辑</el-button>
            <el-button size="small" text type="danger" :data-test="`business-remove-${row.name}`" @click="remove(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 新建/编辑抽屉：表单由意图 YANG schema 自动渲染（R05；devices 为嵌套 list）。 -->
      <el-drawer v-model="drawerOpen" :title="editingName ? `编辑 ${editingName}` : '新建业务实例'" size="560px">
        <el-form ref="formRef" :model="form.formData" :rules="form.rules.value" label-position="top">
          <el-form-item label="实例名" prop="__name" :error="nameError" required>
            <el-input
              v-model="instanceName"
              :disabled="!!editingName"
              placeholder="如 biz-vlan-100（K8s 资源名规则）"
              data-test="business-name-input"
            />
          </el-form-item>
          <el-form-item
            v-for="field in form.visibleFields.value"
            :key="field.path"
            :label="field.label || form.keyOf(field)"
            :prop="form.keyOf(field)"
          >
            <FieldRenderer
              :field="field"
              :model-value="form.formData[form.keyOf(field)]"
              @update:model-value="form.formData[form.keyOf(field)] = $event"
            />
          </el-form-item>
        </el-form>
        <div class="drawer-actions">
          <el-button @click="drawerOpen = false">取消</el-button>
          <el-button
            type="primary"
            :disabled="!form.submittable.value || !instanceName"
            data-test="business-submit"
            @click="submit"
          >提交</el-button>
        </div>
      </el-drawer>

      <!-- 详情抽屉：每设备收敛状态与失败原因（BIC-04 deviceStates）。 -->
      <el-drawer v-model="detailOpen" :title="`实例详情：${detail?.name || ''}`" size="480px">
        <template v-if="detail">
          <h4>每设备状态</h4>
          <el-table :data="deviceStates(detail)" size="small" data-test="device-states">
            <el-table-column prop="device" label="设备" width="130" />
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag :type="phaseTag(row.phase)" size="small">{{ row.phase }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="reason" label="原因" min-width="160" show-overflow-tooltip />
          </el-table>
          <h4>认领的原生配置</h4>
          <el-table :data="claims(detail)" size="small">
            <el-table-column prop="device" label="设备" width="130" />
            <el-table-column prop="path" label="路径" min-width="200" show-overflow-tooltip />
          </el-table>
        </template>
      </el-drawer>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import {
  getYangSchema,
  listBusinessVlanServices,
  applyBusinessVlanService,
  deleteBusinessVlanService,
  type BusinessVlanServiceItem,
} from '../api'
import { useConfigForm } from '../composables/useConfigForm'
import type { Field } from '../utils/crdSchemaParser'
import FieldRenderer from '../components/config/FieldRenderer.vue'

// 平台作用域业务控制台（FE-17）：与设备作用域模块控制台并列——无设备选择器，
// 一个意图实例管理 spec.devices 里的 N 台设备。
const MODULE = 'business-vlan-service'

const pageTitle = ref('跨设备 VLAN 打通')
const schemaError = ref('')
const listError = ref('')
const loading = ref(false)
const items = ref<BusinessVlanServiceItem[]>([])

const schemaFields = ref<Field[]>([])
const form = useConfigForm(computed(() => schemaFields.value))
const formRef = ref<FormInstance>()

const drawerOpen = ref(false)
const detailOpen = ref(false)
const detail = ref<BusinessVlanServiceItem | null>(null)
const instanceName = ref('')
const editingName = ref('')
const nameError = ref('')

async function loadSchema() {
  try {
    const res = await getYangSchema(MODULE, 'nested')
    const payload = res.data?.data
    schemaFields.value = payload?.fields || []
    if (payload?.title) pageTitle.value = payload.description || '跨设备 VLAN 打通'
    if (!schemaFields.value.length) schemaError.value = `模块 ${MODULE} 无可渲染字段`
  } catch (e: any) {
    schemaError.value = e?.response?.data?.message || e?.message || '加载业务模型失败'
  }
}

async function loadList() {
  loading.value = true
  listError.value = ''
  try {
    const res = await listBusinessVlanServices()
    if (res.data?.success === false) {
      // 后端信封错误（HTTP 恒 200）：如未连接集群的 503 降级。
      listError.value = res.data?.message || '业务配置暂不可用'
      items.value = []
      return
    }
    items.value = res.data?.data?.items || []
  } catch (e: any) {
    listError.value = e?.response?.data?.message || e?.message || '读取业务实例失败'
    items.value = []
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingName.value = ''
  instanceName.value = ''
  nameError.value = ''
  form.resetForm()
  drawerOpen.value = true
}

function openEdit(row: BusinessVlanServiceItem) {
  editingName.value = row.name
  instanceName.value = row.name
  nameError.value = ''
  form.resetForm(row.spec || {})
  drawerOpen.value = true
}

function openDetail(row: BusinessVlanServiceItem) {
  detail.value = row
  detailOpen.value = true
}

async function submit() {
  if (formRef.value) {
    try {
      await formRef.value.validate()
    } catch {
      /* 行内提示；权威门禁在 form.blocked */
    }
  }
  if (form.blocked.value || !instanceName.value) return
  try {
    await applyBusinessVlanService(instanceName.value, form.visiblePayload())
    ElMessage.success('已提交，编排收敛中（状态见列表）')
    drawerOpen.value = false
    await loadList()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '提交失败')
  }
}

async function remove(row: BusinessVlanServiceItem) {
  try {
    await ElMessageBox.confirm(
      `删除业务实例 ${row.name}？各设备上认领的原生配置将被清理。`,
      '确认删除',
      { type: 'warning' },
    )
  } catch {
    return
  }
  try {
    await deleteBusinessVlanService(row.name)
    ElMessage.success('删除已受理（设备清理完成后实例消失）')
    await loadList()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '删除失败')
  }
}

// ===== status 聚合呈现（BIC-04）=====
function condition(row: BusinessVlanServiceItem, type: string): any {
  return (row.status?.conditions || []).find((c: any) => c?.type === type)
}

function deviceStates(row: BusinessVlanServiceItem): any[] {
  return row.status?.deviceStates || []
}

function claims(row: BusinessVlanServiceItem): any[] {
  return row.status?.claims || []
}

function convergeText(row: BusinessVlanServiceItem): string {
  const v = condition(row, 'Validated')
  if (v && v.status === 'False') return '校验失败'
  const c = condition(row, 'Converged')
  if (!c) return '待处理'
  if (c.status === 'True') return '已收敛'
  const states = deviceStates(row)
  const failed = states.filter((s: any) => s.phase === 'failed').length
  if (failed > 0) return `部分失败 ${states.length - failed}/${states.length}`
  return '收敛中'
}

function convergeTagType(row: BusinessVlanServiceItem): string {
  const text = convergeText(row)
  if (text === '已收敛') return 'success'
  if (text === '校验失败' || text.startsWith('部分失败')) return 'danger'
  return 'info'
}

function phaseTag(phase: string): string {
  if (phase === 'synced') return 'success'
  if (phase === 'failed') return 'danger'
  return 'info'
}

onMounted(async () => {
  await Promise.all([loadSchema(), loadList()])
})
</script>

<style scoped>
.business-console {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 14px;
}

.tip {
  font-size: 11.5px;
  color: var(--ink-3, #93a2b1);
}

.drawer-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 16px;
}

h4 {
  margin: 12px 0 8px;
}
</style>
