<template>
  <div class="app-shell text-foreground">
    <main v-if="isAuthLayout" class="app-auth-shell">
      <RouterView />
    </main>

    <SidebarProvider v-else>
      <AppSidebar />
      <SidebarInset class="min-w-0 overflow-x-hidden">
        <SiteHeader />
        <main class="app-content-frame">
          <RouterView />
        </main>
      </SidebarInset>
    </SidebarProvider>

    <Toaster rich-colors position="top-right" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import AppSidebar from '@/components/AppSidebar.vue'
import SiteHeader from '@/components/SiteHeader.vue'
import { Toaster } from '@/components/ui/sonner'
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar'

const route = useRoute()
const isAuthLayout = computed(() => Boolean(route.meta.authLayout))
</script>
