import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import RegisterView from '@/views/RegisterView.vue'

const { registerMock, ensureGtmLoadedMock, waitForGtmEventMock, redirectToMock } = vi.hoisted(() => ({
  registerMock: vi.fn(),
  ensureGtmLoadedMock: vi.fn(),
  waitForGtmEventMock: vi.fn(async () => true),
  redirectToMock: vi.fn(),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    processing: false,
    register: registerMock,
  }),
}))

vi.mock('@/lib/gtm', () => ({
  ensureGtmLoaded: ensureGtmLoadedMock,
  waitForGtmEvent: waitForGtmEventMock,
}))

vi.mock('@/lib/navigation', () => ({
  redirectTo: redirectToMock,
}))

describe('RegisterView', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    registerMock.mockReset()
    registerMock.mockResolvedValue({})
    ensureGtmLoadedMock.mockReset()
    waitForGtmEventMock.mockReset()
    waitForGtmEventMock.mockResolvedValue(true)
    redirectToMock.mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
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
    expect(waitForGtmEventMock).toHaveBeenCalledWith(expect.objectContaining({
      event: 'sign_up',
      method: 'email',
      account_role: 'admin',
      page_type: 'register',
    }))
    expect(wrapper.text()).toContain('Creating your workspace')

    await vi.advanceTimersByTimeAsync(1600)
    await flushPromises()

    expect(redirectToMock).toHaveBeenCalledWith('/dashboard')
  })
})
