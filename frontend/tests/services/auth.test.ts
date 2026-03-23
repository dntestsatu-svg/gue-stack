import { beforeEach, describe, expect, it, vi } from 'vitest'
import { initCsrf, login } from '@/services/auth'

const { getMock, postMock, setCSRFTokenMock, hasCookieMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
  setCSRFTokenMock: vi.fn(),
  hasCookieMock: vi.fn(() => false),
}))

vi.mock('@/services/http', () => ({
  default: {
    get: getMock,
    post: postMock,
  },
  setCSRFToken: setCSRFTokenMock,
  hasCookie: hasCookieMock,
}))

describe('auth service', () => {
  beforeEach(() => {
    getMock.mockReset()
    postMock.mockReset()
    setCSRFTokenMock.mockReset()
    hasCookieMock.mockReset()
    hasCookieMock.mockReturnValue(false)
  })

  it('throws a clear error when csrf endpoint returns unexpected payload', async () => {
    getMock.mockResolvedValue({ data: '<!doctype html>' })

    await expect(initCsrf()).rejects.toThrow(
      'CSRF bootstrap failed: Unexpected auth API response. Check /api proxy or VITE_API_BASE_URL configuration.',
    )
  })

  it('stores csrf token from login response', async () => {
    postMock.mockResolvedValue({
      data: {
        data: {
          user: { id: 1, name: 'Jane', email: 'jane@example.com', role: 'admin', is_active: true },
          expires_in: 900,
          csrf_token: 'csrf-123',
        },
      },
    })

    const result = await login({ email: 'jane@example.com', password: 'password123' })

    expect(setCSRFTokenMock).toHaveBeenCalledWith('csrf-123')
    expect(result.user.email).toBe('jane@example.com')
  })
})
