import axios from 'axios'

type BrowserContext = {
  hostname: string
  protocol: string
}

const loopbackHosts = new Set(['localhost', '127.0.0.1'])

export function resolveApiBaseURL(rawBaseURL: string | undefined, ctx?: BrowserContext): string {
  const trimmed = rawBaseURL?.trim()
  if (!trimmed) {
    return inferDefaultBaseURL(ctx)
  }
  return normalizeLoopbackHost(trimmed, ctx)
}

function inferDefaultBaseURL(ctx?: BrowserContext): string {
  const browser = ctx ?? getBrowserContext()
  if (!browser) {
    return 'http://localhost:8080'
  }
  const scheme = browser.protocol === 'https:' ? 'https:' : 'http:'
  return `${scheme}//${browser.hostname}:8080`
}

function normalizeLoopbackHost(baseURL: string, ctx?: BrowserContext): string {
  const browser = ctx ?? getBrowserContext()
  if (!browser) {
    return baseURL
  }

  let parsed: globalThis.URL
  try {
    parsed = new globalThis.URL(baseURL)
  } catch {
    return baseURL
  }

  // Avoid localhost vs 127.0.0.1 cross-site cookie issues during local development.
  if (loopbackHosts.has(browser.hostname) && loopbackHosts.has(parsed.hostname) && parsed.hostname !== browser.hostname) {
    parsed.hostname = browser.hostname
    return parsed.toString().replace(/\/$/, '')
  }
  return baseURL
}

function getBrowserContext(): BrowserContext | null {
  if (typeof window === 'undefined') {
    return null
  }
  return {
    hostname: window.location.hostname,
    protocol: window.location.protocol,
  }
}

const api = axios.create({
  baseURL: resolveApiBaseURL(import.meta.env.VITE_API_BASE_URL),
  timeout: 10000,
  withCredentials: true,
})

const csrfHeaderName = import.meta.env.VITE_CSRF_HEADER_NAME ?? 'X-CSRF-Token'
const csrfCookieName = import.meta.env.VITE_CSRF_COOKIE_NAME ?? 'csrf_token'
let csrfTokenCache = ''

export function setCSRFToken(token: string) {
  csrfTokenCache = token.trim()
}

export function clearCSRFToken() {
  csrfTokenCache = ''
}

api.interceptors.request.use((config) => {
  const method = (config.method ?? 'get').toUpperCase()
  if (method === 'GET' || method === 'HEAD' || method === 'OPTIONS') {
    return config
  }

  const csrfToken = csrfTokenCache || readCookie(csrfCookieName)
  if (csrfToken) {
    config.headers[csrfHeaderName] = csrfToken
  }

  return config
})

export function getApiErrorMessage(error: unknown): string {
  if (axios.isAxiosError(error)) {
    const serverMessage = (error.response?.data as { message?: string } | undefined)?.message
    return serverMessage ?? error.message
  }
  if (error instanceof Error) {
    return error.message
  }
  return 'Unexpected error'
}

function readCookie(name: string): string {
  if (typeof document === 'undefined' || !document.cookie) {
    return ''
  }
  const target = `${name}=`
  const cookies = document.cookie.split(';')
  for (const raw of cookies) {
    const entry = raw.trim()
    if (entry.startsWith(target)) {
      return decodeURIComponent(entry.slice(target.length))
    }
  }
  return ''
}

export default api
