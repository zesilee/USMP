import type { StorybookConfig } from '@storybook/vue3-vite'

// Storybook（Vue3 + Vite）—— YANG 模型驱动动态渲染组件的隔离开发/展示环境（R05）。
// 给 DynamicTable/FieldRenderer 喂各种 mock YANG field，无需起后端即可
// 开发、调参、回归其渲染。运行：npm run storybook（build：npm run build-storybook）。
const config: StorybookConfig = {
  stories: ['../src/**/*.stories.@(ts|tsx)'],
  framework: {
    name: '@storybook/vue3-vite',
    options: {},
  },
  core: {
    disableTelemetry: true,
  },
}

export default config
