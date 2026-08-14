import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import { playwright } from '@vitest/browser-playwright'
import { fileURLToPath } from 'node:url'

// F3 真浏览器套件配置（与默认 happy-dom 套件分离）。
// 用真 Chromium 渲染组件，antd 的 Select 弹层/teleport、嵌套 list 行真实落地，
// 断言的是真实渲染结果而非 happy-dom 近似。运行：npm run test:browser
export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    include: ['test/browser/**/*.{test,spec}.{ts,tsx}'],
    setupFiles: ['./test/setup.ts'],
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
