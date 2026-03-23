import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createHead } from '@unhead/vue/client'
import { createMemoryHistory, createRouter } from 'vue-router'
import LandingPageView from '@/views/LandingPageView.vue'

const { ensureGtmLoadedMock, trackNavigationWithGtmMock } = vi.hoisted(() => ({
  ensureGtmLoadedMock: vi.fn(),
  trackNavigationWithGtmMock: vi.fn(),
}))

vi.mock('@/lib/gtm', () => ({
  ensureGtmLoaded: ensureGtmLoadedMock,
  trackNavigationWithGtm: trackNavigationWithGtmMock,
}))

describe('LandingPageView', () => {
  beforeEach(() => {
    ensureGtmLoadedMock.mockReset()
    trackNavigationWithGtmMock.mockReset()
  })

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
    expect(ensureGtmLoadedMock).toHaveBeenCalledOnce()
  })

  it('tracks landing CTA clicks with destination metadata', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: LandingPageView }],
    })

    await router.push('/')
    await router.isReady()

    const wrapper = mount(LandingPageView, {
      global: {
        plugins: [router, createHead()],
      },
    })

    await wrapper.get('a[href="/register"]').trigger('click')

    expect(trackNavigationWithGtmMock).toHaveBeenCalledWith(
      {
        event: 'landing_cta_click',
        cta_name: 'register',
        cta_location: 'hero',
        destination_path: '/register',
        page_type: 'landing',
      },
      '/register',
    )
  })
})
