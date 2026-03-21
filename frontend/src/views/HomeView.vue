<template>
  <section class="grid gap-8 lg:grid-cols-2 lg:items-center">
    <div>
      <p class="mb-3 inline-flex rounded-full border border-amber-300 bg-amber-100 px-3 py-1 text-xs font-semibold uppercase tracking-wide text-amber-800">
        Fullstack Starter Kit
      </p>
      <h1 class="text-4xl font-bold leading-tight text-slate-900 sm:text-5xl">
        Build fast. Ship clean architecture.
      </h1>
      <p class="mt-4 max-w-xl text-base text-slate-600">
        Vue 3 + Gin starter kit with JWT auth, Redis, queue workers, cache abstraction, and production-friendly defaults.
      </p>
      <div class="mt-6 flex gap-3">
        <RouterLink to="/register"><Button size="lg">Create Account</Button></RouterLink>
        <RouterLink to="/login"><Button variant="outline" size="lg">Sign In</Button></RouterLink>
      </div>
    </div>
    <Card class-name="overflow-hidden">
      <div class="bg-linear-to-r from-cyan-100 via-amber-100 to-rose-100 p-6">
        <p class="text-sm font-medium text-slate-700">Architecture Flow</p>
        <pre class="mt-3 whitespace-pre-wrap rounded-lg bg-slate-900 p-4 font-mono text-xs text-cyan-200">client -> router -> middleware -> handler -> service -> repository -> cache -> database</pre>
      </div>
    </Card>
  </section>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
</script>
