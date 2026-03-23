const defaultSiteURL = 'https://apigoqr.com'

export const siteConfig = {
  name: 'APIGOQR',
  legalName: 'APIGOQR Gateway Orchestration',
  description:
    'Gateway QRIS modern untuk merchant dengan webhook idempotent, callback final yang stabil, settlement tracking, dan dokumentasi API yang jelas.',
  siteURL: normalizeSiteURL(import.meta.env.VITE_SITE_URL ?? defaultSiteURL),
  ogImage: '/images/landing-og.svg',
  heroImage: '/images/landing-hero.svg',
}

export function withSiteURL(path: string) {
  return new URL(path, siteConfig.siteURL).toString()
}

function normalizeSiteURL(value: string) {
  return value.replace(/\/+$/, '') + '/'
}
