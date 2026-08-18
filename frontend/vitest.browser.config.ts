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
      // 组 5 接线：F3 真浏览器与 happy-dom 同口径走 antd 测试镜像
      // （外网无 EviewUI 真包，antd-backend/README）；桥真浏览器校准
      // 属组 6.1（内网跑时另行切换）。
      '@ui-backend': fileURLToPath(new URL('./src/ui/antd-backend', import.meta.url)),
      '@bridge': fileURLToPath(new URL('./src/ui/eview', import.meta.url)),
    },
  },
})
