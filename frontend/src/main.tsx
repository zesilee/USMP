import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { UiProvider } from './ui'
import App from './App'
import './styles/reset.scss'
import './styles/theme.scss'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <UiProvider>
      <App />
    </UiProvider>
  </StrictMode>,
)
