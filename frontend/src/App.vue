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
        <div class="sidebar-title">设备列表</div>
        <DeviceTree
          :devices="devices"
          @node-selected="onNodeSelected"
        />
      </aside>

      <!-- Content Area -->
      <main class="app-content">
        <div v-if="currentDevice && currentYangPath" class="content-wrapper">
          <div class="page-header">
            <h2 class="page-title">{{ currentYangName }}</h2>
            <p class="page-description">设备: {{ currentDevice.ip }}</p>
          </div>
          <VlanManager
            v-if="currentYangPath === '/vlans'"
            :device-ip="currentDevice.ip"
          />
          <InterfaceManager
            v-else-if="currentYangPath === '/interfaces'"
            :device-ip="currentDevice.ip"
          />
        </div>

        <!-- Dashboard - Default View -->
        <Dashboard v-else />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import DeviceTree from './components/DeviceTree.vue'
import VlanManager from './components/vlan/VlanManager.vue'
import InterfaceManager from './components/interfaces/InterfaceManager.vue'
import Dashboard from './components/Dashboard.vue'
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
.app-container {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background-color: var(--bg-page);
}

// Header
.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 64px;
  padding: 0 var(--spacing-xl);
  background-color: var(--bg-card);
  border-bottom: 1px solid var(--border-color);
  box-shadow: var(--shadow-e1);
  z-index: 10;

  .header-left {
    display: flex;
    align-items: center;
    gap: var(--spacing-md);

    .app-title {
      font-size: var(--font-size-xl);
      font-weight: var(--font-weight-semibold);
      color: var(--text-primary);
      margin: 0;
      letter-spacing: -0.01em;
    }
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: var(--spacing-lg);

    .header-status {
      display: flex;
      align-items: center;
      gap: var(--spacing-sm);
      padding: var(--spacing-sm) var(--spacing-md);
      background-color: var(--bg-elevated);
      border-radius: var(--radius-md);

      .status-text {
        font-size: var(--font-size-sm);
        color: var(--text-secondary);
        font-weight: var(--font-weight-medium);
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
  width: 280px;
  background-color: var(--bg-card);
  border-right: 1px solid var(--border-color);
  overflow-y: auto;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;

  .sidebar-title {
    padding: var(--spacing-lg) var(--spacing-xl) var(--spacing-md);
    font-size: var(--font-size-xs);
    font-weight: var(--font-weight-semibold);
    color: var(--text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.08em;
  }
}

.app-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-2xl);
}

// Page Header
.page-header {
  margin-bottom: var(--spacing-2xl);

  .page-title {
    font-size: var(--font-size-3xl);
    font-weight: var(--font-weight-bold);
    color: var(--text-primary);
    margin: 0 0 var(--spacing-sm);
    letter-spacing: -0.02em;
    white-space: nowrap;
  }

  .page-description {
    font-size: var(--font-size-base);
    color: var(--text-secondary);
    margin: 0;
    white-space: nowrap;
  }
}

// Content Wrapper
.content-wrapper {
  max-width: 1200px;
}

// Empty State
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  min-height: 500px;

  .empty-icon {
    opacity: 0.4;
    margin-bottom: var(--spacing-xl);
  }

  .empty-title {
    font-size: var(--font-size-xl);
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
    margin: 0 0 var(--spacing-sm);
  }

  .empty-desc {
    font-size: var(--font-size-base);
    color: var(--text-tertiary);
    margin: 0;
    max-width: 400px;
    text-align: center;
  }
}

// Status Dot
.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;

  &--success {
    background-color: var(--color-success);
    box-shadow: 0 0 0 3px var(--color-success-bg);
  }
}
</style>
