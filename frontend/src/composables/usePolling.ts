import { onBeforeUnmount, onMounted } from 'vue'

export function usePolling(task: () => Promise<void>, intervalMs: number) {
  let timer: number | null = null
  let inFlight = false

  const run = async () => {
    if (inFlight) {
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
    void run()
    timer = window.setInterval(() => {
      void run()
    }, intervalMs)
  })

  onBeforeUnmount(() => {
    if (timer !== null) {
      window.clearInterval(timer)
    }
  })

  return { runNow: run }
}
