import { defineConfig } from 'vitest/config'
import { fileURLToPath } from 'node:url'

export default defineConfig({
  test: {
    environment: 'happy-dom',
    setupFiles: ['./test/setup.ts'],
    globals: true,
    testTimeout: 15000,
    // React 重建期窗口：运行纯逻辑层套件（框架无关）。test/{stores,composables}
    // 其余套件为沿用资产，被测源随重建逐组加回 include（tasks 5/6/10 组）；
    // 浏览器模式套件随 tasks 12.1 重建配置。
    include: [
      'test/utils/**/*.{test,spec}.{js,ts,jsx,tsx}',
      'test/golden/**/*.{test,spec}.{js,ts,jsx,tsx}',
      'test/styles/**/*.{test,spec}.{js,ts,jsx,tsx}',
      'test/composables/useFieldLabels.test.ts',
      // deriveOverview 纯函数面随占位保留（vue 外壳退役），套件持续在场。
      'test/composables/useFleetOverview.test.ts',
    ],
    exclude: ['**/node_modules/**', '**/dist/**'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      // 覆盖率口径随清场重算分母（先例：2026-07-13 legacy CRD 退役）：窗口期
      // 仅测纯逻辑层，口径收敛到其被测面；React 层重建时逐组扩回。
      include: ['src/utils/**/*.ts', 'src/i18n/**/*.ts', 'src/composables/**/*.ts'],
      exclude: ['src/**/*.d.ts'],
      // 覆盖率「不下降」棘轮（T08）：阈值 = 当前实测水平向下取整留余量。
      // 只准升不准降——低于阈值 CI 即 fail。补测后应把阈值同步上调，形成单向棘轮。
      // 历史轨迹（Vue 全量口径）：2026-07-06 66.55/66.57/56.67/66.88 →
      // 2026-07-24 起 86.5/79.8/81.0/87.5。窗口口径实测（2026-08-14，含
      // composables 纯函数面）：95.14/85.87/95.16/96.39；React 层组件测试回归后
      // 恢复全量口径并逐步爬回。
      thresholds: {
        statements: 94.5,
        branches: 85.5,
        functions: 94.5,
        lines: 95.5
      }
    }
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  }
})
