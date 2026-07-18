<template>
  <div class="field-renderer">
    <!-- String（units → 单位后缀，FE-15） -->
    <div v-if="field.type === 'string'" class="field-scalar">
      <el-input
        :model-value="modelValue"
        @update:model-value="$emit('update:modelValue', $event)"
        :placeholder="placeholderOf"
        :disabled="field.readonly || disabled"
      />
      <span v-if="field.units" class="field-units">{{ field.units }}</span>
    </div>

    <!-- Number（units → 单位后缀，FE-15） -->
    <div v-else-if="field.type === 'number'" class="field-scalar">
      <el-input-number
        :model-value="modelValue"
        @update:model-value="$emit('update:modelValue', $event)"
        :placeholder="placeholderOf"
        :disabled="field.readonly || disabled"
        :min="field.minimum"
        :max="field.maximum"
        controls-position="right"
        style="width: 100%"
      />
      <span v-if="field.units" class="field-units">{{ field.units }}</span>
    </div>

    <!-- Boolean -->
    <el-switch
      v-else-if="field.type === 'boolean'"
      :model-value="modelValue"
      @update:model-value="$emit('update:modelValue', $event)"
      :disabled="field.readonly || disabled"
    />

    <!-- Enum：必填且 ≤3 选项 → 分段控件（FE-01，NCE 保真）；segmented 无清空能力，
         可选枚举需保留「清空=键不入 payload」语义，故仅必填走此分支。 -->
    <el-segmented
      v-else-if="field.type === 'enum' && enumUsesSegmented"
      :model-value="modelValue"
      @update:model-value="$emit('update:modelValue', $event)"
      :options="segmentedOptions"
      :disabled="field.readonly || disabled"
    />
    <el-select
      v-else-if="field.type === 'enum'"
      :model-value="modelValue"
      @update:model-value="$emit('update:modelValue', $event)"
      :placeholder="placeholderOf"
      :disabled="field.readonly || disabled"
      clearable
      style="width: 100%"
    >
      <el-option
        v-for="option in field.options"
        :key="String(option.value)"
        :label="option.label"
        :value="option.value"
      />
    </el-select>

    <!-- Presence group (YANG presence 容器)：存在即开关。关闭 → emit undefined
         （键不入 payload，节点不存在）；开启 → 保留/新建对象并展开子表单（FE-12）。 -->
    <div v-else-if="field.type === 'group' && field.presence" class="field-presence">
      <el-switch
        :model-value="presenceOn"
        :disabled="field.readonly || disabled"
        @update:model-value="togglePresence(!!$event)"
      />
      <div v-if="presenceOn && childFields.length" class="field-group presence-fields">
        <div class="group-fields">
          <div v-for="subField in childFields" :key="subField.path" class="sub-field">
            <label class="field-label">{{ subField.label }}</label>
            <FieldRenderer
              :field="subField"
              :disabled="disabled"
              :model-value="(modelValue || {})[keyOf(subField)]"
              @update:model-value="updateSubField(keyOf(subField), $event)"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- Group (single nested object) -->
    <div v-else-if="field.type === 'group'" class="field-group">
      <div class="group-fields">
        <div v-for="subField in childFields" :key="subField.path" class="sub-field">
          <label class="field-label">{{ subField.label }}</label>
          <FieldRenderer
            :field="subField"
            :disabled="disabled"
            :model-value="(modelValue || {})[keyOf(subField)]"
            @update:model-value="updateSubField(keyOf(subField), $event)"
          />
        </div>
      </div>
    </div>

    <!-- Leaf-list (repeatable scalar values) -->
    <div v-else-if="field.type === 'leaf-list'" class="field-list">
      <div v-for="(item, idx) in items" :key="idx" class="list-row leaf-list-row">
        <el-select
          v-if="field.options?.length"
          :model-value="item"
          @update:model-value="updateItem(idx, $event)"
          :disabled="field.readonly || disabled"
          clearable
          style="width: 100%"
        >
          <el-option
            v-for="option in field.options"
            :key="String(option.value)"
            :label="option.label"
            :value="option.value"
          />
        </el-select>
        <el-input
          v-else
          :model-value="item"
          @update:model-value="updateItem(idx, $event)"
          :placeholder="field.placeholder"
          :disabled="field.readonly || disabled"
        />
        <el-button type="danger" size="small" link :icon="Delete" @click="removeItem(idx)">{{ t('common.delete') }}</el-button>
      </div>
      <el-button type="primary" size="small" plain :icon="Plus" @click="addItem">
        {{ t('console.addItem', { label: field.label }) }}
      </el-button>
    </div>

    <!-- Choice (mutually-exclusive branches → RadioGroup / Tabs) -->
    <div v-else-if="field.type === 'choice'" class="field-choice">
      <!-- 全单叶 case → RadioGroup（选分支 + 展示激活分支的单个输入） -->
      <template v-if="choiceUsesRadio">
        <el-radio-group :model-value="activeCase" @update:model-value="switchCase(String($event))">
          <el-radio v-for="c in field.cases" :key="c.name" :value="c.name">{{ c.label }}</el-radio>
        </el-radio-group>
        <div v-if="activeCaseFields.length" class="choice-active-fields">
          <div v-for="sub in activeCaseFields" :key="sub.path" class="sub-field">
            <label class="field-label">{{ sub.label }}</label>
            <FieldRenderer
              :field="sub"
              :disabled="disabled"
              :model-value="scope[keyOf(sub)]"
              @update:model-value="updateMember(keyOf(sub), $event)"
            />
          </div>
        </div>
      </template>
      <!-- 任一 case 含多字段/容器 → Tabs -->
      <el-tabs v-else :model-value="activeCase" @update:model-value="switchCase(String($event))">
        <el-tab-pane v-for="c in field.cases" :key="c.name" :label="c.label" :name="c.name">
          <div v-for="sub in (c.fields || [])" :key="sub.path" class="sub-field">
            <label class="field-label">{{ sub.label }}</label>
            <FieldRenderer
              :field="sub"
              :disabled="disabled"
              :model-value="scope[keyOf(sub)]"
              @update:model-value="updateMember(keyOf(sub), $event)"
            />
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>

    <!-- List (repeatable rows of a nested sub-form) -->
    <div v-else-if="field.type === 'list'" class="field-list">
      <div v-for="(row, idx) in rows" :key="idx" class="list-row">
        <div class="list-row-fields">
          <div v-for="subField in childFields" :key="subField.path" class="sub-field">
            <label class="field-label">{{ subField.label }}</label>
            <FieldRenderer
              :field="subField"
              :model-value="(row || {})[keyOf(subField)]"
              @update:model-value="updateRow(idx, keyOf(subField), $event)"
            />
          </div>
        </div>
        <el-button
          type="danger"
          size="small"
          link
          :icon="Delete"
          @click="removeRow(idx)"
        >{{ t('common.delete') }}</el-button>
      </div>
      <el-button type="primary" size="small" plain :icon="Plus" @click="addRow">
        {{ t('console.addItem', { label: field.label }) }}
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Plus, Delete } from '@element-plus/icons-vue'
import type { Field } from '../../utils/crdSchemaParser'

