import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { fileURLToPath } from 'node:url'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      // 生产真身 = EviewUI 桥（组 5 接线单点切换；测试侧的同名别名见
      // vitest.config.ts——外网映射 antd 镜像）。组 7.2：USMP_UI_BACKEND=antd
      // 时构建走 antd 镜像——外网 e2e-local 全栈门禁恢复（外网无 @nce 真包，
      // eview 版构建必炸；antd 链路实证零 @nce 依赖）。eview 真验=内网 E2E。
      '@ui-backend': fileURLToPath(
        new URL(process.env.USMP_UI_BACKEND === 'antd' ? './src/ui/antd-backend' : './src/ui/eview', import.meta.url),
      ),
    },
  },
  server: {
    port: 3000,
  },
})
