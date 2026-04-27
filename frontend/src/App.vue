<template>
  <div id="app" class="app-container">
    <!-- Header -->
    <header class="app-header">
      <div class="header-left">
        <div class="logo">
          <svg width="28" height="28" viewBox="0 0 28 28" fill="none">
            <rect width="28" height="28" rx="6" fill="#165DFF"/>
            <path d="M7 14H21M14 7V21" stroke="white" stroke-width="2.5" stroke-linecap="round"/>
            <circle cx="14" cy="14" r="4" stroke="white" stroke-width="2"/>
          </svg>
        </div>
        <h1 class="app-title">交换机设备管理平台</h1>
      </div>
      <div class="header-right">
        <div class="header-status">
          <span class="status-dot status-dot--success"></span>
          <span class="status-text">系统运行正常</span>
        </div>
      </div>
    </header>

    <!-- Main Content -->
    <div class="app-body">
      <!-- Sidebar - Device Tree -->
      <aside class="app-sidebar">
        <DeviceTree
          :devices="devices"
          @node-selected="onNodeSelected"
        />
      </aside>

      <!-- Content Area -->
      <main class="app-content">
        <div v-if="currentDevice && currentYangPath" class="content-wrapper">
          <VlanManager
            v-if="currentYangPath === '/vlans'"
            :device-ip="currentDevice.ip"
          />
          <InterfaceManager
            v-else-if="currentYangPath === '/interfaces'"
            :device-ip="currentDevice.ip"
          />
        </div>

        <!-- Empty State -->
        <div v-else class="empty-state">
          <div class="empty-icon">
          </div>
          <h3 class="empty-title">选择设备和配置项</h3>
          <p class="empty-desc">从左侧导航树选择设备和 YANG 配置节点，开始管理交换机配置</p>
        </div>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import DeviceTree from './components/DeviceTree.vue'
import VlanManager from './components/vlan/VlanManager.vue'
import InterfaceManager from './components/interfaces/InterfaceManager.vue'
import type { DeviceInfo } from './types/yang'

const devices = ref<DeviceInfo[]>([
  {
    ip: '192.168.1.1',
    port: 830,
    username: 'admin',
    password: 'admin'
  }
])

const currentDevice = ref<DeviceInfo | null>(null)
const currentYangPath = ref('')
const currentYangName = ref('')

const onNodeSelected = (device: DeviceInfo, yangPath: string, name: string) => {
  currentDevice.value = device
  currentYangPath.value = yangPath
  currentYangName.value = name
}
</script>

<style lang="scss">
@import './styles/variables.scss';

.app-container {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background-color: $bg-page;
}

// Header
.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 56px;
  padding: 0 $spacing-xl;
  background-color: $bg-card;
  border-bottom: 1px solid $border-color;

  .header-left {
    display: flex;
    align-items: center;
    gap: $spacing-md;

    .app-title {
      font-size: $font-size-lg;
      font-weight: $font-weight-semibold;
      color: $text-primary;
      margin: 0;
    }
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: $spacing-lg;

    .header-status {
      display: flex;
      align-items: center;
      gap: $spacing-sm;

      .status-text {
        font-size: $font-size-sm;
        color: $text-secondary;
      }
    }
  }
}

// Body Layout
.app-body {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.app-sidebar {
  width: 260px;
  background-color: $bg-card;
  border-right: 1px solid $border-color;
  overflow-y: auto;
  flex-shrink: 0;
}

.app-content {
  flex: 1;
  overflow-y: auto;
  padding: $spacing-xl;
}

// Content Wrapper
.content-wrapper {
  max-width: 1000px;
  margin: 0 auto;
}

// Empty State
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  min-height: 400px;

  .empty-title {
    font-size: $font-size-xl;
    font-weight: $font-weight-semibold;
    color: $text-primary;
    margin: 20px 0 8px 0;
  }

  .empty-desc {
    font-size: $font-size-base;
    color: $text-tertiary;
    margin: 0;
  }
}

// Status Dot
.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;

  &--success {
    background-color: $color-success;
    box-shadow: 0 0 0 3px rgba($color-success, 0.2);
  }
}

// Module Placeholder
.module-placeholder {
  padding: 60px 20px;
}
</style>
