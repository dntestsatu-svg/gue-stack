import type { RouteLocationNormalized } from 'vue-router'

const guestOnlyPaths = new Set(['/login'])

export function resolveRedirect(
  to: RouteLocationNormalized,
  isAuthenticated: boolean,
  isActive: boolean | null,
): string | null {
  if (to.meta.requiresAuth && !isAuthenticated) {
    return '/login'
  }
  if (to.meta.requiresActive && isActive === false) {
    return '/login'
  }
  if (guestOnlyPaths.has(to.path) && isAuthenticated) {
    return '/dashboard'
  }
  return null
}
