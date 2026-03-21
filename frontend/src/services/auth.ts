import api from './http'
import type { ApiResponse, AuthResponseData } from './types'

export interface RegisterPayload {
  name: string
  email: string
  password: string
}

export interface LoginPayload {
  email: string
  password: string
}

export async function register(payload: RegisterPayload) {
  const { data } = await api.post<ApiResponse<AuthResponseData>>('/api/v1/auth/register', payload)
  return data.data
}

export async function login(payload: LoginPayload) {
  const { data } = await api.post<ApiResponse<AuthResponseData>>('/api/v1/auth/login', payload)
  return data.data
}

export async function refresh(refreshToken: string) {
  const { data } = await api.post<ApiResponse<AuthResponseData>>('/api/v1/auth/refresh', {
    refresh_token: refreshToken,
  })
  return data.data
}

export async function logout(refreshToken: string) {
  await api.post('/api/v1/auth/logout', {
    refresh_token: refreshToken,
  })
}
