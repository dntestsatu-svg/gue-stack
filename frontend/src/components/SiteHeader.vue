<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { Bell } from 'lucide-vue-next'
import { useUserStore } from '@/stores/user'
import ThemeToggle from '@/components/ThemeToggle.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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
  <header class="sticky top-0 z-20 border-b border-[var(--border)] bg-[var(--background)]/95 backdrop-blur supports-[backdrop-filter]:bg-[var(--background)]/70">
    <div class="flex min-h-16 min-w-0 flex-wrap items-center gap-3 px-4 py-3 lg:px-6">
      <SidebarTrigger class="-ml-1" />
      <Separator orientation="vertical" class="h-5" />

      <div class="min-w-0 flex-1">
        <h1 class="truncate text-base font-semibold md:text-lg">{{ pageTitle }}</h1>
        <p class="text-muted-foreground hidden truncate text-xs md:block">{{ pageSubtitle }}</p>
      </div>

      <div class="ml-auto flex shrink-0 items-center gap-2 self-start sm:self-center">
        <Badge variant="outline" class="hidden capitalize md:inline-flex">{{ roleLabel }}</Badge>
        <ThemeToggle />
        <Button variant="outline" size="icon" class="hidden md:inline-flex">
          <Bell class="h-4 w-4" />
          <span class="sr-only">Notifications</span>
        </Button>
      </div>
    </div>
  </header>
</template>
