import { defineConfig } from 'vitest/config'
import { fileURLToPath } from 'node:url'

// 翻转波探针配置：react → openinula+垫片（src/runtime/react-shim），验证
// antd 组件能否在 openinula 运行时上存活（决定"运行时先切、组件库后换"
// 的渐进路径是否成立）。跑 test/flip/ 探针。
const SHIM = fileURLToPath(new URL('./src/runtime/react-shim.ts', import.meta.url))
const ALIASES = [
  { find: /^react\/jsx-dev-runtime$/, replacement: 'openinula/jsx-dev-runtime' },
  { find: /^react\/jsx-runtime$/, replacement: 'openinula/jsx-runtime' },
  { find: /^react-dom\/client$/, replacement: SHIM },
  { find: /^react-dom\/test-utils$/, replacement: SHIM },
  { find: /^react-dom$/, replacement: SHIM },
  { find: /^react$/, replacement: SHIM },
]

export default defineConfig({
  test: {
    globals: true,
    environment: 'happy-dom',
    include: ['test/flip/**/*.{test,spec}.{ts,tsx}'],
    setupFiles: ['./test/gate/setup.ts'],
    server: { deps: { inline: true } },
    alias: ALIASES,
  },
  resolve: {
    alias: [...ALIASES, { find: '@', replacement: fileURLToPath(new URL('./src', import.meta.url)) }],
  },
})
