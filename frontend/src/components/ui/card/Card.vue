<template>
  <div :class="cn('app-panel rounded-xl border', className)">
    <slot />
  </div>
</template>

<script setup lang="ts">
import { cn } from '@/lib/utils'

withDefaults(
  defineProps<{
    className?: string
  }>(),
  {
    className: '',
  },
)
</script>
