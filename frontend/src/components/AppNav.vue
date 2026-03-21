<template>
  <header class="border-b border-slate-200/80 bg-white/80 backdrop-blur">
    <nav class="mx-auto flex w-full max-w-6xl items-center justify-between px-4 py-4 sm:px-6 lg:px-8">
      <RouterLink to="/" class="font-mono text-xl font-semibold tracking-tight text-slate-800">GUE Starter</RouterLink>
      <div class="flex items-center gap-3">
        <RouterLink v-if="!auth.isAuthenticated" to="/login" class="text-sm font-medium text-slate-700 hover:text-amber-700">Login</RouterLink>
        <RouterLink v-if="!auth.isAuthenticated" to="/register" class="text-sm font-medium text-slate-700 hover:text-amber-700">Register</RouterLink>
        <RouterLink v-if="auth.isAuthenticated" to="/dashboard" class="text-sm font-medium text-slate-700 hover:text-amber-700">Dashboard</RouterLink>
      </div>
    </nav>
  </header>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
</script>
