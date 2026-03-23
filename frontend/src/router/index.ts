import {
  type RouteRecordRaw,
  type Router,
} from 'vue-router'
import { publicPagesCatalog } from '@/lib/public-pages'
import { useAuthStore } from '@/stores/auth'
import { useUserStore } from '@/stores/user'
import { resolveRedirect, resolveRoleRedirect } from './guards'

const guestOnlyPaths = new Set(['/login', '/register'])

export const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'landing',
    component: () => import('@/views/LandingPageView.vue'),
    meta: {
      publicLayout: true,
      title: 'Gateway QRIS untuk Merchant',
    },
  },
  ...publicPagesCatalog.map((page) => ({
    path: page.path,
    name: page.key,
    component: () => import('@/views/PublicContentPageView.vue'),
    meta: {
      publicLayout: true,
      title: page.title,
    },
  } satisfies RouteRecordRaw)),
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/LoginView.vue'),
    meta: { authLayout: true, title: 'Masuk', noindex: true },
  },
  {
    path: '/register',
    name: 'register',
    component: () => import('@/views/RegisterView.vue'),
    meta: { authLayout: true, title: 'Daftar', noindex: true },
  },
  {
    path: '/dashboard',
    name: 'dashboard',
    component: () => import('@/views/DashboardView.vue'),
    meta: { requiresAuth: true, requiresActive: true, title: 'Dashboard', noindex: true },
  },
  {
    path: '/histori-transaksi',
    name: 'histori-transaksi',
    component: () => import('@/views/TransactionHistoryView.vue'),
    meta: { requiresAuth: true, requiresActive: true, title: 'Histori Transaksi', noindex: true },
  },
  {
    path: '/toko',
    name: 'toko',
    component: () => import('@/views/TokoView.vue'),
    meta: { requiresAuth: true, requiresActive: true, title: 'Manajemen Toko', noindex: true },
  },
  {
    path: '/testing',
    name: 'testing',
    component: () => import('@/views/TestingView.vue'),
    meta: { requiresAuth: true, requiresActive: true, title: 'Testing Toko', noindex: true },
  },
  {
    path: '/dokumentasi-api',
    name: 'dokumentasi-api',
    component: () => import('@/views/ApiDocumentationView.vue'),
    meta: { requiresAuth: true, requiresActive: true, title: 'Dokumentasi API', noindex: true },
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
      noindex: true,
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
      noindex: true,
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
      noindex: true,
    },
  },
]

export function installRouterGuards(router: Router) {
  router.beforeEach(async (to) => {
    const auth = useAuthStore()
    const userStore = useUserStore()
    const isGuestOnlyPath = guestOnlyPaths.has(to.path)

    const needsAuthResolution = Boolean(
      to.meta.requiresAuth
      || to.meta.requiresActive
      || Array.isArray(to.meta.allowedRoles)
    )

    if (isGuestOnlyPath) {
      await auth.restoreGuestSession()
    } else if (needsAuthResolution) {
      await auth.restoreSession()
    }

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
}
