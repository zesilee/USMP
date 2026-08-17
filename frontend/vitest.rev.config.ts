import { defineConfig } from 'vitest/config'
import { fileURLToPath } from 'node:url'

// 波 A 反向 alias 验证配置（change frontend-eviewui-inula-switch）：
// inula-intl/inula-router 的 CJS 构建 require("openinula")，把 openinula
// 反向别名到 react——验证两件套在 React 19 运行时上可用（它们只消费
// 17 级 API）。跑 test/rev/ 探针：npx vitest run --config vitest.rev.config.ts
export default defineConfig({
  test: {
    globals: true,
    environment: 'happy-dom',
    include: ['test/rev/**/*.{test,spec}.{ts,tsx}'],
    setupFiles: ['./test/setup.ts'],
    server: { deps: { inline: [/inula-/] } },
    alias: [
      // 两件套 ESM 构建内嵌 openinula hook 运行时（反向 alias 改不了内嵌代码），
      // 强制走 external require("openinula") 的 CJS 构建。
      { find: /^inula-intl$/, replacement: 'inula-intl/build/cjs/intl.js' },
      { find: /^inula-router$/, replacement: 'inula-router/router/cjs/router.js' },
      { find: /^openinula\/jsx-runtime$/, replacement: 'react/jsx-runtime' },
      { find: /^openinula$/, replacement: 'react' },
    ],
  },
  resolve: {
    alias: [
      { find: /^openinula\/jsx-runtime$/, replacement: 'react/jsx-runtime' },
      { find: /^openinula$/, replacement: 'react' },
      { find: '@', replacement: fileURLToPath(new URL('./src', import.meta.url)) },
    ],
  },
})
