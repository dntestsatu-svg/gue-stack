import api from './http'
import type {
  ApiResponse,
  DashboardOverview,
  DashboardStatusSeriesPoint,
  TransactionHistoryItem,
  TransactionHistoryPage,
  TransactionHistoryQuery,
} from './types'

type UnknownRecord = Record<string, unknown>

function toRecord(value: unknown): UnknownRecord {
  return typeof value === 'object' && value !== null ? (value as UnknownRecord) : {}
}

function toString(value: unknown, fallback = ''): string {
  return typeof value === 'string' ? value : fallback
}

function toNumber(value: unknown, fallback = 0): number {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value
  }
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value)
    if (Number.isFinite(parsed)) {
      return parsed
    }
  }
  return fallback
}

function toBoolean(value: unknown, fallback = false): boolean {
  return typeof value === 'boolean' ? value : fallback
}

function toArray<T>(value: unknown, mapper: (item: unknown) => T): T[] {
  if (!Array.isArray(value)) {
    return []
  }
  return value.map(mapper)
}

function unwrapApiData<T>(response: ApiResponse<T> | UnknownRecord): T | UnknownRecord {
  if (response && typeof response === 'object' && 'data' in response) {
    return (response as ApiResponse<T>).data
  }
  return response as UnknownRecord
}

function normalizeStatusSeriesPoint(item: unknown): DashboardStatusSeriesPoint {
  const record = toRecord(item)
  return {
    bucket: toString(record.bucket),
    success_count: toNumber(record.success_count),
    failed_expired_count: toNumber(record.failed_expired_count),
  }
}

function normalizeHistoryItem(item: unknown): TransactionHistoryItem {
  const record = toRecord(item)
  const rawType = toString(record.type, 'deposit')
  return {
    id: toNumber(record.id),
    toko_id: toNumber(record.toko_id),
    toko_name: toString(record.toko_name),
    player: toString(record.player) || undefined,
    code: toString(record.code) || undefined,
    type: rawType === 'withdraw' ? 'withdraw' : 'deposit',
    status: toString(record.status),
    reference: toString(record.reference) || undefined,
    amount: toNumber(record.amount),
    netto: toNumber(record.netto),
    created_at: toString(record.created_at),
  }
}

function normalizeOverview(payload: unknown): DashboardOverview {
  const record = toRecord(payload)
  const metrics = toRecord(record.metrics)
  const externalBalance = toRecord(record.external_balance)

  return {
    window_hours: toNumber(record.window_hours, 12),
    can_view_project_profit: toBoolean(record.can_view_project_profit, false),
    metrics: {
      total_transactions: toNumber(metrics.total_transactions),
      success_transactions: toNumber(metrics.success_transactions),
      pending_transactions: toNumber(metrics.pending_transactions),
      failed_transactions: toNumber(metrics.failed_transactions),
      success_rate: toNumber(metrics.success_rate),
      success_deposit: toNumber(metrics.success_deposit),
      success_withdraw: toNumber(metrics.success_withdraw),
      net_flow: toNumber(metrics.net_flow),
      project_profit: toNumber(metrics.project_profit),
    },
    status_series: toArray(record.status_series, normalizeStatusSeriesPoint),
    latest_success_orders: toArray(record.latest_success_orders, normalizeHistoryItem),
    external_balance: {
      pending_balance: toNumber(externalBalance.pending_balance),
      available_balance: toNumber(externalBalance.available_balance),
    },
    external_balance_error: toString(record.external_balance_error) || undefined,
    updated_at: toString(record.updated_at),
  }
}

function normalizeHistoryPage(payload: unknown): TransactionHistoryPage {
  const record = toRecord(payload)
  return {
    items: toArray(record.items, normalizeHistoryItem),
    total: toNumber(record.total),
    limit: toNumber(record.limit, 20),
    offset: toNumber(record.offset, 0),
    has_more: toBoolean(record.has_more),
  }
}

export async function fetchOverview() {
  const { data } = await api.get<ApiResponse<DashboardOverview> | UnknownRecord>('/api/v1/dashboard/overview')
  return normalizeOverview(unwrapApiData<DashboardOverview>(data))
}

export async function fetchHistory(query: TransactionHistoryQuery = {}) {
  const { data } = await api.get<ApiResponse<TransactionHistoryPage> | UnknownRecord>('/api/v1/transactions/history', {
    params: query,
  })
  return normalizeHistoryPage(unwrapApiData<TransactionHistoryPage>(data))
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
