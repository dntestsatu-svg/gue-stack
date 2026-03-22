import { createRouter, createWebHistory } from 'vue-router'
import LoginView from '@/views/LoginView.vue'
import DashboardView from '@/views/DashboardView.vue'
import TransactionHistoryView from '@/views/TransactionHistoryView.vue'
import TokoView from '@/views/TokoView.vue'
import { useAuthStore } from '@/stores/auth'
import { useUserStore } from '@/stores/user'
import { resolveRedirect } from './guards'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/dashboard' },
    { path: '/login', name: 'login', component: LoginView },
    { path: '/dashboard', name: 'dashboard', component: DashboardView, meta: { requiresAuth: true, requiresActive: true } },
    { path: '/histori-transaksi', name: 'histori-transaksi', component: TransactionHistoryView, meta: { requiresAuth: true, requiresActive: true } },
    { path: '/toko', name: 'toko', component: TokoView, meta: { requiresAuth: true, requiresActive: true } },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  const userStore = useUserStore()
  await auth.restoreSession()

  if (auth.isAuthenticated && !userStore.profile) {
    try {
      await userStore.fetchMe()
    } catch {
      await auth.logout()
      return { path: '/login' }
    }
  }

  if (auth.isAuthenticated && userStore.profile && !userStore.profile.is_active) {
    await auth.logout()
    if (to.path !== '/login') {
      return { path: '/login' }
    }
  }

  const redirectPath = resolveRedirect(
    to,
    auth.isAuthenticated,
    userStore.profile?.is_active ?? null,
  )
  if (redirectPath) {
    return { path: redirectPath }
  }
  return true
})

export default router
