<template>
  <button v-bind="attrs" :type="type" :class="classes" :disabled="disabled">
    <slot />
  </button>
</template>

<script setup lang="ts">
import { computed, useAttrs } from 'vue'
import { buttonVariants, type ButtonVariants } from './variants'
import { cn } from '@/lib/utils'

defineOptions({ inheritAttrs: false })

const attrs = useAttrs()

const props = withDefaults(
  defineProps<{
    variant?: ButtonVariants['variant']
    size?: ButtonVariants['size']
    type?: 'button' | 'submit' | 'reset'
    disabled?: boolean
  }>(),
  {
    variant: 'default',
    size: 'default',
    type: 'button',
    disabled: false,
  },
)

const classes = computed(() => cn(buttonVariants({ variant: props.variant, size: props.size }), attrs.class as string))
</script>
