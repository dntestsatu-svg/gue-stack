import { beforeEach, describe, expect, it, vi } from 'vitest'
import { exportHistory, fetchHistory } from '@/services/dashboard'

const { getMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
}))

vi.mock('@/services/http', () => ({
  default: {
    get: getMock,
  },
}))

describe('dashboard service', () => {
  beforeEach(() => {
    getMock.mockReset()
  })

  it('requests paginated transaction history with query params', async () => {
    getMock.mockResolvedValue({
      data: {
        data: {
          items: [],
          total: 0,
          limit: 50,
          offset: 20,
          has_more: false,
        },
      },
    })

    const result = await fetchHistory({
      limit: 50,
      offset: 20,
      q: 'trx-1',
      from: '2026-03-20',
      to: '2026-03-21',
    })

    expect(getMock).toHaveBeenCalledWith('/api/v1/transactions/history', {
      params: {
        limit: 50,
        offset: 20,
        q: 'trx-1',
        from: '2026-03-20',
        to: '2026-03-21',
      },
    })
    expect(result.limit).toBe(50)
    expect(result.offset).toBe(20)
  })

  it('exports history and resolves filename from content-disposition', async () => {
    getMock.mockResolvedValue({
      data: new globalThis.Blob(['csv-data']),
      headers: {
        'content-disposition': 'attachment; filename="history-custom.csv"',
      },
    })

    const result = await exportHistory('csv', { q: 'success' })

    expect(getMock).toHaveBeenCalledWith('/api/v1/transactions/history/export', {
      params: {
        q: 'success',
        format: 'csv',
      },
      responseType: 'blob',
    })
    expect(result.fileName).toBe('history-custom.csv')
    expect(result.blob).toBeInstanceOf(globalThis.Blob)
  })
})
