import type { Preview } from '@storybook/vue3'
import { setup } from '@storybook/vue3'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import { i18n } from '../src/i18n'

// 全局注册 Element Plus 与 i18n，使动态渲染组件在 story 中与真实应用一致地渲染。
setup((app) => {
  app.use(ElementPlus)
  app.use(i18n)
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
