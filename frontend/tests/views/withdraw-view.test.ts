import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import WithdrawView from '@/views/WithdrawView.vue'

const {
  fetchWithdrawOptionsMock,
  inquiryWithdrawMock,
  transferWithdrawMock,
  toastSuccessMock,
  toastErrorMock,
} = vi.hoisted(() => ({
  fetchWithdrawOptionsMock: vi.fn(),
  inquiryWithdrawMock: vi.fn(),
  transferWithdrawMock: vi.fn(),
  toastSuccessMock: vi.fn(),
  toastErrorMock: vi.fn(),
}))

vi.mock('@/services/withdraw', () => ({
  fetchOptions: fetchWithdrawOptionsMock,
  inquiry: inquiryWithdrawMock,
  transfer: transferWithdrawMock,
}))

vi.mock('vue-sonner', () => ({
  toast: {
    success: toastSuccessMock,
    error: toastErrorMock,
  },
}))

describe('WithdrawView', () => {
  beforeEach(() => {
    fetchWithdrawOptionsMock.mockReset()
    inquiryWithdrawMock.mockReset()
    transferWithdrawMock.mockReset()
    toastSuccessMock.mockReset()
    toastErrorMock.mockReset()

    fetchWithdrawOptionsMock.mockResolvedValue({
      tokos: [
        {
          id: 1,
          name: 'Toko Alpha',
          settlement_balance: 500000,
          available_balance: 900000,
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
  })

  it('renders withdraw workspace and chains inquiry to transfer', async () => {
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
      settlement_balance: 500000,
      remaining_settlement_balance: 400000,
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
      remaining_settlement_balance: 400000,
    })

    const wrapper = mount(WithdrawView)
    await flushPromises()

    expect(wrapper.text()).toContain('Withdraw')
    expect(wrapper.text()).toContain('Toko Alpha')
    expect(wrapper.text()).toContain('PT. BANK CENTRAL ASIA, TBK.')

    await wrapper.get('#withdraw-amount').setValue('100000')
    const actionButton = wrapper.findAll('button').find((button) => button.text().includes('Request Withdraw'))
    expect(actionButton).toBeDefined()
    await actionButton!.trigger('click')
    await flushPromises()

    expect(inquiryWithdrawMock).toHaveBeenCalledWith({
      toko_id: 1,
      bank_id: 8,
      amount: 100000,
    })
    expect(transferWithdrawMock).toHaveBeenCalledWith({
      toko_id: 1,
      bank_id: 8,
      amount: 100000,
      inquiry_id: 77,
    })
    expect(wrapper.text()).toContain('Uangnya akan segera sampai ke bank anda.')
    expect(toastSuccessMock).toHaveBeenCalled()
  })

  it('shows gateway error when inquiry or transfer fails', async () => {
    inquiryWithdrawMock.mockRejectedValue(new Error('Invalid client'))

    const wrapper = mount(WithdrawView)
    await flushPromises()
    await wrapper.get('#withdraw-amount').setValue('100000')

    const actionButton = wrapper.findAll('button').find((button) => button.text().includes('Request Withdraw'))
    expect(actionButton).toBeDefined()
    await actionButton!.trigger('click')
    await flushPromises()

    expect(transferWithdrawMock).not.toHaveBeenCalled()
    expect(toastErrorMock).toHaveBeenCalledWith('Invalid client')
    expect(wrapper.text()).toContain('Invalid client')
  })
})
