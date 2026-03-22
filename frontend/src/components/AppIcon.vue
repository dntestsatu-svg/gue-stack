<script setup lang="ts">
import type { Component } from 'vue'

defineOptions({
  inheritAttrs: false,
})

defineProps<{
  icon: Component
}>()
</script>

<template>
  <component :is="icon" v-bind="$attrs" />
</template>
