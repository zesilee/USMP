import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath } from 'node:url'

export default defineConfig({
  plugins: [vue()],
  test: {
    environment: 'happy-dom',
    setupFiles: ['./test/setup.ts'],
    globals: true,
    include: ['src/**/*.{test,spec}.{js,ts,jsx,tsx}', 'test/**/*.{test,spec}.{js,ts,jsx,tsx}'],
    // test/browser/** 属浏览器模式套件（vitest.browser.config.ts），不在 happy-dom 下跑
    exclude: ['**/node_modules/**', '**/dist/**', 'test/browser/**'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      include: ['src/**/*.{vue,ts,tsx}'],
      exclude: ['src/**/*.d.ts'],
      // 覆盖率「不下降」棘轮（T08）：阈值 = 当前实测水平向下取整留余量。
      // 只准升不准降——低于阈值 CI 即 fail。补测后应把阈值同步上调，形成单向棘轮。
      // 基线实测(2026-07-06)：Stmts 66.55 / Branch 66.57 / Funcs 56.67 / Lines 66.88。
      // 2026-07-08 P3 choice 补测后实测：Stmts 71.25 / Branch 70.02 / Funcs 62.75 / Lines 71.47。
      // 2026-07-13 legacy CRD 链路退役（低覆盖代码删除，分母重算）后实测：
      // Stmts 79.53 / Branch 75.61 / Funcs 72.21 / Lines 80.08。
      thresholds: {
        statements: 82,
        branches: 77,
        functions: 74,
        lines: 82
      }
    }
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  }
})
