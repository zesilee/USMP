import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import './styles/variables.scss'
import './styles/reset.scss'
import './styles/theme.scss'
import App from './App.vue'

const app = createApp(App)
app.use(ElementPlus)
app.mount('#app')
