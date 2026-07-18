<template>
  <div class="sidebar" :class="{ collapsed: isCollapsed }">
    <div class="brand">
      <div class="brand-mark" aria-hidden="true"></div>
      <div v-if="!isCollapsed" class="brand-text">
        <div class="brand-name">USMP</div>
        <div class="brand-sub">Switch Mgmt</div>
      </div>
    </div>

    <el-menu
      class="nav"
      :default-active="activeMenu"
      :collapse="isCollapsed"
      router
    >
      <el-menu-item index="/">
        <el-icon><DataLine /></el-icon>
        <template #title>{{ t('nav.overview') }}</template>
      </el-menu-item>

      <el-menu-item index="/devices">
        <el-icon><Monitor /></el-icon>
        <template #title>{{ t('nav.devices') }}</template>
      </el-menu-item>

      <!-- 原生配置菜单（LT-03）：SND 左树驱动 14 组/3 层导航（已接入可点、未接入
           禁用占位）；left-tree 失败回退 category 分组（R08 导航不消失）。 -->
      <el-sub-menu index="native-config">
        <template #title>
          <el-icon><Connection /></el-icon>
          <span>{{ t('nav.nativeConfig') }}</span>
        </template>
        <template v-if="leftTreeReady">
          <LeftTreeMenu :nodes="leftTree" index-prefix="lt" />
        </template>
        <!-- 降级：任务域分组（FE-13）：任一模块带 category 时按组渲染，未标注归「其他」；
             全部未标注则平铺（等价旧形态）。 -->
        <template v-else-if="nativeGrouped">
          <el-menu-item-group
            v-for="g in nativeGroups"
            :key="g.category || '__default__'"
            :title="g.category || t('nav.otherGroup')"
          >
            <el-menu-item
              v-for="m in g.modules"
              :key="m.name"
              :index="`/module/${m.name}`"
              :data-test="`module-item-${m.name}`"
            >
              {{ m.title }}
            </el-menu-item>
          </el-menu-item-group>
        </template>
        <template v-else>
          <el-menu-item
            v-for="m in nativeModules"
            :key="m.name"
            :index="`/module/${m.name}`"
            :data-test="`module-item-${m.name}`"
          >
            {{ m.title }}
          </el-menu-item>
        </template>
      </el-sub-menu>

      <!-- 业务网络配置（FE-17）：task-name=business-network 的意图模块自动出现
           （category 分桶零硬编码），指向平台作用域业务控制台 /business/:module。
           无业务模块（后端未接入/降级）时整组隐藏。 -->
      <el-sub-menu v-if="businessModules.length" index="business-config" data-test="business-group">
        <template #title>
          <el-icon><Share /></el-icon>
          <span>{{ t('nav.businessConfig') }}</span>
        </template>
        <el-menu-item
          v-for="m in businessModules"
          :key="m.name"
          :index="`/business/${m.name}`"
          :data-test="`business-item-${m.name}`"
        >
          {{ m.title }}
        </el-menu-item>
      </el-sub-menu>

      <el-menu-item index="/logs">
        <el-icon><Document /></el-icon>
        <template #title>{{ t('nav.logs') }}</template>
      </el-menu-item>

      <el-menu-item index="/settings">
        <el-icon><Tools /></el-icon>
        <template #title>{{ t('nav.settings') }}</template>
      </el-menu-item>
    </el-menu>

    <div v-if="!isCollapsed" class="side-foot">
      <span class="pulse-dot" aria-hidden="true"></span>
      <span>{{ t('nav.sideFoot') }}</span>
    </div>

    <div class="collapse-btn" @click="toggleCollapse">
      <el-icon v-if="isCollapsed"><Expand /></el-icon>
      <el-icon v-else><Fold /></el-icon>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useMenuStore } from '../../stores/menu'
import LeftTreeMenu from './LeftTreeMenu.vue'
import { DataLine, Monitor, Connection, Document, Tools, Fold, Expand, Share } from '@element-plus/icons-vue'

