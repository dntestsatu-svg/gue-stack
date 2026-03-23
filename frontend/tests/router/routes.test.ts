import { describe, expect, it } from 'vitest'
import { routes } from '@/router'

describe('router public routes', () => {
  it('exposes root path as a public landing page', () => {
    const rootRoute = routes.find((route) => route.path === '/')

    expect(rootRoute).toBeTruthy()
    expect(rootRoute?.redirect).toBeUndefined()
    expect(rootRoute?.meta?.publicLayout).toBe(true)
    expect(rootRoute?.meta?.requiresAuth).toBeUndefined()
  })
})
