<template>
  <div class="xml-viewer" data-test="xml-viewer">
    <div v-if="!lines.length" class="xml-empty">{{ t('console.batch.noPayload') }}</div>
    <ol v-else class="xml-lines">
      <li v-for="(line, i) in lines" :key="i" class="xml-line">
        <span class="ln" aria-hidden="true">{{ i + 1 }}</span>
        <code class="code" :style="{ paddingLeft: line.indent * 1.25 + 'em' }">
          <span v-for="(tk, j) in line.tokens" :key="j" :class="'tk-' + tk.kind">{{ tk.text }}</span>
        </code>
      </li>
    </ol>
  </div>
</template>

<script setup lang="ts">
// NETCONF 报文查看器（FE-23）：带行号的缩进着色视图。格式化纯属**展示层**，
// 报文内容零改动（试运行的意义就在于「所见即将发」）。token 经 v-for 渲染而
// 非 v-html——报文含设备侧任意文本，绝不拼 HTML（XSS 免疫）。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatXml } from '../../utils/xmlFormat'

const props = defineProps<{ xml?: string }>()
const { t } = useI18n()

const lines = computed(() => formatXml(props.xml ?? ''))
</script>

<style scoped>
.xml-viewer {
  font-family: var(--f-mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
  font-size: 12px;
  line-height: 1.7;
  max-height: 320px;
  overflow: auto;
  background: var(--el-fill-color-blank, #fff);
}

.xml-lines {
  margin: 0;
  padding: 6px 0;
  list-style: none;
}

.xml-line {
  display: flex;
  align-items: flex-start;
  white-space: pre;
}

.xml-line:hover {
  background: var(--el-fill-color-light);
}

/* 行号栏：不可选中（复制报文时不带行号） */
.ln {
  flex: 0 0 auto;
  min-width: 2.6em;
  padding: 0 10px 0 6px;
  text-align: right;
  color: var(--el-text-color-placeholder);
  user-select: none;
}

.code {
  flex: 1 1 auto;
  white-space: pre;
  font: inherit;
}

.xml-empty {
  padding: 12px;
  color: var(--el-text-color-secondary);
}

/* 着色：标签名/属性名/属性值/文本各一色（对齐 NCE 报文视图观感） */
.tk-tag {
  color: var(--el-color-primary);
}
.tk-attr-name {
  color: var(--el-color-danger);
}
.tk-attr-value {
  color: var(--el-color-success);
}
.tk-punct {
  color: var(--el-text-color-secondary);
}
.tk-comment {
  color: var(--el-text-color-placeholder);
  font-style: italic;
}
.tk-text {
  color: var(--el-text-color-primary);
}
</style>
