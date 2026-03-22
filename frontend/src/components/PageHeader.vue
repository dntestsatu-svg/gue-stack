<script setup lang="ts">
import { computed } from 'vue'
import { useFormatters } from '@/composables/useFormatters'

const props = defineProps<{
  eyebrow: string
  title: string
  description: string
  updatedAt?: string
}>()

const { formatTime } = useFormatters()

const descriptionText = computed(() => {
  if (!props.updatedAt) {
    return props.description
  }

  return `${props.description} Updated ${formatTime(props.updatedAt)}`
})
</script>

<template>
  <header class="dashboard-hero">
    <div class="page-header-copy">
      <p class="dashboard-eyebrow">{{ eyebrow }}</p>
      <h1 class="text-2xl font-semibold tracking-tight md:text-3xl">{{ title }}</h1>
      <p class="text-sm text-muted-foreground">
        {{ descriptionText }}
      </p>
    </div>

    <div v-if="$slots.actions" class="page-header-actions">
      <slot name="actions" />
    </div>
  </header>
</template>
