import type { RouteLocationNormalized } from 'vue-router'

const guestOnlyPaths = new Set(['/login', '/register'])

export function resolveRedirect(to: RouteLocationNormalized, isAuthenticated: boolean): string | null {
  if (to.meta.requiresAuth && !isAuthenticated) {
    return '/login'
  }
  if (guestOnlyPaths.has(to.path) && isAuthenticated) {
    return '/dashboard'
  }
  return null
}
