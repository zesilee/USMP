<template>
  <div class="main-layout">
    <div class="sidebar-wrapper">
      <Sidebar />
    </div>

    <div class="content-wrapper">
      <div class="header-wrapper">
        <Header />
      </div>

      <main class="main-content">
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import Sidebar from './Sidebar.vue'
import Header from './Header.vue'
</script>

<style scoped>
.main-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
}

.sidebar-wrapper {
  flex-shrink: 0;
}

.content-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.main-content {
  flex: 1;
  padding: 24px;
  overflow-y: auto;
  background: #f5f7fa;
}
</style>
