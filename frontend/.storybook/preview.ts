import type { Preview } from '@storybook/vue3'
import { setup } from '@storybook/vue3'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'

// 全局注册 Element Plus，使动态渲染组件在 story 中与真实应用一致地渲染。
setup((app) => {
  app.use(ElementPlus)
})

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
