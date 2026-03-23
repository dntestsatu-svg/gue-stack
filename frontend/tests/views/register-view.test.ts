import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import RegisterView from '@/views/RegisterView.vue'

const { registerMock, ensureGtmLoadedMock, pushGtmEventMock } = vi.hoisted(() => ({
  registerMock: vi.fn(),
  ensureGtmLoadedMock: vi.fn(),
  pushGtmEventMock: vi.fn(() => false),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    processing: false,
    register: registerMock,
  }),
}))

vi.mock('@/lib/gtm', () => ({
  ensureGtmLoaded: ensureGtmLoadedMock,
  pushGtmEvent: pushGtmEventMock,
}))

describe('RegisterView', () => {
  beforeEach(() => {
    registerMock.mockReset()
    registerMock.mockResolvedValue({})
    ensureGtmLoadedMock.mockReset()
    pushGtmEventMock.mockReset()
    pushGtmEventMock.mockReturnValue(false)
  })

  it('loads GTM on mount and emits sign_up tracking after successful registration', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/register', component: RegisterView },
        { path: '/dashboard', component: { template: '<div>Dashboard</div>' } },
        { path: '/login', component: { template: '<div>Login</div>' } },
      ],
    })

    await router.push('/register')
    await router.isReady()
    const pushSpy = vi.spyOn(router, 'push').mockResolvedValue(undefined)

    const wrapper = mount(RegisterView, {
      global: {
        plugins: [router],
      },
    })

    expect(ensureGtmLoadedMock).toHaveBeenCalledOnce()

    await wrapper.get('#name').setValue(' John Doe ')
    await wrapper.get('#email').setValue(' john@example.com ')
    await wrapper.get('#password').setValue('secret123')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(registerMock).toHaveBeenCalledWith({
      name: 'John Doe',
      email: 'john@example.com',
      password: 'secret123',
    })
    expect(pushGtmEventMock).toHaveBeenCalledWith(expect.objectContaining({
      event: 'sign_up',
      method: 'email',
      account_role: 'admin',
      page_type: 'register',
    }))
    expect(pushSpy).toHaveBeenCalledWith('/dashboard')
  })
})
