const gtmId = (import.meta.env.VITE_GTM_ID ?? '').trim()
const GTM_SCRIPT_ID = 'public-gtm-script'

declare global {
  interface Window {
    dataLayer?: Array<Record<string, unknown>>
  }
}

type GtmEventPayload = Record<string, unknown> & {
  event: string
  eventCallback?: () => void
  eventTimeout?: number
}

function getDataLayer() {
  if (typeof window === 'undefined') {
    return null
  }

  window.dataLayer = window.dataLayer ?? []
  return window.dataLayer
}

export function isGtmEnabled() {
  return gtmId.length > 0
}

export function ensureGtmLoaded() {
  if (!isGtmEnabled() || typeof document === 'undefined') {
    return false
  }

  if (document.getElementById(GTM_SCRIPT_ID)) {
    return true
  }

  const dataLayer = getDataLayer()
  if (!dataLayer) {
    return false
  }

  dataLayer.push({
    'gtm.start': Date.now(),
    event: 'gtm.js',
  })

  const script = document.createElement('script')
  script.id = GTM_SCRIPT_ID
  script.async = true
  script.src = `https://www.googletagmanager.com/gtm.js?id=${encodeURIComponent(gtmId)}`
  document.head.appendChild(script)
  return true
}

export function pushGtmEvent(payload: GtmEventPayload) {
  if (!isGtmEnabled()) {
    return false
  }

  ensureGtmLoaded()
  const dataLayer = getDataLayer()
  if (!dataLayer) {
    return false
  }

  dataLayer.push(payload)
  return true
}

export function trackNavigationWithGtm(payload: GtmEventPayload, destination: string) {
  if (typeof window === 'undefined') {
    return
  }

  const navigate = () => {
    window.location.assign(destination)
  }

  if (!pushGtmEvent({
    ...payload,
    eventCallback: navigate,
    eventTimeout: 800,
  })) {
    navigate()
    return
  }

  window.setTimeout(navigate, 900)
}
