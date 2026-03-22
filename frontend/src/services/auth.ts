import api, { hasCookie, setCSRFToken } from './http'
import type { ApiResponse, AuthResponseData } from './types'

export interface LoginPayload {
  email: string
  password: string
}

export interface RegisterPayload {
  name: string
  email: string
  password: string
}

interface CsrfData {
  csrf_token: string
}

const sessionHintCookieName = import.meta.env.VITE_SESSION_HINT_COOKIE_NAME ?? 'session_hint'

export async function initCsrf() {
  const { data } = await api.get<ApiResponse<CsrfData>>('/api/v1/auth/csrf')
  setCSRFToken(data.data.csrf_token)
}

export async function login(payload: LoginPayload) {
  const { data } = await api.post<ApiResponse<AuthResponseData>>('/api/v1/auth/login', payload)
  setCSRFToken(data.data.csrf_token)
  return data.data
}

export async function register(payload: RegisterPayload) {
  const { data } = await api.post<ApiResponse<AuthResponseData>>('/api/v1/auth/register', payload)
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

export function hasSessionHint() {
  return hasCookie(sessionHintCookieName)
}