const { t } = useI18n()

const props = defineProps<{
  field: Field
  modelValue: any
  // 外部禁用（如编辑态的 create-only 标识字段、must 未满足的 presence 开关，FE-11/12）。
  disabled?: boolean
}>()

// ===== presence 容器（FE-12）=====
// modelValue 为对象（含空对象 {}）即「存在=开」；undefined/null 即「不存在=关」。
const presenceOn = computed<boolean>(() => props.modelValue != null)

function togglePresence(on: boolean) {
  emit('update:modelValue', on ? { ...(props.modelValue || {}) } : undefined)
}

// ===== choice（YANG choice/case 互斥分支）=====
// 成员数据以扁平叶名为键、与其它字段同级（sibling），因此 modelValue 直接是承载这些
// 成员键的 scope 对象；本组件读写其中的成员键并整体 emit 回去，由父层按成员键 reconcile。
const LEAF_TYPES = ['string', 'number', 'boolean', 'enum']
const scope = computed<Record<string, any>>(() => (props.modelValue as Record<string, any>) || {})

// 用户显式选择的激活 case（未选时由数据推断）。
const activeCaseRef = ref<string | null>(null)

function caseKeys(c: { fields: Field[] }): string[] {
  return (c.fields || []).map(keyOf)
}
function caseHasData(c: { fields: Field[] }): boolean {
  return caseKeys(c).some((k) => {
    const v = scope.value[k]
    return v !== undefined && v !== null && v !== ''
  })
}

