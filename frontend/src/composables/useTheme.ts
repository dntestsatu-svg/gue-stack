import { ref } from 'vue'

export type ThemeMode = 'light' | 'dark'

const STORAGE_KEY = 'gue_theme'

const theme = ref<ThemeMode>(resolveInitialTheme())

function resolveInitialTheme(): ThemeMode {
  if (typeof document === 'undefined') {
    return 'light'
  }

  if (document.documentElement.classList.contains('dark')) {
    return 'dark'
  }

  return 'light'
}

function applyTheme(nextTheme: ThemeMode) {
  if (typeof document === 'undefined') {
    return
  }

  document.documentElement.classList.toggle('dark', nextTheme === 'dark')
  document.documentElement.style.colorScheme = nextTheme
  localStorage.setItem(STORAGE_KEY, nextTheme)
  theme.value = nextTheme
}

function toggleTheme() {
  const nextTheme: ThemeMode = theme.value === 'dark' ? 'light' : 'dark'
  applyTheme(nextTheme)
}

export function useTheme() {
  return {
    theme,
    toggleTheme,
  }
}

export { STORAGE_KEY as THEME_STORAGE_KEY }
