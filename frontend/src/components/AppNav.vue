<template>
  <header class="app-nav sticky top-0 z-40 border-b border-[var(--border)] bg-[var(--panel-overlay)] backdrop-blur-xl">
    <nav class="mx-auto flex w-full max-w-7xl items-center justify-between px-4 py-3 sm:px-6 lg:px-8">
      <RouterLink to="/dashboard" class="font-mono text-xl font-semibold tracking-tight text-[var(--foreground)]">GUE Control</RouterLink>
      <div v-if="auth.isAuthenticated" class="flex items-center gap-2">
        <RouterLink
          to="/dashboard"
          class="app-nav-link"
          :class="{ 'app-nav-link-active': route.path === '/dashboard' }"
        >
          Dashboard
        </RouterLink>
        <RouterLink
          to="/histori-transaksi"
          class="app-nav-link"
          :class="{ 'app-nav-link-active': route.path === '/histori-transaksi' }"
        >
          Histori Transaksi
        </RouterLink>
        <RouterLink
          to="/toko"
          class="app-nav-link"
          :class="{ 'app-nav-link-active': route.path === '/toko' }"
        >
          Toko
        </RouterLink>
      </div>
      <div class="flex items-center gap-2">
        <ThemeToggle />
        <Button
          v-if="auth.isAuthenticated"
          variant="ghost"
          size="sm"
          class="min-w-20"
          @click="handleLogout"
        >
          Logout
        </Button>
        <RouterLink v-else to="/login" class="app-nav-link">Login</RouterLink>
      </div>
    </nav>
  </header>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { Button } from '@/components/ui/button'
import ThemeToggle from '@/components/ThemeToggle.vue'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const handleLogout = async () => {
  await auth.logout()
  await router.push('/login')
}
</script>
