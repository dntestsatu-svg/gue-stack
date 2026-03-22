import { describe, expect, it } from 'vitest'
import { resolveRedirect, resolveRoleRedirect } from '@/router/guards'

const route = (path: string, requiresAuth = false) => ({
  path,
  meta: { requiresAuth },
}) as any

describe('router guards', () => {
  it('redirects guests away from protected route', () => {
    const redirect = resolveRedirect(route('/dashboard', true), false, null)
    expect(redirect).toBe('/login')
  })

  it('redirects authenticated users away from guest routes', () => {
    const redirect = resolveRedirect(route('/login'), true, true)
    expect(redirect).toBe('/dashboard')
  })

  it('redirects authenticated users away from register guest route', () => {
    const redirect = resolveRedirect(route('/register'), true, true)
    expect(redirect).toBe('/dashboard')
  })

  it('allows valid route access', () => {
    const redirect = resolveRedirect(route('/dashboard', true), true, true)
    expect(redirect).toBeNull()
  })

  it('redirects inactive user from active-only route', () => {
    const redirect = resolveRedirect(
      { path: '/dashboard', meta: { requiresAuth: true, requiresActive: true } } as any,
      true,
      false,
    )
    expect(redirect).toBe('/login')
  })

  it('redirects role if route has allowed roles', () => {
    const redirect = resolveRoleRedirect(
      { path: '/users', meta: { allowedRoles: ['dev', 'admin'] } } as any,
      'user' as any,
    )
    expect(redirect).toBe('/dashboard')
  })
})
