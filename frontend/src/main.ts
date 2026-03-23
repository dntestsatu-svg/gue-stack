import { createPinia } from 'pinia'
import { ViteSSG } from 'vite-ssg'

import App from './App.vue'
import { installRouterGuards, routes } from './router'
import './style.css'

export const createApp = ViteSSG(
  App,
  { routes },
  ({ app, router }) => {
    const pinia = createPinia()

    app.use(pinia)

    if (!import.meta.env.SSR) {
      installRouterGuards(router)
    }
  },
)
