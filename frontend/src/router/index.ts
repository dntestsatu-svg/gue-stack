import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '@/views/HomeView.vue'
import LoginView from '@/views/LoginView.vue'
import RegisterView from '@/views/RegisterView.vue'
import DashboardView from '@/views/DashboardView.vue'
import { useAuthStore } from '@/stores/auth'
import { resolveRedirect } from './guards'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'home', component: HomeView },
    { path: '/login', name: 'login', component: LoginView },
    { path: '/register', name: 'register', component: RegisterView },
    { path: '/dashboard', name: 'dashboard', component: DashboardView, meta: { requiresAuth: true } },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  await auth.restoreSession()

  const redirectPath = resolveRedirect(to, auth.isAuthenticated)
  if (redirectPath) {
    return { path: redirectPath }
  }
  return true
})

export default router
