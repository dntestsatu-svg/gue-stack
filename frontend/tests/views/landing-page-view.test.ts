import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createHead } from '@unhead/vue/client'
import { createMemoryHistory, createRouter } from 'vue-router'
import LandingPageView from '@/views/LandingPageView.vue'

describe('LandingPageView', () => {
  it('renders public SEO-oriented content and CTA links', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: LandingPageView },
        { path: '/login', component: { template: '<div>Login</div>' } },
        { path: '/register', component: { template: '<div>Register</div>' } },
        { path: '/dashboard', component: { template: '<div>Dashboard</div>' } },
      ],
    })

    await router.push('/')
    await router.isReady()

    const wrapper = mount(LandingPageView, {
      global: {
        plugins: [router, createHead()],
      },
    })

    expect(wrapper.text()).toContain('APIGOQR')
    expect(wrapper.text()).toContain('Gateway QRIS untuk merchant yang butuh orkestrasi stabil')
    expect(wrapper.text()).toContain('Webhook idempotent dan callback final')
    expect(wrapper.text()).toContain('Masuk untuk mulai integrasi')
    expect(wrapper.text()).toContain('Buat akun merchant')
  })
})
