import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import TokoView from '@/views/TokoView.vue'
import { useUserStore } from '@/stores/user'

const { fetchBalancesMock, fetchTokosMock, applySettlementMock, createTokoMock } = vi.hoisted(() => ({
  fetchBalancesMock: vi.fn(),
  fetchTokosMock: vi.fn(),
  applySettlementMock: vi.fn(),
  createTokoMock: vi.fn(),
}))

vi.mock('@/services/toko', () => ({
  fetchBalances: fetchBalancesMock,
  fetchTokos: fetchTokosMock,
  applySettlement: applySettlementMock,
  createToko: createTokoMock,
}))

vi.mock('@/composables/usePolling', () => ({
  usePolling: (task: () => Promise<void>) => {
    void task()
    return { runNow: task }
  },
}))

describe('TokoView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    fetchBalancesMock.mockReset()
    fetchTokosMock.mockReset()
    applySettlementMock.mockReset()
    createTokoMock.mockReset()
  })

  it('renders toko and settlement data for authorized role', async () => {
    fetchBalancesMock.mockResolvedValue([
      {
        toko_id: 1,
        toko_name: 'Toko Alpha',
        settlement_balance: 120000,
        available_balance: 450000,
        updated_at: '2026-03-21T10:00:00Z',
      },
    ])
    fetchTokosMock.mockResolvedValue([
      {
        id: 1,
        name: 'Toko Alpha',
        token: 'tok_abc',
        charge: 3,
        callback_url: 'https://example.com/callback',
      },
    ])

    const userStore = useUserStore()
    userStore.setProfile({
      id: 11,
      name: 'Admin',
      email: 'admin@gue.local',
      role: 'admin',
      is_active: true,
    })

    const wrapper = mount(TokoView)
    await flushPromises()

    expect(wrapper.text()).toContain('Manage Toko & Settlement')
    expect(wrapper.text()).toContain('Toko Alpha')
    expect(wrapper.text()).toContain('Settlement Balances')
    expect(fetchBalancesMock).toHaveBeenCalledTimes(1)
    expect(fetchTokosMock).toHaveBeenCalledTimes(1)
  })

  it('does not render create toko action for user role', async () => {
    fetchBalancesMock.mockResolvedValue([])
    fetchTokosMock.mockResolvedValue([])

    const userStore = useUserStore()
    userStore.setProfile({
      id: 99,
      name: 'Viewer User',
      email: 'viewer@gue.local',
      role: 'user',
      is_active: true,
    })

    const wrapper = mount(TokoView)
    await flushPromises()

    expect(wrapper.text()).not.toContain('Create Toko')
  })
})
