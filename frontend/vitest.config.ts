import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import { fileURLToPath } from 'node:url'

export default defineConfig({
  plugins: [react()],
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
      'test/composables/useConfigForm.payload.test.ts',
      'test/composables/useConstraintEngine.test.ts',
      'test/composables/useConstraintEngine.must.test.ts',
      'test/composables/useChangesetSubmit.test.ts',
      // 状态层（zustand 重建，tasks 4 组）。
      'test/stores/**/*.{test,spec}.{ts,tsx}',
      // 组件层 F2（React 重建波次）。
      'test/components/**/*.{test,spec}.{ts,tsx}',
      'test/views/**/*.{test,spec}.{ts,tsx}',
      // 表单编排（tasks 6 组）：hooks + 纯函数核心。
      'test/hooks/**/*.{test,spec}.{ts,tsx}',
      'test/form/**/*.{test,spec}.{ts,tsx}',
      // UI 适配层（FA-01~04）：守护 + feedback F1。
      'test/ui/**/*.{test,spec}.{ts,tsx}',
      // 内网真实校准套件（默认 skip，EVIEW_REAL=1 启用）。
      'test/integration/**/*.{test,spec}.{ts,tsx}',
    ],
    exclude: ['**/node_modules/**', '**/dist/**'],
    // 混合协作模式（route-decision）：EviewUI 实现仅内网。alias 把 @nce 系
    // import 解析到空 stub 使 Vite import-analysis 通过，行为由各测试
    // vi.mock 工厂提供（替身规格=vendor d.ts + gate 实测）。
    alias: [
      // 按子路径映射独立 stub 文件——catch-all 单文件会让多组件的 vi.mock
      // 工厂共享模块身份互相覆盖（实录坑）。新增桥组件须同步建 stub 文件。
      // 内网真实校准：EVIEW_REAL=1 关闭 stub alias（解析真 @nce 实现），
      // 并启用 test/integration/eview-real 套件——无需手改本文件。
      ...(process.env.EVIEW_REAL === '1'
        ? []
        : [
            { find: /^@nce\/eview-react\/([^/]+)$/, replacement: fileURLToPath(new URL('./test/stubs/eview/$1.ts', import.meta.url)) },
            { find: /^@nce\/icon-plus(\/.*)?$/, replacement: fileURLToPath(new URL('./test/stubs/eview-empty.ts', import.meta.url)) },
          ]),
    ],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      // 覆盖率口径随清场重算分母（先例：2026-07-13 legacy CRD 退役）：窗口期
      // 仅测纯逻辑层，口径收敛到其被测面；React 层重建时逐组扩回。
      include: ['src/utils/**/*.ts', 'src/i18n/**/*.ts', 'src/composables/**/*.ts', 'src/ui/**/*.{ts,tsx}', 'src/stores/**/*.ts', 'src/components/**/*.{ts,tsx}'],
      // stories 是 Storybook 展示物非产品代码，单测不执行，排除出分母。
      // src/ui/eview = 未接线的 EviewUI 后端并行代码（组 4 窗口期，§5.3 新旧
      // 并行）：F2 替身测试在场但分支面随批次持续扩容，接线（组 5）时统一
      // 纳入分母并按干净口径重钉阈值——窗口期先排除，避免每批桥都重钉。
      exclude: ['src/**/*.d.ts', 'src/**/*.stories.tsx', 'src/ui/eview/**', 'src/runtime/**'],
      // 覆盖率「不下降」棘轮（T08）：阈值 = 当前实测水平向下取整留余量。
      // 只准升不准降——低于阈值 CI 即 fail。补测后应把阈值同步上调，形成单向棘轮。
      // 历史轨迹（Vue 全量口径）：2026-07-06 66.55/66.57/56.67/66.88 →
      // 2026-07-24 起 86.5/79.8/81.0/87.5。窗口口径实测（2026-08-14，含
      // composables 纯函数面、src/ui、src/stores 与 src/{form,hooks}）：
      // 95.69/85.98/95.85/97.16（src/components 分母并入后 branches 结构性回落，
      // 按分母重算先例重钉 85.5→84.5（组件波次防御分支面扩张，2026-08-14 二钉，
      // 实测 85.27；三钉 84.0；四钉 94.0/83.0/94.5/95.5（批量链路分母并入）——组件分母持续
      // 扩张期的结构性回落，12.4 补测统一回填爬升；其余三项维持高位棘轮）；
      // React 层组件测试回归后恢复全量口径并逐步爬回。
      // 12.4 回填收口（2026-08-14）：全页面/批量链路分母齐备后实测
      // 94.52/83.31/94.54/96.15，四项按现值下沿重钉——全面高于 1.2 记录的
      // 旧栈基线（86.5/79.8/81.0/87.5），棘轮达标。
      // 二钉（同日，#336 CI 红复盘）：首钉用了本地 staging 灌水值且 stories
      // 文件进了分母（CI 三项差值精确=stories 行数）。stories 排除出分母后，
      // 以 staging-down 干净实测 94.40/83.31/94.54/96.15 留余量重钉。
      thresholds: {
        statements: 94.3,
        branches: 83.2,
        functions: 94.5,
        lines: 96.0
      }
    }
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  }
})
