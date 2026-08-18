import type { StorybookConfig } from '@storybook/react-vite'
import { fileURLToPath } from 'node:url'

// Storybook（React + Vite）—— YANG 模型驱动动态渲染组件的隔离开发/展示环境（R05）。
// 给 FieldRenderer 等组件喂各种 mock YANG field，无需起后端即可开发、调参、
// 回归其渲染。运行：npm run storybook（build：npm run build-storybook）。
// 旧 Vue 栈故事内容不迁移（frontend-react-antd-switch Non-Goal），按需重建。
const config: StorybookConfig = {
  stories: ['../src/**/*.stories.@(ts|tsx)'],
  framework: {
    name: '@storybook/react-vite',
    options: {},
  },
  core: {
    disableTelemetry: true,
  },
  // 组 5 接线：外网无 EviewUI 真包——storybook 构建与 vitest 同口径走
  // antd 测试镜像（@ui-backend）+ @nce 空 stub（antd-backend/README；
  // 组 8 工具链收尾复核）。
  viteFinal: async (cfg) => {
    cfg.resolve = cfg.resolve ?? {}
    const alias = Array.isArray(cfg.resolve.alias) ? cfg.resolve.alias : []
    cfg.resolve.alias = [
      ...alias,
      { find: '@ui-backend', replacement: fileURLToPath(new URL('../src/ui/antd-backend', import.meta.url)) },
      { find: /^@nce\/eview-react\/locales\/.+$/, replacement: fileURLToPath(new URL('../test/stubs/eview-locales.ts', import.meta.url)) },
      { find: /^@nce\/eview-react\/([^/]+)$/, replacement: fileURLToPath(new URL('../test/stubs/eview/$1.ts', import.meta.url)) },
      { find: /^@nce\/icon-plus(\/.*)?$/, replacement: fileURLToPath(new URL('../test/stubs/eview-empty.ts', import.meta.url)) },
    ]
    return cfg
  },
}

export default config
