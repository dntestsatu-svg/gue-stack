<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import ChangePasswordDialog from '@/components/ChangePasswordDialog.vue'
import { useUserStore } from '@/stores/user'
import ThemeToggle from '@/components/ThemeToggle.vue'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { SidebarTrigger } from '@/components/ui/sidebar'

const route = useRoute()
const userStore = useUserStore()

const pageTitle = computed(() => {
  const title = route.meta.title
  return typeof title === 'string' ? title : 'Workspace'
})

const pageSubtitle = computed(() => {
  switch (route.name) {
    case 'dashboard':
      return 'Operational summary, performance chart, and latest successful transactions.'
    case 'histori-transaksi':
      return 'Server-side filtering and export-ready history for large datasets.'
    case 'toko':
      return 'Toko provisioning and settlement balance management.'
    case 'users':
      return 'Role-based user management with hierarchical access.'
    default:
      return 'Enterprise gateway administration.'
  }
})

const roleLabel = computed(() => userStore.profile?.role ?? 'guest')
</script>

<template>
  <header class="app-topbar sticky top-0 z-20">
    <div class="app-topbar-inner">
      <SidebarTrigger class="-ml-1" />
      <Separator orientation="vertical" class="h-5" />

      <div class="app-topbar-copy">
        <h1>{{ pageTitle }}</h1>
        <p>{{ pageSubtitle }}</p>
      </div>

      <div class="app-topbar-actions">
        <Badge variant="outline" class="hidden capitalize md:inline-flex">{{ roleLabel }}</Badge>
        <ThemeToggle />
        <ChangePasswordDialog />
      </div>
    </div>
  </header>
</template>
