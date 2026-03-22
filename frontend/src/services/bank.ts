import api from './http'
import type { ApiResponse, BankItem, BankListPage, BankListQuery, BankPaymentOption } from './types'

export interface CreateBankPayload {
  payment_id: number
  account_name: string
  account_number: string
}

export async function list(query: BankListQuery = {}) {
  const { data } = await api.get<ApiResponse<BankListPage>>('/api/v1/banks', {
    params: query,
  })
  return data.data
}

export async function paymentOptions(query: { q?: string; limit?: number } = {}) {
  const { data } = await api.get<ApiResponse<BankPaymentOption[]>>('/api/v1/banks/payment-options', {
    params: query,
  })
  return data.data
}

export async function create(payload: CreateBankPayload) {
  const { data } = await api.post<ApiResponse<BankItem>>('/api/v1/banks', payload)
  return data.data
}

export async function remove(bankID: number) {
  const { data } = await api.delete<ApiResponse<null>>(`/api/v1/banks/${bankID}`)
  return data.message ?? 'Bank deleted successfully'
}
