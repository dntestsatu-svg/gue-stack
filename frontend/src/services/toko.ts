import api from './http'
import type { ApiResponse, TokoBalanceItem, TokoItem, TokoWorkspacePage, TokoWorkspaceQuery } from './types'

export interface CreateTokoPayload {
  name: string
  callback_url?: string
}

export interface ManualSettlementPayload {
  settlement_balance: number
}

export async function fetchWorkspace(query: TokoWorkspaceQuery = {}) {
  const { data } = await api.get<ApiResponse<TokoWorkspacePage>>('/api/v1/tokos/workspace', {
    params: query,
  })
  return data.data
}

export async function fetchBalances() {
  const { data } = await api.get<ApiResponse<TokoBalanceItem[]>>('/api/v1/tokos/balances')
  return data.data
}

export async function fetchTokos() {
  const { data } = await api.get<ApiResponse<TokoItem[]>>('/api/v1/tokos')
  return data.data
}

export async function createToko(payload: CreateTokoPayload) {
  const { data } = await api.post<ApiResponse<TokoItem>>('/api/v1/tokos', payload)
  return data.data
}

export async function applySettlement(tokoID: number, payload: ManualSettlementPayload) {
  const { data } = await api.patch<ApiResponse<TokoBalanceItem>>(`/api/v1/tokos/${tokoID}/settlement`, payload)
  return data.data
}
