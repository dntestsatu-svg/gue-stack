import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import BankManagementView from '@/views/BankManagementView.vue'
import { useUserStore } from '@/stores/user'

const {
  listBanksMock,
  inquiryBankMock,
  createBankMock,
  removeBankMock,
  toastSuccessMock,
  toastErrorMock,
} = vi.hoisted(() => ({
  listBanksMock: vi.fn(),
  inquiryBankMock: vi.fn(),
  createBankMock: vi.fn(),
  removeBankMock: vi.fn(),
  toastSuccessMock: vi.fn(),
  toastErrorMock: vi.fn(),
}))

vi.mock('@/services/bank', () => ({
  list: listBanksMock,
  inquiry: inquiryBankMock,
  create: createBankMock,
  remove: removeBankMock,
  paymentOptions: vi.fn(),
}))

vi.mock('vue-sonner', () => ({
  toast: {
    success: toastSuccessMock,
    error: toastErrorMock,
  },
}))

describe('BankManagementView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    listBanksMock.mockReset()
    inquiryBankMock.mockReset()
    createBankMock.mockReset()
    removeBankMock.mockReset()
    toastSuccessMock.mockReset()
    toastErrorMock.mockReset()

    listBanksMock.mockResolvedValue({
      items: [
        {
          id: 10,
          payment_id: 8,
          bank_name: 'PT. BANK CENTRAL ASIA, TBK.',
          account_name: 'PT GUE CONTROL',
          account_number: '1234567890',
          created_at: '2026-03-22T10:00:00Z',
        },
      ],
      total: 1,
      limit: 10,
      offset: 0,
      has_more: false,
    })

    Object.assign(globalThis.navigator, {
      clipboard: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    })
  })

  it('renders paginated bank list for admin role', async () => {
    const userStore = useUserStore()
    userStore.setProfile({
      id: 11,
      name: 'Admin',
      email: 'admin@gue.local',
      role: 'admin',
      is_active: true,
    })

    const wrapper = mount(BankManagementView, {
      global: {
        stubs: {
          BankCatalogSelect: true,
        },
      },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Bank Management')
    expect(wrapper.text()).toContain('PT. BANK CENTRAL ASIA, TBK.')
    expect(wrapper.text()).toContain('Showing 1-1 of 1')
    expect(listBanksMock).toHaveBeenCalledWith({ limit: 10, offset: 0, q: undefined })
  })

  it('creates a bank using searchable bank selection flow', async () => {
    const userStore = useUserStore()
    userStore.setProfile({
      id: 11,
      name: 'Admin',
      email: 'admin@gue.local',
      role: 'admin',
      is_active: true,
    })
    inquiryBankMock.mockResolvedValue({
      payment_id: 8,
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
    createBankMock.mockResolvedValue({
      id: 22,
      payment_id: 8,
      bank_name: 'PT. BANK CENTRAL ASIA, TBK.',
      account_name: 'PT GUE CONTROL',
      account_number: '1234567890',
      created_at: '2026-03-22T10:10:00Z',
    })

    const wrapper = mount(BankManagementView, {
      global: {
        stubs: {
          BankCatalogSelect: {
            props: ['modelValue', 'selectedLabel'],
            emits: ['update:modelValue', 'select'],
            template: `
              <button
                data-testid="bank-catalog-select"
                @click="$emit('update:modelValue', 8); $emit('select', { id: 8, bank_name: 'PT. BANK CENTRAL ASIA, TBK.' })"
              >
                Select Bank
              </button>
            `,
          },
        },
      },
    })
    await flushPromises()

    const addButton = wrapper.findAll('button').find((button) => button.text().includes('Add Bank'))
    expect(addButton).toBeDefined()
    await addButton!.trigger('click')
    await flushPromises()

    const selectButton = document.body.querySelector('[data-testid="bank-catalog-select"]') as { click: () => void } | null
    expect(selectButton).toBeTruthy()
    selectButton!.click()

    const accountNumberInput = document.body.querySelector('#bank-account-number') as { value: string; dispatchEvent: (event: unknown) => boolean } | null
    expect(accountNumberInput).toBeTruthy()

    accountNumberInput!.value = '1234567890'
    accountNumberInput!.dispatchEvent(new window.Event('input'))
    await flushPromises()

    const saveButton = Array.from(document.body.querySelectorAll('button')).find((button) =>
      button.textContent?.includes('Save Bank'),
    )
    expect(saveButton).toBeTruthy()
    saveButton!.click()
    await flushPromises()

    expect(inquiryBankMock).toHaveBeenCalledWith({
      payment_id: 8,
      account_number: '1234567890',
    })
    expect(document.body.textContent).toContain('Apakah benar nama bank anda adalah')
    expect(document.body.textContent).toContain('"PT GUE CONTROL"')

    const confirmButton = Array.from(document.body.querySelectorAll('button')).find((button) =>
      button.textContent?.includes('Yes, Save Bank'),
    )
    expect(confirmButton).toBeTruthy()
    confirmButton!.click()
    await flushPromises()

    expect(createBankMock).toHaveBeenCalledWith({
      payment_id: 8,
      account_name: 'PT GUE CONTROL',
      account_number: '1234567890',
      inquiry_id: 88,
    })
    expect(toastSuccessMock).toHaveBeenCalled()
  })

  it('shows inquiry error when bank verification fails', async () => {
    const userStore = useUserStore()
    userStore.setProfile({
      id: 11,
      name: 'Admin',
      email: 'admin@gue.local',
      role: 'admin',
      is_active: true,
    })
    inquiryBankMock.mockRejectedValue(new Error('Invalid client'))

    const wrapper = mount(BankManagementView, {
      global: {
        stubs: {
          BankCatalogSelect: {
            props: ['modelValue', 'selectedLabel'],
            emits: ['update:modelValue', 'select'],
            template: `
              <button
                data-testid="bank-catalog-select"
                @click="$emit('update:modelValue', 8); $emit('select', { id: 8, bank_name: 'PT. BANK CENTRAL ASIA, TBK.' })"
              >
                Select Bank
              </button>
            `,
          },
        },
      },
    })
    await flushPromises()

    const addButton = wrapper.findAll('button').find((button) => button.text().includes('Add Bank'))
    expect(addButton).toBeDefined()
    await addButton!.trigger('click')
    await flushPromises()

    const selectButton = document.body.querySelector('[data-testid="bank-catalog-select"]') as { click: () => void } | null
    expect(selectButton).toBeTruthy()
    selectButton!.click()

    const accountNumberInput = document.body.querySelector('#bank-account-number') as { value: string; dispatchEvent: (event: unknown) => boolean } | null
    expect(accountNumberInput).toBeTruthy()

    accountNumberInput!.value = '1234567890'
    accountNumberInput!.dispatchEvent(new window.Event('input'))
    await flushPromises()

    const saveButton = Array.from(document.body.querySelectorAll('button')).find((button) =>
      button.textContent?.includes('Save Bank'),
    )
    expect(saveButton).toBeTruthy()
    saveButton!.click()
    await flushPromises()

    expect(inquiryBankMock).toHaveBeenCalled()
    expect(createBankMock).not.toHaveBeenCalled()
    expect(toastErrorMock).toHaveBeenCalledWith('Invalid client')
  })

  it('hides bank management action for role user', async () => {
    const userStore = useUserStore()
    userStore.setProfile({
      id: 91,
      name: 'Basic User',
      email: 'user@gue.local',
      role: 'user',
      is_active: true,
    })

    const wrapper = mount(BankManagementView, {
      global: {
        stubs: {
          BankCatalogSelect: true,
        },
      },
    })
    await flushPromises()

    expect(wrapper.text()).not.toContain('Add Bank')
  })
})
