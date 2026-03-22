<script setup lang="ts">
import type { Component } from "vue"
import type { SidebarMenuButtonProps } from "./SidebarMenuButtonChild.vue"
import { reactiveOmit } from "@vueuse/core"
import SidebarMenuButtonChild from "./SidebarMenuButtonChild.vue"

defineOptions({
  inheritAttrs: false,
})

const props = withDefaults(defineProps<SidebarMenuButtonProps & {
  tooltip?: string | Component
}>(), {
  as: "button",
  variant: "default",
  size: "default",
})

const delegatedProps = reactiveOmit(props, "tooltip")
</script>

<template>
  <SidebarMenuButtonChild
    v-bind="{ ...delegatedProps, ...$attrs }"
    :title="typeof tooltip === 'string' ? tooltip : undefined"
  >
    <slot />
  </SidebarMenuButtonChild>
</template>
