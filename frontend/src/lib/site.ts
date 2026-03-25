const defaultSiteURL = 'https://apigoqr.com'

export const siteConfig = {
  name: 'APIGOQR',
  legalName: 'APIGOQR Gateway Orchestration',
  description:
    'Gateway QRIS modern untuk merchant dengan webhook idempotent, callback final yang stabil, settlement tracking, dan dokumentasi API yang jelas.',
  siteURL: resolveSiteURL(import.meta.env.VITE_SITE_URL ?? import.meta.env.VITE_API_BASE_URL ?? defaultSiteURL),
  ogImage: '/images/landing-og.svg',
  heroImage: '/images/landing-hero.svg',
}

export function withSiteURL(path: string) {
  return new URL(path, siteConfig.siteURL).toString()
}

function resolveSiteURL(value: string) {
  const trimmed = value.trim()
  if (isAbsoluteHTTPURL(trimmed)) {
    return normalizeSiteURL(trimmed)
  }

  const runtimeOrigin = getRuntimeOrigin()
  if (runtimeOrigin) {
    return normalizeSiteURL(new URL(trimmed || '/', runtimeOrigin).toString())
  }

  return normalizeSiteURL(defaultSiteURL)
}

function normalizeSiteURL(value: string) {
  return value.replace(/\/+$/, '') + '/'
}

function getRuntimeOrigin() {
  if (typeof window === 'undefined') {
    return ''
  }

  const origin = window.location?.origin?.trim() ?? ''
  return isAbsoluteHTTPURL(origin) ? origin : ''
}

function isAbsoluteHTTPURL(value: string) {
  if (!value) {
    return false
  }

  try {
    const url = new URL(value)
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
}
