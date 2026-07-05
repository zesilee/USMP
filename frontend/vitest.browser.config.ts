import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { playwright } from '@vitest/browser-playwright'
import { fileURLToPath } from 'node:url'

// 浏览器模式配置（与默认 happy-dom 套件分离）。
// 用真 Chromium 渲染组件，Element Plus 的 el-table/el-select 等真实出行/展开，
// 断言的是真实渲染结果而非 happy-dom 近似。运行：npm run test:browser
export default defineConfig({
  plugins: [vue()],
  test: {
    globals: true,
    include: ['test/browser/**/*.{test,spec}.{ts,tsx}'],
    browser: {
      enabled: true,
      provider: playwright(),
      headless: true,
      instances: [{ browser: 'chromium' }],
    },
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
})
