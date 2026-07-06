<template>
  <div class="reconcile-steps">
    <div class="section-lbl">对账进行中 · Reconciler</div>
    <div class="recon-steps">
      <template v-for="(s, i) in progress.steps" :key="s.key">
        <div class="rstep" :class="s.state">
          <div class="ico">
            <svg v-if="s.state === 'done'" viewBox="0 0 24 24" aria-hidden="true"><path d="M4 12l5 5L20 6" /></svg>
            <svg v-else-if="s.state === 'error'" viewBox="0 0 24 24" aria-hidden="true">
              <path d="M6 6l12 12M18 6L6 18" />
            </svg>
          </div>
          <div class="rstep-txt">
            <b>{{ s.title }}</b>
            <span>{{ s.sub }}</span>
          </div>
        </div>
        <div v-if="i < progress.steps.length - 1" class="rline" :class="{ done: progress.steps[i].state === 'done' }" />
      </template>
    </div>

    <div v-if="resultChip" class="recon-result" :class="resultChip.cls">
      <span class="glyph" />{{ resultChip.label }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ReconcileProgress } from '../../utils/reconcileProgress'

const props = defineProps<{ progress: ReconcileProgress; timedOut?: boolean }>()

// 终局徽标：收敛/漂移/失败；超时（未拿到终态）诚实标注「仍在对账」而非成功。
const resultChip = computed(() => {
  if (props.timedOut) return { cls: 'recon', label: '对账仍在进行 · 可去概览大盘继续观察' }
  switch (props.progress.outcome) {
    case 'converged':
      return { cls: 'conv', label: '已收敛 · 期望态与实际态一致' }
    case 'drifted':
      return { cls: 'drift', label: '已漂移 · 回读发现差异（reconciler 将持续纠偏）' }
    case 'error':
      return { cls: 'error', label: '下发失败 · 保留原配置' }
    default:
      return null
  }
})
</script>

<style scoped>
.section-lbl {
  font-size: 11px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--ink-3, #93a2b1);
  font-weight: 600;
  margin: 4px 0 12px;
  display: flex;
  align-items: center;
  gap: 10px;
}

.section-lbl::after {
  content: '';
  flex: 1;
  height: 1px;
  background: var(--line, #e6ebf0);
}

.recon-steps {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-top: 6px;
}

.rstep {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  padding: 9px 0;
}

.rstep .ico {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  border: 2px solid var(--line, #e6ebf0);
  position: relative;
}

.rstep .ico svg {
  width: 12px;
  height: 12px;
  stroke: #fff;
  stroke-width: 2.4;
  fill: none;
}

.rstep.done .ico {
  background: var(--st-conv, #2f8a4c);
  border-color: var(--st-conv, #2f8a4c);
}

.rstep.active .ico {
  border-color: var(--st-recon, #2a6fd6);
  background: var(--surface, #fff);
}

.rstep.active .ico::after {
  content: '';
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--st-recon, #2a6fd6);
  animation: blink 1s infinite;
}

.rstep.error .ico {
  background: var(--st-off, #c0392b);
  border-color: var(--st-off, #c0392b);
}

.rstep.wait .ico {
  background: var(--surface, #fff);
}

.rstep-txt b {
  font-size: 13px;
  font-weight: 600;
  display: block;
  color: var(--ink, #1f2d3d);
}

.rstep-txt span {
  font-size: 11.5px;
  color: var(--ink-3, #93a2b1);
  font-family: var(--f-mono, monospace);
}

.rstep.wait .rstep-txt b {
  color: var(--ink-3, #93a2b1);
}

.rline {
  width: 2px;
  background: var(--line, #e6ebf0);
  margin-left: 10px;
  height: 10px;
}

.rline.done {
  background: var(--st-conv, #2f8a4c);
}

.recon-result {
  margin-top: 20px;
  padding: 13px 15px;
  border-radius: var(--r-ctl, 8px);
  font-size: 13px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 9px;
}

.recon-result .glyph {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
  flex-shrink: 0;
}

.recon-result.conv {
  background: var(--st-conv-bg, #e4f2e8);
  color: var(--st-conv, #2f8a4c);
}

.recon-result.drift {
  background: var(--st-drift-bg, #faeeda);
  color: var(--st-drift, #b26a00);
}

.recon-result.error {
  background: var(--st-off-bg, #fbe6e3);
  color: var(--st-off, #c0392b);
}

.recon-result.recon {
  background: var(--st-recon-bg, #e6effb);
  color: var(--st-recon, #2a6fd6);
}

@keyframes blink {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.3;
  }
}
</style>
