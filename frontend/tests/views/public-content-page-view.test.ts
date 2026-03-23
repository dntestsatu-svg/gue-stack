import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createHead } from '@unhead/vue/client'
import { createMemoryHistory, createRouter } from 'vue-router'
import PublicContentPageView from '@/views/PublicContentPageView.vue'

describe('PublicContentPageView', () => {
  it('renders a public merchant SEO page from the content catalog', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/fitur-qris-merchant', component: PublicContentPageView },
      ],
    })

    await router.push('/fitur-qris-merchant')
    await router.isReady()

    const wrapper = mount(PublicContentPageView, {
      global: {
        plugins: [router, createHead()],
      },
    })

    expect(wrapper.text()).toContain('Fitur gateway QRIS merchant yang dirancang untuk alur final dan bisa diaudit.')
    expect(wrapper.text()).toContain('Generate QRIS yang menyimpan transaksi lokal lebih dulu')
    expect(wrapper.text()).toContain('Webhook final dan callback merchant yang lebih stabil')
    expect(wrapper.text()).toContain('Kontrol Balance dan Withdraw')
  })
})
