import { defineConfig } from 'vitest/config'
import { fileURLToPath } from 'node:url'

// 垂直切片闸门配置（change frontend-eviewui-inula-switch 组 1）：
// 把 react 系 alias 到 openinula，独立跑 test/gate/ 探针套件——验证
// vitest + @testing-library/react + happy-dom 在 openinula 运行时下可用
// （tasks 1.2 红线项），不影响主套件（vitest.config.ts 仍指 react）。
// 运行：npx vitest run --config vitest.gate.config.ts
export default defineConfig({
  test: {
    globals: true,
    environment: 'happy-dom',
    include: ['test/gate/**/*.{test,spec}.{ts,tsx}'],
    setupFiles: ['./test/gate/setup.ts'],
    // test.alias 对 inline 依赖也生效（resolve.alias 只影响源码面），
    // testing-library 内部的 react-dom/client 导入靠这里接管。
    alias: [
      { find: /^react\/jsx-dev-runtime$/, replacement: 'openinula/jsx-dev-runtime' },
      { find: /^react\/jsx-runtime$/, replacement: 'openinula/jsx-runtime' },
      { find: /^react-dom\/client$/, replacement: 'openinula' },
      { find: /^react-dom\/test-utils$/, replacement: 'openinula' },
      { find: /^react-dom$/, replacement: 'openinula' },
      { find: /^react$/, replacement: 'openinula' },
      { find: /^react-intl$/, replacement: 'inula-intl' },
    ],
    server: {
      deps: {
        // 关键：testing-library 是 node_modules 里的 CJS 依赖，默认外部化时
        // 其内部 import 'react-dom/client' 不经过 alias（会加载真 react-dom，
        // 与 openinula 元素 vtype 互不相认）。inline 使其走转换管线吃到 alias。
        inline: true,
      },
    },
  },
  resolve: {
    alias: [
      { find: /^react\/jsx-dev-runtime$/, replacement: 'openinula/jsx-dev-runtime' },
      { find: /^react\/jsx-runtime$/, replacement: 'openinula/jsx-runtime' },
      { find: /^react-dom\/client$/, replacement: 'openinula' },
      { find: /^react-dom\/test-utils$/, replacement: 'openinula' },
      { find: /^react-dom$/, replacement: 'openinula' },
      { find: /^react$/, replacement: 'openinula' },
      { find: /^react-intl$/, replacement: 'inula-intl' },
      { find: '@', replacement: fileURLToPath(new URL('./src', import.meta.url)) },
    ],
  },
})
