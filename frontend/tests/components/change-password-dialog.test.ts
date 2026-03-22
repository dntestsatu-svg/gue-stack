import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ChangePasswordDialog from '@/components/ChangePasswordDialog.vue'

const {
  initCsrfMock,
  changePasswordMock,
  toastSuccessMock,
} = vi.hoisted(() => ({
  initCsrfMock: vi.fn(),
  changePasswordMock: vi.fn(),
  toastSuccessMock: vi.fn(),
}))

vi.mock('@/services/auth', () => ({
  initCsrf: initCsrfMock,
}))

vi.mock('@/services/user', () => ({
  changePassword: changePasswordMock,
}))

vi.mock('vue-sonner', () => ({
  toast: {
    success: toastSuccessMock,
  },
}))

describe('ChangePasswordDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    initCsrfMock.mockResolvedValue(undefined)
    changePasswordMock.mockResolvedValue('Password updated successfully')
  })

  it('opens dialog and submits change password request', async () => {
    const wrapper = mount(ChangePasswordDialog, {
      attachTo: document.body,
      global: {
        stubs: {
          Teleport: false,
        },
      },
    })

    const trigger = wrapper.get('button')
    await trigger.trigger('click')
    await flushPromises()

    const currentPassword = document.body.querySelector('#current-password') as any
    const newPassword = document.body.querySelector('#new-password') as any
    const confirmPassword = document.body.querySelector('#confirm-password') as any

    expect(currentPassword).not.toBeNull()
    expect(newPassword).not.toBeNull()
    expect(confirmPassword).not.toBeNull()

    currentPassword!.value = 'oldsecret123'
    currentPassword!.dispatchEvent(new globalThis.Event('input'))
    newPassword!.value = 'newsecret123'
    newPassword!.dispatchEvent(new globalThis.Event('input'))
    confirmPassword!.value = 'newsecret123'
    confirmPassword!.dispatchEvent(new globalThis.Event('input'))

    const form = document.body.querySelector('form')
    form?.dispatchEvent(new globalThis.Event('submit', { bubbles: true, cancelable: true }))
    await flushPromises()

    expect(initCsrfMock).toHaveBeenCalled()
    expect(changePasswordMock).toHaveBeenCalledWith({
      current_password: 'oldsecret123',
      new_password: 'newsecret123',
      confirm_password: 'newsecret123',
    })
    expect(toastSuccessMock).toHaveBeenCalledWith('Password updated successfully')
  })

  it('shows validation message when confirmation does not match', async () => {
    const wrapper = mount(ChangePasswordDialog, {
      attachTo: document.body,
      global: {
        stubs: {
          Teleport: false,
        },
      },
    })

    await wrapper.get('button').trigger('click')
    await flushPromises()

    const currentPassword = document.body.querySelector('#current-password') as any
    const newPassword = document.body.querySelector('#new-password') as any
    const confirmPassword = document.body.querySelector('#confirm-password') as any

    currentPassword.value = 'oldsecret123'
    currentPassword.dispatchEvent(new globalThis.Event('input'))
    newPassword.value = 'newsecret123'
    newPassword.dispatchEvent(new globalThis.Event('input'))
    confirmPassword.value = 'different123'
    confirmPassword.dispatchEvent(new globalThis.Event('input'))

    const form = document.body.querySelector('form')
    form?.dispatchEvent(new globalThis.Event('submit', { bubbles: true, cancelable: true }))
    await flushPromises()

    expect(document.body.textContent).toContain('Konfirmasi password baru belum sama.')
    expect(changePasswordMock).not.toHaveBeenCalled()
  })
})
