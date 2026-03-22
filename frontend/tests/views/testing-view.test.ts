import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import TestingView from '@/views/TestingView.vue'

const {
  fetchTokosMock,
  generateQrisMock,
  checkCallbackReadinessMock,
  toastSuccessMock,
  toastErrorMock,
} = vi.hoisted(() => ({
  fetchTokosMock: vi.fn(),
  generateQrisMock: vi.fn(),
  checkCallbackReadinessMock: vi.fn(),
  toastSuccessMock: vi.fn(),
  toastErrorMock: vi.fn(),
}))

vi.mock('@/services/toko', () => ({
  fetchTokos: fetchTokosMock,
}))

vi.mock('@/services/testing', () => ({
  generateQris: generateQrisMock,
  checkCallbackReadiness: checkCallbackReadinessMock,
}))

vi.mock('vue-sonner', () => ({
  toast: {
    success: toastSuccessMock,
    error: toastErrorMock,
  },
}))

describe('TestingView', () => {
  beforeEach(() => {
    fetchTokosMock.mockReset()
    generateQrisMock.mockReset()
    checkCallbackReadinessMock.mockReset()
    toastSuccessMock.mockReset()
    toastErrorMock.mockReset()

    fetchTokosMock.mockResolvedValue([
      {
        id: 1,
        name: 'Toko Alpha',
        token: 'tok_alpha',
        charge: 3,
        callback_url: 'https://merchant.example.com/callback',
      },
    ])
  })

  it('renders testing workspace and runs qris + callback checks', async () => {
    generateQrisMock.mockResolvedValue({
      toko_id: 1,
      toko_name: 'Toko Alpha',
      data: 'qr-data',
      trx_id: 'trx-001',
      expired_at: 1770000000,
      server_processing_ms: 18,
    })
    checkCallbackReadinessMock.mockResolvedValue({
      toko_id: 1,
      toko_name: 'Toko Alpha',
      callback_url: 'https://merchant.example.com/callback',
      ready: true,
      message: 'API kamu sudah ready.',
      detail: 'Callback URL merespons sesuai kontrak integrasi.',
      status_code: 200,
      received_success: true,
      callback_latency_ms: 42,
      server_processing_ms: 11,
    })

    const wrapper = mount(TestingView)
    await flushPromises()

    expect(wrapper.text()).toContain('Testing Toko')
    expect(wrapper.text()).toContain('Generate QRIS')
    expect(wrapper.text()).toContain('Callback Readiness')
    expect(wrapper.text()).toContain('Toko Alpha')

    await wrapper.get('#testing-username').setValue('player-01')
    await wrapper.get('#testing-amount').setValue('25000')
    const generateButton = wrapper.findAll('button').find((button) => button.text().includes('Generate QRIS'))
    expect(generateButton).toBeDefined()
    await generateButton!.trigger('click')
    await flushPromises()

    expect(generateQrisMock).toHaveBeenCalledWith({
      toko_id: 1,
      username: 'player-01',
      amount: 25000,
      expire: 300,
      custom_ref: undefined,
    })
    expect(wrapper.text()).toContain('trx-001')
    expect(wrapper.text()).toContain('qr-data')

    const callbackButton = wrapper.findAll('button').find((button) => button.text().includes('Check Callback Readiness'))
    expect(callbackButton).toBeDefined()
    await callbackButton!.trigger('click')
    await flushPromises()

    expect(checkCallbackReadinessMock).toHaveBeenCalledWith({ toko_id: 1 })
    expect(wrapper.text()).toContain('API kamu sudah ready.')
    expect(wrapper.text()).toContain('42 ms')
  })
})
