<template>
  <div class="min-h-screen bg-[radial-gradient(circle_at_top_right,_#fff7ed,_#f8fafc_45%,_#ecfeff)] text-slate-900">
    <AppNav />
    <main class="mx-auto w-full max-w-6xl px-4 py-8 sm:px-6 lg:px-8">
      <RouterView />
    </main>
  </div>
</template>

<script setup lang="ts">
import { RouterView } from 'vue-router'
import AppNav from '@/components/AppNav.vue'
</script>
