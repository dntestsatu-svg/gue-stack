<script setup lang="ts">
import { Loader2 } from 'lucide-vue-next'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{
  class?: string
}>(), {
  class: '',
})
</script>

<template>
  <Loader2 :class="cn('h-4 w-4 animate-spin', props.class)" />
</template>

