import api from './http'
import type {
  ApiResponse,
  WithdrawInquiryResult,
  WithdrawOptionsResult,
  WithdrawTransferResult,
} from './types'

export interface WithdrawInquiryPayload {
  toko_id: number
  bank_id: number
  amount: number
}

export interface WithdrawTransferPayload {
  toko_id: number
  bank_id: number
  amount: number
  inquiry_id: number
}

export async function fetchOptions() {
  const { data } = await api.get<ApiResponse<WithdrawOptionsResult>>('/api/v1/withdraw/options')
  return data.data
}

export async function inquiry(payload: WithdrawInquiryPayload) {
  const { data } = await api.post<ApiResponse<WithdrawInquiryResult>>('/api/v1/withdraw/inquiry', payload)
  return data.data
}

export async function transfer(payload: WithdrawTransferPayload) {
  const { data } = await api.post<ApiResponse<WithdrawTransferResult>>('/api/v1/withdraw/transfer', payload)
  return data.data
}
