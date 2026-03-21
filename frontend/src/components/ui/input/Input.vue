<template>
  <input
    :value="modelValue"
    :type="type"
    :placeholder="placeholder"
    :class="cn('flex h-10 w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none ring-0 placeholder:text-slate-400 focus:border-amber-500 focus:ring-2 focus:ring-amber-200', className)"
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
