import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import DashboardView from '@/views/DashboardView.vue'
import { useUserStore } from '@/stores/user'

const { pushMock, fetchOverviewMock } = vi.hoisted(() => ({
  pushMock: vi.fn(),
  fetchOverviewMock: vi.fn(),
}))

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRouter: () => ({ push: pushMock }),
  }
})

vi.mock('@/services/dashboard', () => ({
  fetchOverview: fetchOverviewMock,
}))

vi.mock('@/composables/usePolling', () => ({
  usePolling: (task: () => Promise<void>) => {
    void task()
    return { runNow: task }
  },
}))

describe('DashboardView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    fetchOverviewMock.mockReset()
    pushMock.mockReset()
  })

  it('renders dashboard cards and transaction table', async () => {
    fetchOverviewMock.mockResolvedValue({
      window_hours: 12,
      can_view_project_profit: true,
      metrics: {
        total_transactions: 10,
        success_transactions: 8,
        pending_transactions: 1,
        failed_transactions: 1,
        success_rate: 80,
        success_deposit: 100000,
        success_withdraw: 25000,
        net_flow: 75000,
        project_profit: 3000,
      },
      status_series: [
        { bucket: '2026-03-21T06:00:00Z', success_count: 3, failed_expired_count: 1 },
      ],
      latest_success_orders: [
        {
          id: 1,
          toko_id: 99,
          toko_name: 'Toko Alpha',
          status: 'success',
          type: 'deposit',
          reference: 'REF-1',
          amount: 50000,
          netto: 48500,
          created_at: '2026-03-21T06:10:00Z',
        },
      ],
      external_balance: {
        pending_balance: 2000,
        available_balance: 9000,
      },
      external_balance_error: '',
      updated_at: '2026-03-21T06:20:00Z',
    })

    const userStore = useUserStore()
    userStore.setProfile({
      id: 1,
      name: 'Developer',
      email: 'dev@gue.local',
      role: 'dev',
      is_active: true,
    })

    const wrapper = mount(DashboardView)
    await Promise.resolve()
    await Promise.resolve()

    expect(wrapper.text()).toContain('Enterprise Operations Dashboard')
    expect(wrapper.text()).toContain('Success Rate')
    expect(wrapper.text()).toContain('Latest Order (Success)')
    expect(wrapper.text()).toContain('Toko Alpha')
  })
})
