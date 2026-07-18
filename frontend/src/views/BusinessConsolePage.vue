<template>
  <div class="business-console">
    <el-breadcrumb separator="/">
      <el-breadcrumb-item>{{ t('nav.businessConfig') }}</el-breadcrumb-item>
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
        <el-button type="primary" data-test="business-create" @click="openCreate">{{ t('business.create') }}</el-button>
        <span class="tip">{{ t('business.tip') }}</span>
      </div>

      <!-- 实例列表：平台作用域——每行一个意图实例，收敛状态由 status 聚合。 -->
      <el-table :data="items" v-loading="loading" data-test="business-table">
        <el-table-column prop="name" :label="t('business.colInstance')" min-width="140" />
        <el-table-column label="VLAN" width="90">
          <template #default="{ row }">{{ row.spec?.['vlan-id'] ?? '-' }}</template>
        </el-table-column>
        <el-table-column :label="t('business.colDeviceCount')" width="90">
          <template #default="{ row }">{{ (row.spec?.devices || []).length }}</template>
        </el-table-column>
        <el-table-column :label="t('business.colConverge')" min-width="160">
          <template #default="{ row }">
            <el-tag :type="convergeTagType(row)" size="small" :data-test="`converge-${row.name}`">
              {{ convergeText(row) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('common.actions')" width="200">
          <template #default="{ row }">
            <el-button size="small" text type="primary" @click="openDetail(row)">{{ t('business.detail') }}</el-button>
            <el-button size="small" text type="primary" :data-test="`business-edit-${row.name}`" @click="openEdit(row)">{{ t('common.edit') }}</el-button>
            <el-button size="small" text type="danger" :data-test="`business-remove-${row.name}`" @click="remove(row)">{{ t('common.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 新建/编辑抽屉：表单由意图 YANG schema 自动渲染（R05；devices 为嵌套 list）。 -->
      <el-drawer v-model="drawerOpen" :title="editingName ? t('business.editTitle', { name: editingName }) : t('business.create')" size="560px">
        <el-form ref="formRef" :model="form.formData" :rules="form.rules.value" label-position="top">
          <el-form-item :label="t('business.nameLabel')" prop="__name" :error="nameError" required>
            <el-input
              v-model="instanceName"
              :disabled="!!editingName"
              :placeholder="t('business.namePlaceholder')"
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
          <el-button @click="drawerOpen = false">{{ t('common.cancel') }}</el-button>
          <el-button
            type="primary"
            :disabled="!form.submittable.value || !instanceName"
            data-test="business-submit"
            @click="submit"
          >{{ t('common.submit') }}</el-button>
        </div>
      </el-drawer>

      <!-- 详情抽屉：每设备收敛状态与失败原因（BIC-04 deviceStates）。 -->
      <el-drawer v-model="detailOpen" :title="t('business.detailTitle', { name: detail?.name || '' })" size="480px">
        <template v-if="detail">
          <h4>{{ t('business.perDeviceState') }}</h4>
          <el-table :data="deviceStates(detail)" size="small" data-test="device-states">
            <el-table-column prop="device" :label="t('common.device')" width="130" />
            <el-table-column :label="t('common.status')" width="90">
              <template #default="{ row }">
                <el-tag :type="phaseTag(row.phase)" size="small">{{ row.phase }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="reason" :label="t('business.colReason')" min-width="160" show-overflow-tooltip />
          </el-table>
          <h4>{{ t('business.claimedNative') }}</h4>
          <el-table :data="claims(detail)" size="small">
            <el-table-column prop="device" :label="t('common.device')" width="130" />
            <el-table-column prop="path" :label="t('business.colPath')" min-width="200" show-overflow-tooltip />
          </el-table>
        </template>
      </el-drawer>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
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

const { t } = useI18n()
const pageTitle = ref(t('business.defaultTitle'))
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
    if (payload?.title) pageTitle.value = payload.description || t('business.defaultTitle')
    if (!schemaFields.value.length) schemaError.value = t('business.noRenderableFields', { module: MODULE })
  } catch (e: any) {
    schemaError.value = e?.response?.data?.message || e?.message || t('business.loadModelFailed')
  }
}

async function loadList() {
  loading.value = true
  listError.value = ''
  try {
    const res = await listBusinessVlanServices()
    if (res.data?.success === false) {
      // 后端信封错误（HTTP 恒 200）：如未连接集群的 503 降级。
      listError.value = res.data?.message || t('business.unavailable')
      items.value = []
      return
    }
    items.value = res.data?.data?.items || []
  } catch (e: any) {
    listError.value = e?.response?.data?.message || e?.message || t('business.listFailed')
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
    ElMessage.success(t('business.submitted'))
    drawerOpen.value = false
    await loadList()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || t('business.submitFailed'))
  }
}

async function remove(row: BusinessVlanServiceItem) {
  try {
    await ElMessageBox.confirm(
      t('business.removeConfirm', { name: row.name }),
      t('common.confirmDelete'),
      { type: 'warning' },
    )
  } catch {
    return
  }
  try {
    await deleteBusinessVlanService(row.name)
    ElMessage.success(t('business.removeAccepted'))
    await loadList()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || t('business.removeFailed'))
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

// 收敛状态按语义分档（tag 类型据档位而非文案判断，避免与 i18n 文案耦合）。
type ConvergeKind = 'validationFailed' | 'pending' | 'converged' | 'partialFailed' | 'converging'

function convergeKind(row: BusinessVlanServiceItem): ConvergeKind {
  const v = condition(row, 'Validated')
  if (v && v.status === 'False') return 'validationFailed'
  const c = condition(row, 'Converged')
  if (!c) return 'pending'
  if (c.status === 'True') return 'converged'
  const states = deviceStates(row)
  const failed = states.filter((s: any) => s.phase === 'failed').length
  if (failed > 0) return 'partialFailed'
  return 'converging'
}

function convergeText(row: BusinessVlanServiceItem): string {
  const kind = convergeKind(row)
  switch (kind) {
    case 'validationFailed':
      return t('business.stateValidationFailed')
    case 'pending':
      return t('business.statePending')
    case 'converged':
      return t('common.state.conv')
    case 'partialFailed': {
      const states = deviceStates(row)
      const failed = states.filter((s: any) => s.phase === 'failed').length
      return t('business.statePartialFailed', { ok: states.length - failed, total: states.length })
    }
    default:
      return t('common.state.recon')
  }
}

function convergeTagType(row: BusinessVlanServiceItem): string {
  const kind = convergeKind(row)
  if (kind === 'converged') return 'success'
  if (kind === 'validationFailed' || kind === 'partialFailed') return 'danger'
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
