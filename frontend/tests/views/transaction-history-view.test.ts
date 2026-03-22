import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import TransactionHistoryView from '@/views/TransactionHistoryView.vue'

const { fetchHistoryMock, exportHistoryMock } = vi.hoisted(() => ({
  fetchHistoryMock: vi.fn(),
  exportHistoryMock: vi.fn(),
}))

vi.mock('@/services/dashboard', () => ({
  fetchHistory: fetchHistoryMock,
  exportHistory: exportHistoryMock,
}))

describe('TransactionHistoryView', () => {
  beforeEach(() => {
    fetchHistoryMock.mockReset()
    exportHistoryMock.mockReset()
  })

  it('renders history table with server-side payload', async () => {
    fetchHistoryMock.mockResolvedValue({
      items: [
        {
          id: 10,
          toko_id: 7,
          toko_name: 'Toko Alpha',
          player: 'player-1',
          code: 'A1',
          type: 'deposit',
          status: 'success',
          reference: 'REF-001',
          amount: 50000,
          netto: 48500,
          created_at: '2026-03-21T10:00:00Z',
        },
      ],
      total: 1,
      limit: 20,
      offset: 0,
      has_more: false,
    })

    const wrapper = mount(TransactionHistoryView, {
      global: {
        stubs: {
          DateRangePicker: {
            template: '<div data-testid="date-range-picker-stub" />',
          },
        },
      },
    })

    await Promise.resolve()
    await Promise.resolve()

    expect(wrapper.text()).toContain('Histori Transaksi')
    expect(wrapper.text()).toContain('Latest Order (All Status)')
    expect(wrapper.text()).toContain('Toko Alpha')
    expect(fetchHistoryMock).toHaveBeenCalledTimes(1)
  })
})
