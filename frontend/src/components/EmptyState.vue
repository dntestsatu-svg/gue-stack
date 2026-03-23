<script setup lang="ts">
import type { Component } from 'vue'
import { computed } from 'vue'
import { Database } from 'lucide-vue-next'
import AppIcon from '@/components/AppIcon.vue'
import { Button } from '@/components/ui/button'

const props = withDefaults(defineProps<{
  title?: string
  description?: string
  actionLabel?: string
  icon?: Component
}>(), {
  title: 'Belum Ada Data',
  description: 'Data akan muncul di sini setelah aktivitas pertama dilakukan.',
  actionLabel: '',
})

const emit = defineEmits<{
  action: []
}>()

const resolvedIcon = computed<Component>(() => props.icon ?? Database)
</script>

<template>
  <div
    data-empty-state
    class="app-empty-state mx-auto flex w-full max-w-3xl flex-col items-center justify-center text-center"
  >
    <AppIcon :icon="resolvedIcon" class="mb-4 h-10 w-10 text-muted-foreground" />
    <h3 class="text-lg font-semibold text-foreground">{{ props.title }}</h3>
    <p class="mt-2 max-w-md text-sm text-muted-foreground">{{ props.description }}</p>
    <Button
      v-if="props.actionLabel"
      class="mt-5"
      variant="outline"
      @click="emit('action')"
    >
      {{ props.actionLabel }}
    </Button>
  </div>
</template>
