import { defineStore } from 'pinia'
import { ref } from 'vue'
import axios from 'axios'

interface NativeModel {
  name: string
  title: string
  vendor: string
}

export const useMenuStore = defineStore('menu', () => {
  const nativeModels = ref<NativeModel[]>([])
  const nativeMenuLoaded = ref(false)
  const isCollapsed = ref(false)

  async function loadNativeModels() {
    if (nativeMenuLoaded.value) return
    try {
      const res = await axios.get('/api/crd/models?type=native')
      nativeModels.value = res.data.models || []
      nativeMenuLoaded.value = true
    } catch (e) {
      console.error('Failed to load native models:', e)
    }
  }

  function toggleCollapse() {
    isCollapsed.value = !isCollapsed.value
  }

  return {
    nativeModels,
    nativeMenuLoaded,
    isCollapsed,
    loadNativeModels,
    toggleCollapse
  }
})
