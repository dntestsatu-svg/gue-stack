<template>
  <div class="app-shell text-foreground">
    <div v-if="isPublicLayout" class="landing-shell">
      <RouterView />
    </div>

    <div v-else-if="isAuthLayout" class="app-auth-shell">
      <RouterView />
    </div>

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
import { useHead } from '@unhead/vue'
import AppSidebar from '@/components/AppSidebar.vue'
import SiteHeader from '@/components/SiteHeader.vue'
import { Toaster } from '@/components/ui/sonner'
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar'
import { withSiteURL } from '@/lib/site'

const route = useRoute()
const isPublicLayout = computed(() => Boolean(route.meta.publicLayout))
const isAuthLayout = computed(() => Boolean(route.meta.authLayout))

useHead(() => {
  const rawTitle = typeof route.meta.title === 'string' ? route.meta.title : 'GUE Control'
  const isNoIndex = route.meta.noindex === true
  const canonicalPath = route.path || '/'

  return {
    htmlAttrs: {
      lang: 'id',
    },
    title: rawTitle === 'GUE Control' ? rawTitle : `${rawTitle} | GUE Control`,
    link: [
      {
        rel: 'canonical',
        href: withSiteURL(canonicalPath),
      },
    ],
    meta: [
      {
        name: 'robots',
        content: isNoIndex ? 'noindex, nofollow' : 'index, follow',
      },
    ],
  }
})
</script>
