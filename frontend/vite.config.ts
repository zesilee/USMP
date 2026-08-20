import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { fileURLToPath } from 'node:url'

// 波 C 运行时翻转（RT-01）：USMP_RUNTIME=inula 时构建产物为 openinula 运行时
// ——react/react-dom/jsx-runtime 别名到 openinula、react-intl→inula-intl
// （eview 编译产物内部 require('react')/require('react-intl') 一并接管）、
// @app-router→compat.inula（inula-router v5 实现）。缺省仍 React 19（外网
// 开发/e2e-local antd 口径）；内网交付构建：USMP_RUNTIME=inula npm run build。
// 2026-08-20 默认翻转（内网 E2E 21/21 验收通过）：缺省即 openinula；
// USMP_RUNTIME=react 显式回退；USMP_UI_BACKEND=antd（e2e-local 口径）强制
// react——antd 需 React 18+，两开关联动防误配。
const INULA = process.env.USMP_UI_BACKEND === 'antd' ? false : process.env.USMP_RUNTIME !== 'react'
const inulaAliases = INULA
  ? {
      react: 'openinula',
      'react-dom/client': 'openinula',
      'react-dom': 'openinula',
      'react/jsx-runtime': 'openinula/jsx-runtime',
      'react/jsx-dev-runtime': 'openinula/jsx-dev-runtime',
      'react-intl': 'inula-intl',
    }
  : {}

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      ...inulaAliases,
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      // 生产真身 = EviewUI 桥（组 5 接线单点切换；测试侧的同名别名见
      // vitest.config.ts——外网映射 antd 镜像）。组 7.2：USMP_UI_BACKEND=antd
      // 时构建走 antd 镜像——外网 e2e-local 全栈门禁恢复（外网无 @nce 真包，
      // eview 版构建必炸；antd 链路实证零 @nce 依赖）。eview 真验=内网 E2E。
      // 波 C：路由 compat 经裸别名单点切换（相对导入吃不到别名——@ui-backend
      // 同款教训）。现指 react 直通版；翻转日改指 compat.inula.ts。
      '@app-router': fileURLToPath(
        new URL(INULA ? './src/router/compat.inula.tsx' : './src/router/compat.ts', import.meta.url),
      ),
      '@ui-backend': fileURLToPath(
        new URL(process.env.USMP_UI_BACKEND === 'antd' ? './src/ui/antd-backend' : './src/ui/eview', import.meta.url),
      ),
    },
  },
  server: {
    port: 3000,
  },
})
