import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAuthStore } from '@/stores/auth'
import { useUserStore } from '@/stores/user'

vi.mock('@/services/auth', () => ({
  login: vi.fn(),
  register: vi.fn(),
  refresh: vi.fn(),
  logout: vi.fn(),
}))

describe('auth store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
  })

  it('stores tokens and user on login', async () => {
    const authApi = await import('@/services/auth')
    vi.mocked(authApi.login).mockResolvedValue({
      user: { id: 1, name: 'Jane', email: 'jane@example.com' },
      access_token: 'access-1',
      refresh_token: 'refresh-1',
      expires_in: 900,
    })

    const auth = useAuthStore()
    await auth.login({ email: 'jane@example.com', password: 'password123' })

    expect(auth.accessToken).toBe('access-1')
    expect(auth.refreshToken).toBe('refresh-1')
    expect(localStorage.getItem('access_token')).toBe('access-1')
    expect(useUserStore().profile?.email).toBe('jane@example.com')
  })

  it('refreshes session when refresh token exists', async () => {
    const authApi = await import('@/services/auth')
    vi.mocked(authApi.refresh).mockResolvedValue({
      user: { id: 2, name: 'John', email: 'john@example.com' },
      access_token: 'new-access',
      refresh_token: 'new-refresh',
      expires_in: 900,
    })

    localStorage.setItem('refresh_token', 'existing-refresh')

    const auth = useAuthStore()
    const result = await auth.restoreSession()

    expect(result).toBe(true)
    expect(auth.accessToken).toBe('new-access')
    expect(useUserStore().profile?.name).toBe('John')
  })
})
