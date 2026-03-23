const gtmId = (import.meta.env.VITE_GTM_ID ?? '').trim()
const GTM_SCRIPT_ID = 'public-gtm-script'
const GTM_LOAD_TIMEOUT_MS = 1500
const GTM_EVENT_TIMEOUT_MS = 1800

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

let gtmLoadPromise: Promise<boolean> | null = null

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

function withTimeout<T>(promise: Promise<T>, timeoutMs: number, fallback: T) {
  return Promise.race([
    promise,
    new Promise<T>((resolve) => {
      window.setTimeout(() => resolve(fallback), timeoutMs)
    }),
  ])
}

export function ensureGtmLoaded() {
  if (!isGtmEnabled() || typeof document === 'undefined') {
    return Promise.resolve(false)
  }

  if (gtmLoadPromise) {
    return gtmLoadPromise
  }

  const dataLayer = getDataLayer()
  if (!dataLayer) {
    return Promise.resolve(false)
  }

  const existingScript = document.getElementById(GTM_SCRIPT_ID)
  if (existingScript) {
    if (existingScript.dataset.loaded === 'true') {
      gtmLoadPromise = Promise.resolve(true)
      return gtmLoadPromise
    }

    gtmLoadPromise = withTimeout<boolean>(
      new Promise((resolve) => {
        existingScript.addEventListener('load', () => resolve(true), { once: true })
        existingScript.addEventListener('error', () => resolve(false), { once: true })
      }),
      GTM_LOAD_TIMEOUT_MS,
      false,
    )
    return gtmLoadPromise
  }

  dataLayer.push({
    'gtm.start': Date.now(),
    event: 'gtm.js',
  })

  const script = document.createElement('script')
  script.id = GTM_SCRIPT_ID
  script.async = true
  script.src = `https://www.googletagmanager.com/gtm.js?id=${encodeURIComponent(gtmId)}`

  gtmLoadPromise = withTimeout<boolean>(
    new Promise((resolve) => {
      script.addEventListener('load', () => {
        script.dataset.loaded = 'true'
        resolve(true)
      }, { once: true })
      script.addEventListener('error', () => resolve(false), { once: true })
    }),
    GTM_LOAD_TIMEOUT_MS,
    false,
  )

  document.head.appendChild(script)
  return gtmLoadPromise
}

export function pushGtmEvent(payload: GtmEventPayload) {
  if (!isGtmEnabled()) {
    return false
  }

  const dataLayer = getDataLayer()
  if (!dataLayer) {
    return false
  }

  dataLayer.push(payload)
  return true
}

export async function trackGtmEventBeforeAction(
  payload: GtmEventPayload,
  action: () => void,
  options?: {
    loadTimeoutMs?: number
    eventTimeoutMs?: number
  },
) {
  if (typeof window === 'undefined') {
    action()
    return
  }

  const loadTimeoutMs = options?.loadTimeoutMs ?? GTM_LOAD_TIMEOUT_MS
  const eventTimeoutMs = options?.eventTimeoutMs ?? GTM_EVENT_TIMEOUT_MS

  await withTimeout(ensureGtmLoaded(), loadTimeoutMs, false)

  let actionTriggered = false
  const triggerAction = () => {
    if (actionTriggered) {
      return
    }

    actionTriggered = true
    action()
  }

  const gtmTracked = pushGtmEvent({
    ...payload,
    eventCallback: triggerAction,
    eventTimeout: eventTimeoutMs,
  })

  if (!gtmTracked) {
    triggerAction()
    return
  }

  window.setTimeout(triggerAction, eventTimeoutMs + 100)
}

export function trackNavigationWithGtm(payload: GtmEventPayload, destination: string) {
  return trackGtmEventBeforeAction(payload, () => {
    window.location.assign(destination)
  })
}
