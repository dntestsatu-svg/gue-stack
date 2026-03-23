<template>
  <Button
    variant="outline"
    size="sm"
    class="app-header-control theme-toggle min-w-24 justify-center gap-2"
    @click="toggleTheme"
  >
    <span aria-hidden="true">
      <svg
        v-if="theme === 'dark'"
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 24 24"
        class="h-4 w-4"
        fill="none"
        stroke="currentColor"
        stroke-width="1.8"
      >
        <path d="M21 12.79A9 9 0 1 1 11.21 3c0 .59.07 1.17.2 1.73A7 7 0 0 0 19.27 12c.56.13 1.14.2 1.73.2Z" />
      </svg>
      <svg
        v-else
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 24 24"
        class="h-4 w-4"
        fill="none"
        stroke="currentColor"
        stroke-width="1.8"
      >
        <circle cx="12" cy="12" r="4" />
        <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41" />
      </svg>
    </span>
    <span>{{ theme === 'dark' ? 'Dark' : 'Light' }}</span>
  </Button>
</template>

<script setup lang="ts">
import { Button } from '@/components/ui/button'
import { useTheme } from '@/composables/useTheme'

const { theme, toggleTheme } = useTheme()
</script>
