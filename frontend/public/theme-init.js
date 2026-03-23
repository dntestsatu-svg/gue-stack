/* global document, localStorage, window */
;(() => {
  const storageKey = 'gue_theme'
  const saved = localStorage.getItem(storageKey)
  const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
  const resolved = saved === 'dark' || saved === 'light' ? saved : prefersDark ? 'dark' : 'light'

  document.documentElement.classList.toggle('dark', resolved === 'dark')
  document.documentElement.style.colorScheme = resolved
})()