const route = useRoute()
const menuStore = useMenuStore()
const { t } = useI18n()

const activeMenu = computed(() => route.path)
const isCollapsed = computed(() => menuStore.isCollapsed)
const nativeModules = computed(() => menuStore.nativeModules)
const nativeGroups = computed(() => menuStore.nativeGroups)
const nativeGrouped = computed(() => nativeGroups.value.some((g) => g.category))
const leftTree = computed(() => menuStore.leftTree)
const leftTreeReady = computed(() => leftTree.value.length > 0)
const businessModules = computed(() => menuStore.businessModules)

onMounted(() => {
  // 左树为主路径；nativeModules 仍加载（业务菜单 + 左树降级路径共用）。
  menuStore.loadLeftTree()
  menuStore.loadNativeModules()
})

function toggleCollapse() {
  menuStore.toggleCollapse()
}
</script>

<style scoped>
.sidebar {
  width: var(--sidebar-w);
  height: 100vh;
  background: var(--surface);
  border-right: 1px solid var(--line);
  display: flex;
  flex-direction: column;
  transition: width 0.25s var(--ease, cubic-bezier(0.4, 0, 0.2, 1));
}
.sidebar.collapsed {
  width: 64px;
}

/* 品牌区 */
.brand {
  height: var(--topbar-h);
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 18px;
  border-bottom: 1px solid var(--line);
  flex-shrink: 0;
}
.brand-mark {
  width: 26px;
  height: 26px;
  border-radius: 6px;
  background: var(--ink);
  position: relative;
  flex-shrink: 0;
  display: grid;
  place-items: center;
}
.brand-mark::before {
  content: '';
  width: 12px;
  height: 12px;
  border: 2px solid var(--brand);
  border-radius: 3px;
  transform: rotate(45deg);
}
.brand-name {
  font-weight: 700;
  letter-spacing: 0.06em;
  font-size: 15px;
  color: var(--ink);
  line-height: 1.1;
}
.brand-sub {
  font-size: 10.5px;
  color: var(--ink-3);
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

/* 导航（覆盖 el-menu 为浅色 iMaster 风格） */
.nav {
  flex: 1;
  border-right: none;
  padding: 10px;
  overflow-y: auto;
  background: transparent;
}
.nav :deep(.el-menu-item),
.nav :deep(.el-sub-menu__title) {
  height: 38px;
  line-height: 38px;
  border-radius: var(--r-ctl);
  margin: 2px 0;
  color: var(--ink-2);
  font-weight: 500;
  font-size: 13.5px;
}
.nav :deep(.el-menu-item .el-icon),
.nav :deep(.el-sub-menu__title .el-icon) {
  color: currentColor;
}
.nav :deep(.el-menu-item:hover),
.nav :deep(.el-sub-menu__title:hover) {
  background: var(--sunken);
  color: var(--ink);
}
.nav :deep(.el-menu-item.is-active) {
  background: var(--primary-weak);
  color: var(--primary-ink);
  font-weight: 600;
}
.nav :deep(.el-sub-menu .el-menu-item) {
  min-width: 0;
  padding-left: 44px !important;
}

/* 侧栏底部状态 */
.side-foot {
  border-top: 1px solid var(--line);
  padding: 12px 16px;
  font-size: 11.5px;
  color: var(--ink-3);
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}
.pulse-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--st-conv);
  animation: pulse 2.4s infinite;
}
@keyframes pulse {
  0% { box-shadow: 0 0 0 0 rgba(16, 129, 74, 0.45); }
  70% { box-shadow: 0 0 0 6px rgba(16, 129, 74, 0); }
  100% { box-shadow: 0 0 0 0 rgba(16, 129, 74, 0); }
}

.collapse-btn {
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  border-top: 1px solid var(--line);
  color: var(--ink-3);
  flex-shrink: 0;
}
.collapse-btn:hover {
  color: var(--primary);
  background: var(--sunken);
}

@media (prefers-reduced-motion: reduce) {
  .pulse-dot { animation: none; }
}
</style>
