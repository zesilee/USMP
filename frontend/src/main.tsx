import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider } from 'react-router'
import { UiProvider } from './ui'
import { router } from './router'
import './styles/reset.scss'
import './styles/theme.scss'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <UiProvider>
      <RouterProvider router={router} />
    </UiProvider>
  </StrictMode>,
)
