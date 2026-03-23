import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import App from '@/App.vue'
import ApiDocumentationView from '@/views/ApiDocumentationView.vue'
import WithdrawView from '@/views/WithdrawView.vue'
import DashboardView from '@/views/DashboardView.vue'
import BankManagementView from '@/views/BankManagementView.vue'
import LoginView from '@/views/LoginView.vue'
import TestingView from '@/views/TestingView.vue'
import TokoView from '@/views/TokoView.vue'
import TransactionHistoryView from '@/views/TransactionHistoryView.vue'
import UserManagementView from '@/views/UserManagementView.vue'
import { useAuthStore } from '@/stores/auth'
import { useUserStore } from '@/stores/user'

const {
  initCsrfMock,
  loginMock,
  refreshMock,
  logoutMock,
  hasSessionHintMock,
  fetchOverviewMock,
  fetchHistoryMock,
  exportHistoryMock,
  fetchWorkspaceMock,
  fetchTokosMock,
  applySettlementMock,
  createTokoMock,
  updateTokoMock,
  regenerateTokoTokenMock,
  generateTestingQrisMock,
  checkCallbackReadinessMock,
  listBanksMock,
  inquiryBankMock,
  createBankMock,
  removeBankMock,
  fetchWithdrawOptionsMock,
  fetchWithdrawHistoryMock,
  inquiryWithdrawMock,
  transferWithdrawMock,
  listUsersMock,
  meMock,
  createUserMock,
  updateRoleMock,
  changePasswordMock,
} = vi.hoisted(() => ({
  initCsrfMock: vi.fn(),
  loginMock: vi.fn(),
  refreshMock: vi.fn(),
  logoutMock: vi.fn(),
  hasSessionHintMock: vi.fn(() => false),
  fetchOverviewMock: vi.fn(),
  fetchHistoryMock: vi.fn(),
  exportHistoryMock: vi.fn(),
  fetchWorkspaceMock: vi.fn(),
  fetchTokosMock: vi.fn(),
  applySettlementMock: vi.fn(),
  createTokoMock: vi.fn(),
  updateTokoMock: vi.fn(),
  regenerateTokoTokenMock: vi.fn(),
  generateTestingQrisMock: vi.fn(),
  checkCallbackReadinessMock: vi.fn(),
  listBanksMock: vi.fn(),
  inquiryBankMock: vi.fn(),
  createBankMock: vi.fn(),
  removeBankMock: vi.fn(),
  fetchWithdrawOptionsMock: vi.fn(),
  fetchWithdrawHistoryMock: vi.fn(),
  inquiryWithdrawMock: vi.fn(),
  transferWithdrawMock: vi.fn(),
  listUsersMock: vi.fn(),
  meMock: vi.fn(),
  createUserMock: vi.fn(),
  updateRoleMock: vi.fn(),
  changePasswordMock: vi.fn(),
}))

vi.mock('@/services/auth', () => ({
  initCsrf: initCsrfMock,
  login: loginMock,
  refresh: refreshMock,
  logout: logoutMock,
  hasSessionHint: hasSessionHintMock,
}))

vi.mock('@/services/dashboard', () => ({
  fetchOverview: fetchOverviewMock,
  fetchHistory: fetchHistoryMock,
  exportHistory: exportHistoryMock,
}))

vi.mock('@/services/toko', () => ({
  fetchWorkspace: fetchWorkspaceMock,
  fetchTokos: fetchTokosMock,
  applySettlement: applySettlementMock,
  createToko: createTokoMock,
  updateToko: updateTokoMock,
  regenerateTokoToken: regenerateTokoTokenMock,
}))

vi.mock('@/services/testing', () => ({
  generateQris: generateTestingQrisMock,
  checkCallbackReadiness: checkCallbackReadinessMock,
}))

vi.mock('@/services/bank', () => ({
  list: listBanksMock,
  inquiry: inquiryBankMock,
  create: createBankMock,
  remove: removeBankMock,
  paymentOptions: vi.fn(),
}))

vi.mock('@/services/withdraw', () => ({
  fetchOptions: fetchWithdrawOptionsMock,
  fetchHistory: fetchWithdrawHistoryMock,
  inquiry: inquiryWithdrawMock,
  transfer: transferWithdrawMock,
}))

vi.mock('@/services/user', () => ({
  list: listUsersMock,
  me: meMock,
  create: createUserMock,
  updateRole: updateRoleMock,
  changePassword: changePasswordMock,
}))

vi.mock('@/composables/usePolling', () => ({
  usePolling: (task: () => Promise<void>) => {
    void task()
    return {
      runNow: task,
    }
  },
}))

function createTestRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      {
        path: '/login',
        name: 'login',
        component: LoginView,
        meta: { authLayout: true, title: 'Sign In' },
      },
      {
        path: '/dashboard',
        name: 'dashboard',
        component: DashboardView,
        meta: { requiresAuth: true, requiresActive: true, title: 'Dashboard' },
      },
      {
        path: '/histori-transaksi',
        name: 'histori-transaksi',
        component: TransactionHistoryView,
        meta: { requiresAuth: true, requiresActive: true, title: 'Histori Transaksi' },
      },
      {
        path: '/toko',
        name: 'toko',
        component: TokoView,
        meta: { requiresAuth: true, requiresActive: true, title: 'Manajemen Toko' },
      },
      {
        path: '/testing',
        name: 'testing',
        component: TestingView,
        meta: { requiresAuth: true, requiresActive: true, title: 'Testing Toko' },
      },
      {
        path: '/dokumentasi-api',
        name: 'dokumentasi-api',
        component: ApiDocumentationView,
        meta: { requiresAuth: true, requiresActive: true, title: 'Dokumentasi API' },
      },
      {
        path: '/bank-management',
        name: 'bank-management',
        component: BankManagementView,
        meta: { requiresAuth: true, requiresActive: true, title: 'Bank Management' },
      },
      {
        path: '/withdraw',
        name: 'withdraw',
        component: WithdrawView,
        meta: { requiresAuth: true, requiresActive: true, title: 'Withdraw' },
      },
      {
        path: '/users',
        name: 'users',
        component: UserManagementView,
        meta: { requiresAuth: true, requiresActive: true, title: 'User Management' },
      },
    ],
  })
}

