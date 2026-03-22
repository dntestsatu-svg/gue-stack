import api from './http'
import type { ApiResponse, DashboardOverview, TransactionHistoryPage, TransactionHistoryQuery } from './types'

export async function fetchOverview() {
  const { data } = await api.get<ApiResponse<DashboardOverview>>('/api/v1/dashboard/overview')
  return data.data
}

export async function fetchHistory(query: TransactionHistoryQuery = {}) {
  const { data } = await api.get<ApiResponse<TransactionHistoryPage>>('/api/v1/transactions/history', {
    params: query,
  })
  return data.data
}

export async function exportHistory(format: 'csv' | 'docx', query: TransactionHistoryQuery = {}) {
  const { data, headers } = await api.get<globalThis.Blob>('/api/v1/transactions/history/export', {
    params: {
      ...query,
      format,
    },
    responseType: 'blob',
  })

  let fileName = `transaction-history.${format}`
  const contentDisposition = headers['content-disposition']
  if (typeof contentDisposition === 'string') {
    const match = /filename="?([^";]+)"?/i.exec(contentDisposition)
    if (match?.[1]) {
      fileName = match[1]
    }
  }

  return { blob: data, fileName }
}
