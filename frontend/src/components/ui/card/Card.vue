<template>
  <div :class="cn('rounded-xl border border-slate-200 bg-white shadow-sm', className)">
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
