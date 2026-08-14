import type { StorybookConfig } from '@storybook/react-vite'

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
}

export default config
