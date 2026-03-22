import api, { setCSRFToken } from './http'
import type { ApiResponse, AuthResponseData } from './types'

export interface LoginPayload {
  email: string
  password: string
}

interface CsrfData {
  csrf_token: string
}

export async function initCsrf() {
  const { data } = await api.get<ApiResponse<CsrfData>>('/api/v1/auth/csrf')
  setCSRFToken(data.data.csrf_token)
}

export async function login(payload: LoginPayload) {
  const { data } = await api.post<ApiResponse<AuthResponseData>>('/api/v1/auth/login', payload)
  setCSRFToken(data.data.csrf_token)
  return data.data
}

export async function refresh() {
  const { data } = await api.post<ApiResponse<AuthResponseData>>('/api/v1/auth/refresh')
  setCSRFToken(data.data.csrf_token)
  return data.data
}

export async function logout() {
  await api.post('/api/v1/auth/logout')
}
