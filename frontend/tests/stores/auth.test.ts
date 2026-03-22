import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAuthStore } from '@/stores/auth'
import { useUserStore } from '@/stores/user'

vi.mock('@/services/auth', () => ({
  initCsrf: vi.fn(),
  login: vi.fn(),
  refresh: vi.fn(),
  logout: vi.fn(),
}))

vi.mock('@/services/user', () => ({
  me: vi.fn(),
}))

describe('auth store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('stores user session on login', async () => {
    const authApi = await import('@/services/auth')
    vi.mocked(authApi.login).mockResolvedValue({
      user: { id: 1, name: 'Jane', email: 'jane@example.com', role: 'user', is_active: true },
      expires_in: 900,
      csrf_token: 'csrf-1',
    })

    const auth = useAuthStore()
    await auth.login({ email: 'jane@example.com', password: 'password123' })

    expect(auth.isAuthenticated).toBe(true)
    expect(useUserStore().profile?.email).toBe('jane@example.com')
  })

  it('restores session from /user/me when cookie session still valid', async () => {
    const userApi = await import('@/services/user')
    vi.mocked(userApi.me).mockResolvedValue({
      id: 2,
      name: 'John',
      email: 'john@example.com',
      role: 'admin',
      is_active: true,
    })

    const auth = useAuthStore()
    const result = await auth.restoreSession()

    expect(result).toBe(true)
    expect(auth.isAuthenticated).toBe(true)
    expect(useUserStore().profile?.name).toBe('John')
  })

  it('refreshes session when /user/me fails', async () => {
    const userApi = await import('@/services/user')
    const authApi = await import('@/services/auth')
    vi.mocked(userApi.me).mockRejectedValueOnce(new Error('unauthorized')).mockResolvedValueOnce({
      id: 2,
      name: 'John',
      email: 'john@example.com',
      role: 'admin',
      is_active: true,
    })
    vi.mocked(authApi.refresh).mockResolvedValue({
      user: { id: 2, name: 'John', email: 'john@example.com', role: 'admin', is_active: true },
      expires_in: 900,
      csrf_token: 'csrf-2',
    })

    const auth = useAuthStore()
    const result = await auth.restoreSession()

    expect(result).toBe(true)
    expect(auth.isAuthenticated).toBe(true)
    expect(useUserStore().profile?.name).toBe('John')
  })
})
