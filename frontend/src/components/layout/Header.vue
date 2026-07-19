<template>
  <div class="header">
    <div class="header-left">
      <nav class="crumb" :aria-label="t('header.breadcrumbAria')">
        <span class="crumb-root">{{ t('header.fleet') }}</span>
        <span class="crumb-sep">/</span>
        <b>{{ crumbLabel }}</b>
      </nav>

      <div class="field-search">
        <el-input
          v-model="searchText"
          :placeholder="t('header.searchPlaceholder')"
          size="small"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>
    </div>

    <div class="header-right">
      <div class="device-status" :title="t('header.onlineTitle', { online: onlineCount, total: totalCount })">
        <span class="status-dot online"></span>
        <span>{{ t('header.onlineCount', { online: onlineCount, total: totalCount }) }}</span>
      </div>

      <FreshnessRing
        :age-seconds="ageSeconds"
        :ttl-seconds="ttlSeconds"
        :has-data="hasData"
        :source="source"
      />

      <!-- 语言切换（UI-01）：中文/English，偏好经 locale store 持久化 -->
      <el-dropdown trigger="click" :teleported="false" data-test="locale-switch" class="locale-switch">
        <button type="button" class="locale-btn" :aria-label="t('common.language')">
          <svg class="locale-ico" viewBox="0 0 24 24" aria-hidden="true">
            <circle cx="12" cy="12" r="9" />
            <path d="M3 12h18M12 3c2.5 2.6 3.8 5.7 3.8 9S14.5 18.4 12 21c-2.5-2.6-3.8-5.7-3.8-9S9.5 5.6 12 3z" />
          </svg>
          <span class="locale-label">{{ currentLocaleLabel }}</span>
        </button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item data-test="locale-zh" @click="switchLocale('zh-cn')">
              {{ t('header.localeZh') }}
            </el-dropdown-item>
            <el-dropdown-item data-test="locale-en" @click="switchLocale('en-us')">
              {{ t('header.localeEn') }}
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>

      <el-badge :value="notificationCount" :hidden="notificationCount === 0" class="notification-icon">
        <el-button circle size="small"><el-icon><Bell /></el-icon></el-button>
      </el-badge>

      <div class="user-area">
        <div class="avatar">A</div>
        <div class="user-meta">
          <b>admin</b>
          <span>{{ t('header.userRole') }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Bell, Search } from '@element-plus/icons-vue'
import FreshnessRing from './FreshnessRing.vue'
import { useLiveFreshness } from '../../composables/useFreshness'
import { useLocaleStore, type AppLocale } from '../../stores/locale'

// 顶栏：面包屑（随路由）+ 搜索 + 设备在线（真数据待接）+ 缓存新鲜度环（真数据）+ 语言切换 + 告警 + 用户。
const route = useRoute()
const { t } = useI18n()
const localeStore = useLocaleStore()

// 语言切换：当前语言标签 + 落 store（i18n locale + localStorage 持久化）。
const currentLocaleLabel = computed(() =>
  localeStore.locale === 'en-us' ? t('header.localeEn') : t('header.localeZh'),
)
function switchLocale(next: AppLocale) {
  localeStore.setLocale(next)
}

// 路由名 → 面包屑标签 key。缺省回退路径末段，避免出现空标题。
const CRUMB_KEYS: Record<string, string> = {
  dashboard: 'header.crumb.dashboard',
  devices: 'header.crumb.devices',
  interface: 'header.crumb.interface',
  vlan: 'header.crumb.vlan',
  route: 'header.crumb.route',
  native: 'header.crumb.native',
  logs: 'header.crumb.logs',
  settings: 'header.crumb.settings',
}
const crumbLabel = computed(() => {
  const name = route.name as string | undefined
  if (name && CRUMB_KEYS[name]) return t(CRUMB_KEYS[name])
  const seg = route.path.split('/').filter(Boolean).pop()
  return seg ?? t('header.crumb.fallback')
})

// 缓存新鲜度环：消费 store 真数据 + 本地每秒时钟。无活跃缓存时环显示空态。
const { ageSeconds, ttlSeconds, hasData, source } = useLiveFreshness()

const searchText = ref('')
// TODO(PR-1/PR-3): 在线数接 /devices + /status 真数据；当前为占位。
const onlineCount = ref(10)
const totalCount = ref(12)
const notificationCount = ref(3)
</script>

<style scoped>
.header {
  height: var(--topbar-h);
  background: var(--surface);
  border-bottom: 1px solid var(--line);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 22px;
  gap: 18px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
  min-width: 0;
}

.crumb {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 13px;
  color: var(--ink-3);
  white-space: nowrap;
}
.crumb b {
  color: var(--ink);
  font-weight: 600;
}
.crumb-sep {
  color: var(--line-strong);
}

.field-search {
  width: 240px;
  max-width: 32vw;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 14px;
}

.device-status {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 13px;
  color: var(--ink-2);
  white-space: nowrap;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.status-dot.online {
  background: var(--st-conv);
  box-shadow: 0 0 0 3px var(--st-conv-bg);
}

.notification-icon {
  cursor: pointer;
}

.locale-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 28px;
  padding: 0 10px;
  border: 1px solid var(--line);
  border-radius: 999px;
  background: var(--surface);
  color: var(--ink-2);
  font-size: 12.5px;
  font-family: inherit;
  cursor: pointer;
}
.locale-btn:hover {
  color: var(--ink);
  border-color: var(--line-strong);
}
.locale-ico {
  width: 14px;
  height: 14px;
  stroke: currentColor;
  stroke-width: 1.6;
  fill: none;
  flex-shrink: 0;
}
.locale-label {
  white-space: nowrap;
}

.user-area {
  display: flex;
  align-items: center;
  gap: 9px;
  cursor: pointer;
  padding-left: 4px;
}
.avatar {
  width: 30px;
  height: 30px;
  border-radius: 8px;
  background: var(--primary);
  color: #fff;
  display: grid;
  place-items: center;
  font-weight: 600;
  font-size: 13px;
  flex-shrink: 0;
}
.user-meta {
  font-size: 12.5px;
  line-height: 1.25;
}
.user-meta b {
  display: block;
  font-weight: 600;
  color: var(--ink);
}
.user-meta span {
  color: var(--ink-3);
  font-size: 11px;
}

@media (max-width: 720px) {
  .field-search,
  .user-meta {
    display: none;
  }
}
</style>
