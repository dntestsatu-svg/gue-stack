import api from './http'
import type {
  ApiResponse,
  TestingCallbackReadinessResult,
  TestingGenerateQrisResult,
} from './types'

export interface GenerateTestingQrisPayload {
  toko_id: number
  username: string
  amount: number
  expire?: number
  custom_ref?: string
}

export interface CheckTestingCallbackPayload {
  toko_id: number
}

export async function generateQris(payload: GenerateTestingQrisPayload) {
  const { data } = await api.post<ApiResponse<TestingGenerateQrisResult>>('/api/v1/testing/generate-qris', payload)
  return data.data
}

export async function checkCallbackReadiness(payload: CheckTestingCallbackPayload) {
  const { data } = await api.post<ApiResponse<TestingCallbackReadinessResult>>('/api/v1/testing/callback-readiness', payload)
  return data.data
}
