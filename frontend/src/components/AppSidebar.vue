<script setup lang="ts">
import type { Component } from 'vue'
import { computed, markRaw } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ChartArea,
  History,
  LayoutDashboard,
  LogOut,
  Shield,
  Store,
} from 'lucide-vue-next'
import AppIcon from '@/components/AppIcon.vue'
import { useAuthStore } from '@/stores/auth'
import { useUserStore } from '@/stores/user'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar'

type MenuItem = {
  title: string
  to: string
  icon: Component
  roles?: Array<'dev' | 'superadmin' | 'admin' | 'user'>
}

const auth = useAuthStore()
const userStore = useUserStore()
const route = useRoute()
const router = useRouter()

const menuItems: MenuItem[] = [
  { title: 'Dashboard', to: '/dashboard', icon: markRaw(LayoutDashboard) },
  { title: 'Histori Transaksi', to: '/histori-transaksi', icon: markRaw(History) },
  { title: 'Toko', to: '/toko', icon: markRaw(Store) },
  { title: 'User Management', to: '/users', icon: markRaw(Shield), roles: ['dev', 'superadmin', 'admin'] },
]

const role = computed(() => userStore.profile?.role ?? 'user')
const visibleMenuItems = computed(() =>
  menuItems.filter((item) => !item.roles || item.roles.includes(role.value)),
)

const isItemActive = (to: string) => route.path === to || route.path.startsWith(`${to}/`)

const handleLogout = async () => {
  await auth.logout()
  await router.push('/login')
}
</script>

<template>
  <Sidebar variant="inset" collapsible="icon">
    <SidebarHeader>
      <SidebarMenu>
        <SidebarMenuItem>
          <SidebarMenuButton tooltip="GUE Control" class="h-10" @click="router.push('/dashboard')">
            <div class="flex items-center gap-2">
              <div class="flex h-7 w-7 items-center justify-center rounded-md bg-primary text-primary-foreground">
                <ChartArea class="h-4 w-4" />
              </div>
              <div class="flex min-w-0 flex-col leading-tight">
                <span class="truncate text-sm font-semibold">GUE Control</span>
                <span class="text-muted-foreground truncate text-xs">Gateway Orchestration</span>
              </div>
            </div>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarHeader>

    <SidebarContent>
      <SidebarGroup>
        <SidebarGroupLabel>Workspace</SidebarGroupLabel>
        <SidebarGroupContent>
          <SidebarMenu>
            <SidebarMenuItem v-for="item in visibleMenuItems" :key="item.to">
              <SidebarMenuButton
                :is-active="isItemActive(item.to)"
                :tooltip="item.title"
                @click="router.push(item.to)"
              >
                <AppIcon :icon="item.icon" class="h-4 w-4" />
                <span>{{ item.title }}</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>
    </SidebarContent>

    <SidebarFooter>
      <div class="space-y-3 p-2">
        <div class="rounded-lg border border-[var(--border)] bg-[var(--background-muted)]/60 p-3">
          <div class="flex items-center justify-between gap-2">
            <p class="truncate text-sm font-semibold">{{ userStore.profile?.name || 'Unknown User' }}</p>
            <Badge variant="secondary" class="capitalize">{{ role }}</Badge>
          </div>
          <p class="text-muted-foreground truncate text-xs">{{ userStore.profile?.email }}</p>
        </div>
        <Separator />
        <Button variant="outline" class="w-full justify-start gap-2" @click="handleLogout">
          <LogOut class="h-4 w-4" />
          <span>Logout</span>
        </Button>
      </div>
    </SidebarFooter>
  </Sidebar>
</template>
