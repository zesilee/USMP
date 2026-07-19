<template>
  <!-- UI-01/D4：ElementPlus locale 随语言响应切换（zh-cn/en-us），全局控件文案即时生效 -->
  <el-config-provider :locale="epLocale">
    <MainLayout>
    <!-- :key 按路由路径重建组件：同一组件被相邻路由复用时（如 DeviceConfigPage 同时服务
         /config/vlan 与 /config/interface），否则 setup/onMounted 不重跑 → schema 不重载。 -->
      <router-view :key="$route.path" />
    </MainLayout>
  </el-config-provider>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import enUs from 'element-plus/es/locale/lang/en'
import MainLayout from './components/layout/MainLayout.vue'
import { useLocaleStore } from './stores/locale'

const localeStore = useLocaleStore()
const epLocale = computed(() => (localeStore.locale === 'en-us' ? enUs : zhCn))
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body, #app {
  height: 100%;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}
</style>
