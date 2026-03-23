import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ApiDocumentationView from '@/views/ApiDocumentationView.vue'

describe('ApiDocumentationView', () => {
  it('renders merchant integration documentation', () => {
    const wrapper = mount(ApiDocumentationView)

    expect(wrapper.text()).toContain('Dokumentasi API')
    expect(wrapper.text()).toContain('Merchant Endpoint Catalog')
    expect(wrapper.text()).toContain('Generate QRIS Example')
    expect(wrapper.text()).toContain('Callback Payload ke Merchant Website')
    expect(wrapper.text()).toContain('Financial Rules yang Berlaku')
    expect(wrapper.text()).toContain('3%')
    expect(wrapper.text()).toContain('Deposit fee')
  })
})
