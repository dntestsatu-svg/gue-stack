import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import UserManagementView from '@/views/UserManagementView.vue'
import { useUserStore } from '@/stores/user'

const {
  listUsersMock,
  createUserMock,
  updateRoleMock,
  updateActiveMock,
  removeUserMock,
  toastSuccessMock,
  toastErrorMock,
} = vi.hoisted(() => ({
  listUsersMock: vi.fn(),
  createUserMock: vi.fn(),
  updateRoleMock: vi.fn(),
  updateActiveMock: vi.fn(),
  removeUserMock: vi.fn(),
  toastSuccessMock: vi.fn(),
  toastErrorMock: vi.fn(),
}))

vi.mock('@/services/user', () => ({
  list: listUsersMock,
  create: createUserMock,
  updateRole: updateRoleMock,
  updateActive: updateActiveMock,
  remove: removeUserMock,
}))

vi.mock('vue-sonner', () => ({
  toast: {
    success: toastSuccessMock,
    error: toastErrorMock,
  },
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({
    query: {},
  }),
  useRouter: () => ({
    replace: vi.fn(),
  }),
}))

describe('UserManagementView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    listUsersMock.mockReset()
    createUserMock.mockReset()
    updateRoleMock.mockReset()
    updateActiveMock.mockReset()
    removeUserMock.mockReset()
    toastSuccessMock.mockReset()
    toastErrorMock.mockReset()

    listUsersMock.mockResolvedValue({
      items: [
        {
          id: 1,
          name: 'Admin One',
          email: 'admin.one@gue.local',
          role: 'admin',
          is_active: true,
        },
      ],
      total: 1,
      limit: 10,
      offset: 0,
      has_more: false,
    })
    updateActiveMock.mockResolvedValue({
      id: 1,
      name: 'Admin One',
      email: 'admin.one@gue.local',
      role: 'admin',
      is_active: false,
    })
    removeUserMock.mockResolvedValue('User deleted successfully')

    Object.assign(globalThis.navigator, {
      clipboard: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    })
  })

  it('renders paginated user list with filters', async () => {
    const userStore = useUserStore()
    userStore.setProfile({
      id: 1,
      name: 'Developer',
      email: 'dev@gue.local',
      role: 'dev',
      is_active: true,
    })

    const wrapper = mount(UserManagementView)
    await flushPromises()

    expect(wrapper.text()).toContain('User Management')
    expect(wrapper.text()).toContain('Admin One')
    expect(wrapper.text()).toContain('Showing 1-1 of 1')
    expect(listUsersMock).toHaveBeenCalled()
    expect(listUsersMock).toHaveBeenCalledWith({
      limit: 10,
      offset: 0,
      q: undefined,
      role: undefined,
    })
  })

  it('copies email and shows toast feedback', async () => {
    const userStore = useUserStore()
    userStore.setProfile({
      id: 1,
      name: 'Developer',
      email: 'dev@gue.local',
      role: 'dev',
      is_active: true,
    })

    const wrapper = mount(UserManagementView)
    await flushPromises()

    const copyButton = wrapper.findAll('button').find((button) => button.text().includes('Copy Email'))
    expect(copyButton).toBeDefined()

    await copyButton!.trigger('click')
    await flushPromises()

    expect(globalThis.navigator.clipboard.writeText).toHaveBeenCalledWith('admin.one@gue.local')
    expect(toastSuccessMock).toHaveBeenCalled()
  })

  it('toggles active status with immediate feedback', async () => {
    const userStore = useUserStore()
    userStore.setProfile({
      id: 99,
      name: 'Developer',
      email: 'dev@gue.local',
      role: 'dev',
      is_active: true,
    })

    const wrapper = mount(UserManagementView)
    await flushPromises()

    const switchButton = wrapper.find('[data-slot="switch"]')
    expect(switchButton.exists()).toBe(true)

    await switchButton.trigger('click')
    await flushPromises()

    expect(updateActiveMock).toHaveBeenCalledWith(1, { is_active: false })
    expect(toastSuccessMock).toHaveBeenCalled()
    expect(wrapper.text()).toContain('Inactive')
  })

  it('deletes user after confirmation and reloads the page data', async () => {
    const userStore = useUserStore()
    userStore.setProfile({
      id: 99,
      name: 'Developer',
      email: 'dev@gue.local',
      role: 'dev',
      is_active: true,
    })

    const wrapper = mount(UserManagementView)
    await flushPromises()

    const deleteButton = wrapper.findAll('button').find((button) => button.text().includes('Delete'))
    expect(deleteButton).toBeDefined()

    await deleteButton!.trigger('click')
    await flushPromises()

    const confirmButton = Array.from(document.body.querySelectorAll('button')).find((button) =>
      button.textContent?.includes('Delete User'),
    )
    expect(confirmButton).toBeDefined()

    confirmButton!.click()
    await flushPromises()

    expect(removeUserMock).toHaveBeenCalledWith(1)
    expect(listUsersMock).toHaveBeenCalledTimes(2)
    expect(toastSuccessMock).toHaveBeenCalled()
  })
})
