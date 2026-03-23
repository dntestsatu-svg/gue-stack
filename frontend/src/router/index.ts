import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useUserStore } from '@/stores/user'
import { resolveRedirect, resolveRoleRedirect } from './guards'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/dashboard' },
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { authLayout: true, title: 'Sign In' },
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('@/views/RegisterView.vue'),
      meta: { authLayout: true, title: 'Create Account' },
    },
    {
      path: '/dashboard',
      name: 'dashboard',
      component: () => import('@/views/DashboardView.vue'),
      meta: { requiresAuth: true, requiresActive: true, title: 'Dashboard' },
    },
    {
      path: '/histori-transaksi',
      name: 'histori-transaksi',
      component: () => import('@/views/TransactionHistoryView.vue'),
      meta: { requiresAuth: true, requiresActive: true, title: 'Histori Transaksi' },
    },
    {
      path: '/toko',
      name: 'toko',
      component: () => import('@/views/TokoView.vue'),
      meta: { requiresAuth: true, requiresActive: true, title: 'Manajemen Toko' },
    },
    {
      path: '/testing',
      name: 'testing',
      component: () => import('@/views/TestingView.vue'),
      meta: { requiresAuth: true, requiresActive: true, title: 'Testing Toko' },
    },
    {
      path: '/dokumentasi-api',
      name: 'dokumentasi-api',
      component: () => import('@/views/ApiDocumentationView.vue'),
      meta: { requiresAuth: true, requiresActive: true, title: 'Dokumentasi API' },
    },
    {
      path: '/bank-management',
      name: 'bank-management',
      component: () => import('@/views/BankManagementView.vue'),
      meta: {
        requiresAuth: true,
        requiresActive: true,
        allowedRoles: ['dev', 'superadmin', 'admin'],
        title: 'Bank Management',
      },
    },
    {
      path: '/withdraw',
      name: 'withdraw',
      component: () => import('@/views/WithdrawView.vue'),
      meta: {
        requiresAuth: true,
        requiresActive: true,
        allowedRoles: ['dev', 'superadmin', 'admin'],
        title: 'Withdraw',
      },
    },
    {
      path: '/users',
      name: 'users',
      component: () => import('@/views/UserManagementView.vue'),
      meta: {
        requiresAuth: true,
        requiresActive: true,
        allowedRoles: ['dev', 'superadmin', 'admin'],
        title: 'User Management',
      },
    },
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

  const roleRedirect = resolveRoleRedirect(to, userStore.profile?.role ?? null)
  if (roleRedirect) {
    return { path: roleRedirect }
  }
  return true
})

export default router