// 激活 case：显式选择 > 有数据的 case > 首个 case。
const activeCase = computed<string>(() => {
  const cases = props.field.cases || []
  if (activeCaseRef.value && cases.some((c) => c.name === activeCaseRef.value)) return activeCaseRef.value
  const withData = cases.find(caseHasData)
  return withData?.name || cases[0]?.name || ''
})

// 全部 case 均为「单个叶字段」→ RadioGroup；否则 Tabs（FE-08）。
const choiceUsesRadio = computed<boolean>(() =>
  (props.field.cases || []).every((c) => c.fields?.length === 1 && LEAF_TYPES.includes(c.fields[0].type)),
)

const activeCaseFields = computed<Field[]>(
  () => (props.field.cases || []).find((c) => c.name === activeCase.value)?.fields || [],
)

// 切换 case：记录激活分支并清空其它非激活 case 的成员键（YANG choice 互斥语义），整体 emit。
function switchCase(name: string) {
  activeCaseRef.value = name
  const next: Record<string, any> = { ...scope.value }
  for (const c of props.field.cases || []) {
    if (c.name === name) continue
    for (const k of caseKeys(c)) delete next[k]
  }
  emit('update:modelValue', next)
}

// 编辑激活成员：写入扁平成员键并整体 emit（保留 scope 内其它键）。
function updateMember(key: string, value: any) {
  emit('update:modelValue', { ...scope.value, [key]: value })
}

const emit = defineEmits<{
  'update:modelValue': [value: any]
}>()

// enum 控件细分（FE-01）：必填且 ≤3 选项用分段控件；零选项（异常 schema）降级 select（R08）。
const enumUsesSegmented = computed<boolean>(() => {
  const n = props.field.options?.length ?? 0
  return !!props.field.required && n > 0 && n <= 3
})

const segmentedOptions = computed(() =>
  (props.field.options || []).map((o) => ({ label: o.label, value: o.value })),
)

// dynamicDefault（FE-15）：空值=系统自动分配，占位提示之；显式 placeholder 优先。
const placeholderOf = computed<string | undefined>(() =>
  props.field.placeholder || (props.field.dynamicDefault ? t('console.autoAssigned') : undefined),
)

// 数据以 YANG 叶子名（path 末段）为键，对齐后端转换（非 full path）。
function keyOf(f: Field): string {
  const seg = f.path.split('/').filter(Boolean).pop()
  return seg || f.path
}

// 子字段：group/list 的直接子项。read-only 的 oper 叶子（如 member-port/state）不参与配置。
const childFields = computed<Field[]>(() =>
  (props.field.fields || []).filter(f => !f.readonly)
)

const rows = computed<Record<string, any>[]>(() =>
  Array.isArray(props.modelValue) ? props.modelValue : []
)

function updateSubField(key: string, value: any) {
  const current = (props.modelValue as Record<string, any>) || {}
  emit('update:modelValue', { ...current, [key]: value })
}

function updateRow(idx: number, key: string, value: any) {
  const next = rows.value.map((r, i) => (i === idx ? { ...r, [key]: value } : r))
  emit('update:modelValue', next)
}

function addRow() {
  emit('update:modelValue', [...rows.value, {}])
}

function removeRow(idx: number) {
  emit('update:modelValue', rows.value.filter((_, i) => i !== idx))
}

// leaf-list：可增删的标量数组（元素为字符串/数字/枚举值）。
const items = computed<any[]>(() => (Array.isArray(props.modelValue) ? props.modelValue : []))

function updateItem(idx: number, value: any) {
  emit('update:modelValue', items.value.map((v, i) => (i === idx ? value : v)))
}

function addItem() {
  emit('update:modelValue', [...items.value, ''])
}

function removeItem(idx: number) {
  emit('update:modelValue', items.value.filter((_, i) => i !== idx))
}
</script>

<style scoped>
.field-renderer {
  width: 100%;
}

.field-scalar {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.field-scalar > :first-child {
  flex: 1;
}

.field-units {
  flex: none;
  font-size: 12px;
  color: var(--text-tertiary);
}

.field-presence {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
}

.field-group {
  width: 100%;
  padding: 12px;
  background-color: var(--bg-elevated);
  border-radius: 8px;
  border: 1px solid var(--border-color);
}

.group-fields {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.sub-field {
  display: flex;
  align-items: center;
  gap: 12px;
}

.field-label {
  min-width: 96px;
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0;
}

.field-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
}

.list-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px;
  background-color: var(--bg-elevated);
  border-radius: 8px;
  border: 1px solid var(--border-color);
}

.list-row-fields {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
</style>
