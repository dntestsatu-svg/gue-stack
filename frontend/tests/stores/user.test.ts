import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useUserStore } from '@/stores/user'

vi.mock('@/services/user', () => ({
  me: vi.fn(),
}))

describe('user store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('fetches current user', async () => {
    const userApi = await import('@/services/user')
    vi.mocked(userApi.me).mockResolvedValue({
      id: 10,
      name: 'Alex',
      email: 'alex@example.com',
      role: 'user',
      is_active: true,
    })

    const user = useUserStore()
    await user.fetchMe()

    expect(user.profile?.id).toBe(10)
    expect(user.loading).toBe(false)
  })
})
