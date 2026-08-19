import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import { playwright } from '@vitest/browser-playwright'
import { fileURLToPath } from 'node:url'

// F3 真浏览器套件配置（与默认 happy-dom 套件分离）。
// 用真 Chromium 渲染组件，antd 的 Select 弹层/teleport、嵌套 list 行真实落地，
// 断言的是真实渲染结果而非 happy-dom 近似。运行：npm run test:browser
export default defineConfig({
  plugins: [react()],
  // 真浏览器无 node process——REAL 开关经构建期常量注入。
  define: {
    __EVIEW_REAL__: JSON.stringify(process.env.EVIEW_REAL === '1'),
  },
  test: {
    globals: true,
    include: ['test/browser/**/*.{test,spec}.{ts,tsx}'],
    setupFiles: ['./test/setup.ts'],
    // 外网（非 REAL）：eview-real-browser 套件顶层引真桥（@bridge）——@nce 系
    // import 与 happy-dom 配置同款 stub 别名保收集期通过（用例本体 skip）。
    alias: [
      ...(process.env.EVIEW_REAL === '1'
        ? []
        : [
            { find: /^@nce\/eview-react\/locales\/.+$/, replacement: fileURLToPath(new URL('./test/stubs/eview-locales.ts', import.meta.url)) },
            { find: /^@nce\/eview-react\/([^/]+)$/, replacement: fileURLToPath(new URL('./test/stubs/eview/$1.ts', import.meta.url)) },
            { find: /^@nce\/icon-plus(\/.*)?$/, replacement: fileURLToPath(new URL('./test/stubs/eview-empty.ts', import.meta.url)) },
          ]),
    ],
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
      // （外网无 EviewUI 真包，antd-backend/README）。组 6.1：EVIEW_REAL=1
      // （内网）切真桥全链——eview-real-browser 套件在真 Chromium 校准
      // happy-dom 移交项（Tabs 受控/点击、Tree 全交互）。
      '@ui-backend': fileURLToPath(
        new URL(process.env.EVIEW_REAL === '1' ? './src/ui/eview' : './src/ui/antd-backend', import.meta.url),
      ),
      '@bridge': fileURLToPath(new URL('./src/ui/eview', import.meta.url)),
      '@app-router': fileURLToPath(new URL('./src/router/compat.ts', import.meta.url)),
    },
  },
})
