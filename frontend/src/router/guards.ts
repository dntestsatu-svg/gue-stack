import type { RouteLocationNormalized } from 'vue-router'
import type { UserRole } from '@/services/types'

const guestOnlyPaths = new Set(['/login', '/register'])

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

export function resolveRoleRedirect(
  to: RouteLocationNormalized,
  role: UserRole | null,
): string | null {
  const allowedRoles = Array.isArray(to.meta.allowedRoles)
    ? (to.meta.allowedRoles as UserRole[])
    : []

  if (allowedRoles.length === 0) {
    return null
  }
  if (!role || !allowedRoles.includes(role)) {
    return '/dashboard'
  }
  return null
}
