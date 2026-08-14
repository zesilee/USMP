import type { Preview } from '@storybook/react-vite'
import '../src/styles/reset.scss'
import '../src/styles/theme.scss'

// 全局样式与真实应用一致；antd 组件经 src/ui 适配层导入，无需全局注册。
const preview: Preview = {
  parameters: {
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },
  },
}

export default preview