function mockDashboardOverview() {
  fetchOverviewMock.mockResolvedValue({
    window_hours: 12,
    can_view_project_profit: true,
    can_view_external_balance: true,
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
}

function seedAuthenticatedDev() {
  const auth = useAuthStore()
  const userStore = useUserStore()

  auth.ready = true
  auth.authenticated = true
  vi.spyOn(auth, 'restoreSession').mockResolvedValue(true)

  userStore.setProfile({
    id: 1,
    name: 'Developer',
    email: 'dev@gue.local',
    role: 'dev',
    is_active: true,
  })
}

describe('App SPA routing', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()

    initCsrfMock.mockResolvedValue(undefined)
    logoutMock.mockResolvedValue(undefined)
    hasSessionHintMock.mockReturnValue(false)

    mockDashboardOverview()
    fetchHistoryMock.mockResolvedValue({
      items: [],
      total: 0,
      limit: 20,
      offset: 0,
      has_more: false,
    })
    fetchWorkspaceMock.mockResolvedValue({
      items: [],
      summary: {
        total_tokos: 0,
        total_pending_balance: 0,
        total_settle_balance: 0,
      },
      total: 0,
      limit: 10,
      offset: 0,
      has_more: false,
    })
    fetchTokosMock.mockResolvedValue([
      {
        id: 1,
        name: 'Toko Alpha',
        token: 'tok_alpha',
        charge: 3,
        callback_url: 'https://merchant.example.com/callback',
      },
    ])
    listUsersMock.mockResolvedValue({
      items: [],
      total: 0,
      limit: 10,
      offset: 0,
      has_more: false,
    })
    listBanksMock.mockResolvedValue({
      items: [],
      total: 0,
      limit: 10,
      offset: 0,
      has_more: false,
    })
    inquiryBankMock.mockResolvedValue({
      payment_id: 1,
      account_number: '1234567890',
      account_name: 'PT GUE CONTROL',
      bank_code: '014',
      bank_name: 'PT. BANK CENTRAL ASIA, TBK.',
      partner_ref_no: 'partner-ref',
      vendor_ref_no: '',
      amount: 10000,
      fee: 1500,
      inquiry_id: 88,
    })
    fetchWithdrawOptionsMock.mockResolvedValue({
      tokos: [
        {
          id: 1,
          name: 'Toko Alpha',
          pending_balance: 900000,
          settle_balance: 500000,
        },
      ],
      banks: [
        {
          id: 8,
          bank_name: 'PT. BANK CENTRAL ASIA, TBK.',
          account_name: 'PT GUE CONTROL',
          account_number: '1234567890',
        },
      ],
    })
    fetchWithdrawHistoryMock.mockResolvedValue({
      items: [
        {
          id: 11,
          toko_id: 1,
          toko_name: 'Toko Alpha',
          status: 'pending',
          reference: 'partner-ref-11',
          amount: 50000,
          netto: 48500,
          created_at: '2026-03-21T06:20:00Z',
        },
      ],
      total: 1,
      limit: 10,
      offset: 0,
      has_more: false,
    })
    inquiryWithdrawMock.mockResolvedValue({
      toko_id: 1,
      toko_name: 'Toko Alpha',
      bank_id: 8,
      bank_name: 'PT. BANK CENTRAL ASIA, TBK.',
      account_name: 'PT GUE CONTROL',
      account_number: '1234567890',
      amount: 100000,
      fee: 1500,
      inquiry_id: 77,
      partner_ref_no: 'partner-ref-1',
      settle_balance: 500000,
      remaining_settle_balance: 400000,
    })
    transferWithdrawMock.mockResolvedValue({
      status: true,
      message: 'Uangnya akan segera sampai ke bank anda.',
      toko_id: 1,
      toko_name: 'Toko Alpha',
      bank_id: 8,
      bank_name: 'PT. BANK CENTRAL ASIA, TBK.',
      account_name: 'PT GUE CONTROL',
      account_number: '1234567890',
      amount: 100000,
      remaining_settle_balance: 400000,
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('logs in and renders dashboard actions without runtime errors', async () => {
    loginMock.mockResolvedValue({
      user: {
        id: 1,
        name: 'Developer',
        email: 'dev@gue.local',
        role: 'dev',
        is_active: true,
      },
      expires_in: 900,
      csrf_token: 'csrf-token',
    })

    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

    const pinia = createPinia()
    setActivePinia(pinia)
    const router = createTestRouter()
    await router.push('/login')
    await router.isReady()

    const wrapper = mount(App, {
      global: {
        plugins: [pinia, router],
        stubs: {
          DashboardStatusAreaChart: {
            template: '<div data-testid="dashboard-status-chart-stub">Status Chart</div>',
          },
          DateRangePicker: {
            template: '<div data-testid="date-range-picker-stub" />',
          },
        },
      },
    })

    await wrapper.get('input[type="email"]').setValue('dev@gue.local')
    await wrapper.get('input[type="password"]').setValue('aa123123')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/dashboard')
    expect(wrapper.text()).toContain('Add User')
    expect(wrapper.text()).toContain('List Users')
    expect(wrapper.text()).toContain('Enterprise Operations Dashboard')
    expect(consoleErrorSpy).not.toHaveBeenCalled()
  })

  it('renders route-specific content when SPA navigates between dashboard pages', async () => {
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

    const pinia = createPinia()
    setActivePinia(pinia)
    seedAuthenticatedDev()

    const router = createTestRouter()
    await router.push('/dashboard')
    await router.isReady()

    const wrapper = mount(App, {
      global: {
        plugins: [pinia, router],
        stubs: {
          DashboardStatusAreaChart: {
            template: '<div data-testid="dashboard-status-chart-stub">Status Chart</div>',
          },
          DateRangePicker: {
            template: '<div data-testid="date-range-picker-stub" />',
          },
        },
      },
    })

    await flushPromises()
    expect(wrapper.text()).toContain('Add User')
    expect(wrapper.text()).toContain('List Users')

    await router.push('/histori-transaksi')
    await flushPromises()
    expect(wrapper.text()).toContain('Histori Transaksi')
    expect(wrapper.text()).toContain('Filters')
    expect(wrapper.text()).toContain('Export CSV')

    await router.push('/toko')
    await flushPromises()
    expect(wrapper.text()).toContain('Manage Toko & Settlement')
    expect(wrapper.text()).toContain('Create Toko')

    await router.push('/testing')
    await flushPromises()
    expect(wrapper.text()).toContain('Testing Toko')
    expect(wrapper.text()).toContain('Generate QRIS')
    expect(wrapper.text()).toContain('Callback Readiness')

    await router.push('/dokumentasi-api')
    await flushPromises()
    expect(wrapper.text()).toContain('Dokumentasi API')
    expect(wrapper.text()).toContain('Merchant Endpoint Catalog')
    expect(wrapper.text()).toContain('Callback Payload ke Merchant Website')

    await router.push('/bank-management')
    await flushPromises()
    expect(wrapper.text()).toContain('Bank Management')
    expect(wrapper.text()).toContain('Add Bank')

    await router.push('/withdraw')
    await flushPromises()
    expect(wrapper.text()).toContain('Withdraw')
    expect(wrapper.text()).toContain('Request Withdraw')
    expect(wrapper.text()).toContain('Withdraw Request History')

    await router.push('/users')
    await flushPromises()
    expect(wrapper.text()).toContain('User Management')
    expect(wrapper.text()).toContain('Add User')
    expect(consoleErrorSpy).not.toHaveBeenCalled()
  }, 15000)
})
