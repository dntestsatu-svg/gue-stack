export type UserRole = 'dev' | 'superadmin' | 'admin' | 'user'

export interface ApiResponse<T> {
  status: string
  data: T
  message?: string
}

export interface User {
  id: number
  name: string
  email: string
  role: UserRole
  is_active: boolean
}

export interface UserListPage {
  items: User[]
  total: number
  limit: number
  offset: number
  has_more: boolean
}

export interface UserListQuery {
  limit?: number
  offset?: number
  q?: string
  role?: UserRole
}

export interface AuthResponseData {
  user: User
  expires_in: number
  csrf_token: string
}

export interface DashboardMetrics {
  total_transactions: number
  success_transactions: number
  pending_transactions: number
  failed_transactions: number
  success_rate: number
  success_deposit: number
  success_withdraw: number
  net_flow: number
  project_profit: number
}

export interface DashboardStatusSeriesPoint {
  bucket: string
  success_count: number
  failed_expired_count: number
}

export interface DashboardExternalBalance {
  pending_balance: number
  available_balance: number
}

export interface DashboardOverview {
  window_hours: number
  can_view_project_profit: boolean
  metrics: DashboardMetrics
  status_series: DashboardStatusSeriesPoint[]
  latest_success_orders: TransactionHistoryItem[]
  external_balance: DashboardExternalBalance
  external_balance_error?: string
  updated_at: string
}

export interface TransactionHistoryItem {
  id: number
  toko_id: number
  toko_name: string
  player?: string
  code?: string
  type: 'deposit' | 'withdraw'
  status: string
  reference?: string
  amount: number
  netto: number
  created_at: string
}

export interface TransactionHistoryPage {
  items: TransactionHistoryItem[]
  total: number
  limit: number
  offset: number
  has_more: boolean
}

export interface TransactionHistoryQuery {
  limit?: number
  offset?: number
  q?: string
  from?: string
  to?: string
}

export interface TokoBalanceItem {
  toko_id: number
  toko_name: string
  settlement_balance: number
  available_balance: number
  updated_at: string
}

export interface TokoItem {
  id: number
  name: string
  token: string
  charge: number
  callback_url?: string
}

export interface TokoWorkspaceItem {
  id: number
  name: string
  token: string
  charge: number
  callback_url?: string
  settlement_balance: number
  available_balance: number
  updated_at: string
}

export interface TokoWorkspaceSummary {
  total_tokos: number
  total_settlement_balance: number
  total_available_balance: number
}

export interface TokoWorkspacePage {
  items: TokoWorkspaceItem[]
  summary: TokoWorkspaceSummary
  total: number
  limit: number
  offset: number
  has_more: boolean
}

export interface TokoWorkspaceQuery {
  limit?: number
  offset?: number
  q?: string
}
