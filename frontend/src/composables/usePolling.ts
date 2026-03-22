import { onBeforeUnmount, onMounted } from 'vue'

type PollingOptions = {
  runWhenHidden?: boolean
}

export function usePolling(
  task: () => Promise<void>,
  intervalMs: number,
  options: PollingOptions = {},
) {
  let timer: number | null = null
  let inFlight = false
  let active = true

  const run = async () => {
    if (!active || inFlight) {
      return
    }
    if (!options.runWhenHidden && typeof document !== 'undefined' && document.hidden) {
      return
    }

    inFlight = true
    try {
      await task()
    } finally {
      inFlight = false
    }
  }

  onMounted(() => {
    active = true
    void run()
    timer = window.setInterval(() => {
      void run()
    }, intervalMs)
  })

  onBeforeUnmount(() => {
    active = false
    if (timer !== null) {
      window.clearInterval(timer)
      timer = null
    }
  })

  return { runNow: run }
}
