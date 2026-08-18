import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { fileURLToPath } from 'node:url'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      // 生产真身 = EviewUI 桥（组 5 接线单点切换；测试侧的同名别名见
      // vitest.config.ts——外网映射 antd 镜像）。
      '@ui-backend': fileURLToPath(new URL('./src/ui/eview', import.meta.url)),
    },
  },
  server: {
    port: 3000,
  },
})
