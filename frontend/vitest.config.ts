import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath } from 'node:url'

export default defineConfig({
  plugins: [vue()],
  test: {
    environment: 'happy-dom',
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
      thresholds: {
        statements: 64,
        branches: 64,
        functions: 54,
        lines: 64
      }
    }
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  }
})
