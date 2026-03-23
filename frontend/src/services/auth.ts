import api, { hasCookie, setCSRFToken } from './http'
import type { ApiResponse, AuthResponseData, SessionStatusData } from './types'

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
const authConfigErrorMessage = 'Unexpected auth API response. Check /api proxy or VITE_API_BASE_URL configuration.'

function requireEnvelopeData<T>(payload: ApiResponse<T>, context: string): T {
  if (!payload || typeof payload !== 'object' || payload.data == null) {
    throw new Error(`${context}: ${authConfigErrorMessage}`)
  }
  return payload.data
}

function requireCsrfToken(payload: { csrf_token?: string } | undefined, context: string): string {
  const token = payload?.csrf_token?.trim()
  if (!token) {
    throw new Error(`${context}: ${authConfigErrorMessage}`)
  }
  return token
}

export async function initCsrf() {
  const { data } = await api.get<ApiResponse<CsrfData>>('/api/v1/auth/csrf')
  const payload = requireEnvelopeData(data, 'CSRF bootstrap failed')
  setCSRFToken(requireCsrfToken(payload, 'CSRF bootstrap failed'))
}

export async function login(payload: LoginPayload) {
  const { data } = await api.post<ApiResponse<AuthResponseData>>('/api/v1/auth/login', payload)
  const session = requireEnvelopeData(data, 'Login failed')
  setCSRFToken(requireCsrfToken(session, 'Login failed'))
  return session
}

export async function register(payload: RegisterPayload) {
  const { data } = await api.post<ApiResponse<AuthResponseData>>('/api/v1/auth/register', payload)
  const session = requireEnvelopeData(data, 'Register failed')
  setCSRFToken(requireCsrfToken(session, 'Register failed'))
  return session
}

export async function refresh() {
  const { data } = await api.post<ApiResponse<AuthResponseData>>('/api/v1/auth/refresh')
  const session = requireEnvelopeData(data, 'Refresh failed')
  setCSRFToken(requireCsrfToken(session, 'Refresh failed'))
  return session
}

export async function sessionStatus() {
  const { data } = await api.get<ApiResponse<SessionStatusData>>('/api/v1/auth/session')
  return requireEnvelopeData(data, 'Session probe failed')
}

export async function logout() {
  await api.post('/api/v1/auth/logout')
}

export function hasSessionHint() {
  return hasCookie(sessionHintCookieName)
}
