<template>
  <div class="dashboard">
    <!-- 页头 -->
    <div class="ph">
      <div>
        <h1>车队概览</h1>
        <div class="sub">
          {{ o.total }} 台设备 · 声明式对账实时同步 · 数据直读自 NETCONF / gNMI，不落库
        </div>
      </div>
      <div class="ph-actions">
        <el-button :loading="loading" @click="load">刷新</el-button>
        <el-button type="primary" @click="goConfig">下发配置</el-button>
      </div>
    </div>

    <el-alert
      v-if="error"
      class="load-error"
      type="error"
      :closable="false"
      :title="`概览加载失败：${error}`"
    />

    <!-- hero + 统计栈 -->
    <div class="grid-hero">
      <ConvergenceHero :overview="o" />

      <div class="stat-stack">
        <div class="card stat">
          <div class="stat-k">在线设备</div>
          <div class="stat-v mono">{{ o.online }}<span class="u">/ {{ o.total }}</span></div>
        </div>
        <div class="card stat">
          <div class="stat-k">待对账变更</div>
          <div class="stat-v mono">
            {{ o.pendingCount }}
            <span v-if="o.pendingCount > 0" class="trend warn">需处理</span>
            <span v-else class="trend up">全部收敛</span>
          </div>
        </div>
        <div class="card stat">
          <div class="stat-k">未对账设备</div>
          <div class="stat-v mono">{{ o.unknownCount }}<span class="u">台 · 无 desired 态</span></div>
        </div>
      </div>
    </div>

    <!-- 台账 + 最近对账 -->
    <div class="grid-2">
      <div class="card">
        <div class="card-h">
          <h3>待对账设备 · 期望态 ↔ 实际态</h3>
          <span class="meta">Reconciler 周期轮询</span>
        </div>
        <div class="wrap-tbl">
          <table class="tbl">
            <thead>
              <tr><th>设备</th><th>结局</th><th>最近对账</th></tr>
            </thead>
            <tbody>
              <tr v-for="row in o.ledger" :key="row.ip">
                <td><div class="strong mono">{{ row.ip }}</div></td>
                <td><ReconcileChip :state="row.state" /></td>
                <td class="mono muted">{{ formatTime(row.lastRun) }}</td>
              </tr>
            </tbody>
          </table>
          <div v-if="o.ledger.length === 0" class="empty">
            {{ o.total === 0 ? '暂无设备' : '全部设备已收敛 · 无待对账变更' }}
          </div>
        </div>
      </div>

      <div class="card">
        <div class="card-h">
          <h3>最近对账</h3>
          <button class="link" @click="goLogs">查看全部</button>
        </div>
        <div class="wrap-tbl">
          <table class="tbl">
            <thead>
              <tr><th>设备</th><th>结局</th><th>时刻</th></tr>
            </thead>
            <tbody>
              <tr v-for="row in recentTop" :key="row.ip">
                <td><div class="strong mono">{{ row.ip }}</div></td>
                <td><ReconcileChip :state="row.state" /></td>
                <td class="mono muted">{{ formatTime(row.lastRun) }}</td>
              </tr>
            </tbody>
          </table>
          <div v-if="recentTop.length === 0" class="empty">暂无对账记录</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useFleetOverview } from '../composables/useFleetOverview'
import ConvergenceHero from '../components/dashboard/ConvergenceHero.vue'
import ReconcileChip from '../components/dashboard/ReconcileChip.vue'

const router = useRouter()
const { overview, loading, error, load } = useFleetOverview()

const o = overview
const recentTop = computed(() => o.value.recent.slice(0, 6))

function formatTime(iso: string | null): string {
  if (!iso) return '—'
  const t = Date.parse(iso)
  if (Number.isNaN(t)) return '—'
  return new Date(t).toLocaleString('zh-CN', { hour12: false })
}

function goConfig() {
  router.push('/config/vlan')
}
function goLogs() {
  router.push('/logs')
}

onMounted(load)
</script>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.ph {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}
.ph h1 {
  font-size: 22px;
  font-weight: 700;
  letter-spacing: -0.01em;
  color: var(--ink);
}
.ph .sub {
  color: var(--ink-2);
  font-size: 13px;
  margin-top: 3px;
}
.ph-actions {
  display: flex;
  gap: 9px;
}
.load-error {
  border-radius: var(--r-ctl);
}

.card {
  background: var(--surface);
  border: 1px solid var(--line);
  border-radius: var(--r-card);
  box-shadow: var(--sh-1);
}
.card-h {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 15px 18px;
  border-bottom: 1px solid var(--line);
}
.card-h h3 {
  font-size: 14.5px;
  font-weight: 600;
  color: var(--ink);
}
.card-h .meta {
  font-size: 12px;
  color: var(--ink-3);
}
.link {
  color: var(--primary);
  font-weight: 600;
  cursor: pointer;
  font-size: 13px;
  background: none;
  border: none;
  font-family: inherit;
}
.link:hover {
  color: var(--primary-ink);
  text-decoration: underline;
}

.grid-hero {
  display: grid;
  grid-template-columns: 1.55fr 1fr;
  gap: 16px;
}

.stat-stack {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.stat {
  flex: 1;
  padding: 15px 17px;
  display: flex;
  flex-direction: column;
  justify-content: center;
}
.stat-k {
  font-size: 12px;
  color: var(--ink-2);
}
.stat-v {
  font-size: 27px;
  font-weight: 600;
  letter-spacing: -0.02em;
  margin-top: 2px;
  display: flex;
  align-items: baseline;
  gap: 8px;
  color: var(--ink);
}
.stat-v .u {
  font-size: 13px;
  color: var(--ink-3);
  font-weight: 500;
}
.trend {
  font-size: 11.5px;
  font-weight: 600;
}
.trend.up { color: var(--st-conv); }
.trend.warn { color: var(--st-drift); }

.grid-2 {
  display: grid;
  grid-template-columns: 1.4fr 1fr;
  gap: 16px;
}
.wrap-tbl { overflow-x: auto; }
.tbl {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.tbl thead th {
  text-align: left;
  font-size: 11px;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--ink-3);
  font-weight: 600;
  padding: 10px 14px;
  border-bottom: 1px solid var(--line);
  background: var(--sunken);
}
.tbl tbody td {
  padding: 11px 14px;
  border-bottom: 1px solid var(--line);
  color: var(--ink-2);
  vertical-align: middle;
}
.tbl tbody tr:last-child td { border-bottom: none; }
.tbl tbody tr:hover { background: var(--sunken); }
.tbl .strong { color: var(--ink); font-weight: 600; }
.tbl .muted { color: var(--ink-3); font-size: 12px; }

.empty {
  text-align: center;
  padding: 32px;
  color: var(--ink-3);
  font-size: 13px;
}

@media (max-width: 1080px) {
  .grid-hero,
  .grid-2 {
    grid-template-columns: 1fr;
  }
}
</style>
