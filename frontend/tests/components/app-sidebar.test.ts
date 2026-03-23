import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { createPinia, setActivePinia } from 'pinia'
import AppSidebar from '@/components/AppSidebar.vue'
import { SidebarProvider } from '@/components/ui/sidebar'
import { useUserStore } from '@/stores/user'

describe('AppSidebar', () => {
  it('renders menu safely with current user profile', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const userStore = useUserStore()
    userStore.setProfile({
      id: 1,
      name: 'Developer',
      email: 'dev@gue.local',
      role: 'dev',
      is_active: true,
    })

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/dashboard', component: { template: '<div />' } },
        { path: '/histori-transaksi', component: { template: '<div />' } },
        { path: '/toko', component: { template: '<div />' } },
        { path: '/bank-management', component: { template: '<div />' } },
        { path: '/withdraw', component: { template: '<div />' } },
        { path: '/users', component: { template: '<div />' } },
      ],
    })
    await router.push('/dashboard')
    await router.isReady()

    const wrapper = mount(
      {
        components: { SidebarProvider, AppSidebar },
        template: `
          <SidebarProvider>
            <AppSidebar />
          </SidebarProvider>
        `,
      },
      {
        global: {
          plugins: [pinia, router],
        },
      },
    )

    expect(wrapper.text()).toContain('GUE Control')
    expect(wrapper.text()).toContain('Dashboard')
    expect(wrapper.text()).toContain('Bank Management')
    expect(wrapper.text()).toContain('Withdraw')
    expect(wrapper.text()).toContain('Developer')
  })

  it('hides user management menu for role user', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const userStore = useUserStore()
    userStore.setProfile({
      id: 2,
      name: 'Basic User',
      email: 'user@gue.local',
      role: 'user',
      is_active: true,
    })

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/dashboard', component: { template: '<div />' } },
        { path: '/histori-transaksi', component: { template: '<div />' } },
        { path: '/toko', component: { template: '<div />' } },
        { path: '/bank-management', component: { template: '<div />' } },
        { path: '/withdraw', component: { template: '<div />' } },
        { path: '/users', component: { template: '<div />' } },
      ],
    })
    await router.push('/dashboard')
    await router.isReady()

    const wrapper = mount(
      {
        components: { SidebarProvider, AppSidebar },
        template: `
          <SidebarProvider>
            <AppSidebar />
          </SidebarProvider>
        `,
      },
      {
        global: {
          plugins: [pinia, router],
        },
      },
    )

    expect(wrapper.text()).toContain('Dashboard')
    expect(wrapper.text()).not.toContain('Bank Management')
    expect(wrapper.text()).not.toContain('Withdraw')
    expect(wrapper.text()).not.toContain('User Management')
  })
})
