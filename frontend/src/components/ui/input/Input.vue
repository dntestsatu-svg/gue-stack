<template>
  <input
    :value="modelValue"
    :type="type"
    :placeholder="placeholder"
    :class="cn('flex h-10 w-full rounded-md border border-[var(--border)] bg-[var(--background-elevated)] px-3 py-2 text-sm text-[var(--foreground)] outline-none ring-0 placeholder:text-[var(--muted-foreground)]/70 focus:border-[var(--brand)] focus:ring-2 focus:ring-[color-mix(in_oklab,var(--brand)_22%,transparent)]', className)"
    @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
  />
</template>

<script setup lang="ts">
import { cn } from '@/lib/utils'

defineEmits<{
  'update:modelValue': [value: string]
}>()

withDefaults(
  defineProps<{
    modelValue?: string
    type?: string
    placeholder?: string
    className?: string
  }>(),
  {
    modelValue: '',
    type: 'text',
    placeholder: '',
    className: '',
  },
)
</script>
