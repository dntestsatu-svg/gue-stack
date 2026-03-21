import { describe, expect, it } from 'vitest'
import { resolveRedirect } from '@/router/guards'

const route = (path: string, requiresAuth = false) => ({
  path,
  meta: { requiresAuth },
}) as any

describe('router guards', () => {
  it('redirects guests away from protected route', () => {
    const redirect = resolveRedirect(route('/dashboard', true), false)
    expect(redirect).toBe('/login')
  })

  it('redirects authenticated users away from guest routes', () => {
    const redirect = resolveRedirect(route('/login'), true)
    expect(redirect).toBe('/dashboard')
  })

  it('allows valid route access', () => {
    const redirect = resolveRedirect(route('/dashboard', true), true)
    expect(redirect).toBeNull()
  })
})
