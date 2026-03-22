import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import TokoView from '@/views/TokoView.vue'
import { useUserStore } from '@/stores/user'

const {
  fetchWorkspaceMock,
  applySettlementMock,
  createTokoMock,
  updateTokoMock,
  regenerateTokoTokenMock,
  toastSuccessMock,
  toastErrorMock,
} = vi.hoisted(() => ({
  fetchWorkspaceMock: vi.fn(),
  applySettlementMock: vi.fn(),
  createTokoMock: vi.fn(),
  updateTokoMock: vi.fn(),
  regenerateTokoTokenMock: vi.fn(),
  toastSuccessMock: vi.fn(),
  toastErrorMock: vi.fn(),
}))

vi.mock('@/services/toko', () => ({
  fetchWorkspace: fetchWorkspaceMock,
  applySettlement: applySettlementMock,
  createToko: createTokoMock,
  updateToko: updateTokoMock,
  regenerateTokoToken: regenerateTokoTokenMock,
}))

vi.mock('vue-sonner', () => ({
  toast: {
    success: toastSuccessMock,
    error: toastErrorMock,
  },
}))

vi.mock('@/composables/usePolling', () => ({
  usePolling: (task: () => Promise<void>) => {
    void task()
    return { runNow: task }
  },
}))

describe('TokoView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    fetchWorkspaceMock.mockReset()
    applySettlementMock.mockReset()
    createTokoMock.mockReset()
    updateTokoMock.mockReset()
    regenerateTokoTokenMock.mockReset()
    toastSuccessMock.mockReset()
    toastErrorMock.mockReset()

    fetchWorkspaceMock.mockResolvedValue({
      items: [
        {
          id: 1,
          name: 'Toko Alpha',
          token: 'tok_alpha',
          charge: 3,
          callback_url: 'https://example.com/callback',
          settlement_balance: 120000,
          available_balance: 450000,
          updated_at: '2026-03-21T10:00:00Z',
        },
      ],
      summary: {
        total_tokos: 1,
        total_settlement_balance: 120000,
        total_available_balance: 450000,
      },
      total: 1,
      limit: 10,
      offset: 0,
      has_more: false,
    })
    updateTokoMock.mockResolvedValue({
      id: 1,
      name: 'Toko Alpha Updated',
      token: 'tok_alpha',
      charge: 3,
      callback_url: 'https://example.com/updated',
    })
    regenerateTokoTokenMock.mockResolvedValue({
      id: 1,
      name: 'Toko Alpha',
      token: 'tok_rotated',
      charge: 3,
      callback_url: 'https://example.com/callback',
    })

    Object.assign(globalThis.navigator, {
      clipboard: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    })
  })

  it('renders paginated toko workspace for authorized role', async () => {
    const userStore = useUserStore()
    userStore.setProfile({
      id: 11,
      name: 'Admin',
      email: 'admin@gue.local',
      role: 'admin',
      is_active: true,
    })

    const wrapper = mount(TokoView)
    await flushPromises()

    expect(wrapper.text()).toContain('Manage Toko & Settlement')
    expect(wrapper.text()).toContain('Toko Management')
    expect(wrapper.text()).toContain('Settlement Balances')
    expect(wrapper.text()).toContain('Toko Alpha')
    expect(wrapper.text()).toContain('Showing 1-1 of 1')
    expect(fetchWorkspaceMock).toHaveBeenCalled()
    expect(fetchWorkspaceMock).toHaveBeenCalledWith({ limit: 10, offset: 0 })
  })

  it('does not render create toko action for user role', async () => {
    const userStore = useUserStore()
    userStore.setProfile({
      id: 99,
      name: 'Viewer User',
      email: 'viewer@gue.local',
      role: 'user',
      is_active: true,
    })

    const wrapper = mount(TokoView)
    await flushPromises()

    expect(wrapper.text()).not.toContain('Create Toko')
  })

  it('shows toast feedback after token copy', async () => {
    const userStore = useUserStore()
    userStore.setProfile({
      id: 11,
      name: 'Admin',
      email: 'admin@gue.local',
      role: 'admin',
      is_active: true,
    })

    const wrapper = mount(TokoView)
    await flushPromises()

    const copyButton = wrapper.findAll('button').find((button) => button.text().includes('Copy Token'))
    expect(copyButton).toBeDefined()

    await copyButton!.trigger('click')
    await flushPromises()

    expect(globalThis.navigator.clipboard.writeText).toHaveBeenCalled()
    expect(toastSuccessMock).toHaveBeenCalled()
  })

  it('opens manage dialog, updates toko, and regenerates token for admin role', async () => {
    const userStore = useUserStore()
    userStore.setProfile({
      id: 11,
      name: 'Admin',
      email: 'admin@gue.local',
      role: 'admin',
      is_active: true,
    })

    const wrapper = mount(TokoView)
    await flushPromises()

    const manageButton = wrapper.findAll('button').find((button) => button.text().includes('Manage'))
    expect(manageButton).toBeDefined()

    await manageButton!.trigger('click')
    await flushPromises()

    const nameInput = document.body.querySelector('#manage-toko-name') as { value: string; dispatchEvent: (event: unknown) => boolean } | null
    expect(nameInput).toBeDefined()
    nameInput!.value = 'Toko Alpha Updated'
    nameInput!.dispatchEvent(new window.Event('input'))

    const callbackInput = document.body.querySelector('#manage-callback-url') as { value: string; dispatchEvent: (event: unknown) => boolean } | null
    expect(callbackInput).toBeDefined()
    callbackInput!.value = 'https://example.com/updated'
    callbackInput!.dispatchEvent(new window.Event('input'))
    await flushPromises()

    const regenerateButton = Array.from(document.body.querySelectorAll('button')).find((button) =>
      button.textContent?.includes('Generate New Token'),
    )
    expect(regenerateButton).toBeDefined()
    regenerateButton!.click()
    await flushPromises()

    expect(regenerateTokoTokenMock).toHaveBeenCalledWith(1)

    const saveButton = Array.from(document.body.querySelectorAll('button')).find((button) =>
      button.textContent?.includes('Save Changes'),
    )
    expect(saveButton).toBeDefined()
    saveButton!.click()
    await flushPromises()

    expect(updateTokoMock).toHaveBeenCalledWith(1, {
      name: 'Toko Alpha Updated',
      callback_url: 'https://example.com/updated',
    })
    expect(toastSuccessMock).toHaveBeenCalled()
  })
})
