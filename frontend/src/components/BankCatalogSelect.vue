<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { Check, ChevronsUpDown, Search, TriangleAlert } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Spinner } from '@/components/ui/spinner'
import { cn } from '@/lib/utils'
import { getApiErrorMessage } from '@/services/http'
import * as bankApi from '@/services/bank'
import type { BankPaymentOption } from '@/services/types'

const props = withDefaults(defineProps<{
  modelValue?: number | null
  selectedLabel?: string
  disabled?: boolean
  placeholder?: string
}>(), {
  modelValue: null,
  selectedLabel: '',
  disabled: false,
  placeholder: 'Pilih bank dari payment catalog',
})

const emit = defineEmits<{
  'update:modelValue': [value: number | null]
  select: [value: BankPaymentOption]
}>()

const open = ref(false)
const searchTerm = ref('')
const loading = ref(false)
const errorMessage = ref('')
const options = ref<BankPaymentOption[]>([])
let searchTimer: ReturnType<typeof globalThis.setTimeout> | undefined

const triggerLabel = computed(() => props.selectedLabel.trim() || props.placeholder)

const loadOptions = async () => {
  loading.value = true
  errorMessage.value = ''
  try {
    options.value = await bankApi.paymentOptions({
      q: searchTerm.value.trim() || undefined,
      limit: 20,
    })
  } catch (error) {
    errorMessage.value = getApiErrorMessage(error)
  } finally {
    loading.value = false
  }
}

const scheduleSearch = () => {
  if (searchTimer) {
    globalThis.clearTimeout(searchTimer)
  }
  searchTimer = globalThis.setTimeout(() => {
    if (open.value) {
      void loadOptions()
    }
  }, 250)
}

const handleSelect = (option: BankPaymentOption) => {
  emit('update:modelValue', option.id)
  emit('select', option)
  open.value = false
}

watch(open, (isOpen) => {
  if (isOpen) {
    void loadOptions()
    return
  }
  searchTerm.value = ''
  errorMessage.value = ''
})

watch(searchTerm, () => {
  scheduleSearch()
})

onBeforeUnmount(() => {
  if (searchTimer) {
    globalThis.clearTimeout(searchTimer)
  }
})
</script>

<template>
  <Popover v-model:open="open">
    <PopoverTrigger as-child>
      <Button
        type="button"
        variant="outline"
        role="combobox"
        :aria-expanded="open"
        :disabled="disabled"
        class="w-full justify-between gap-3"
      >
        <span class="truncate text-left" :class="cn(!selectedLabel && 'text-muted-foreground')">
          {{ triggerLabel }}
        </span>
        <ChevronsUpDown class="h-4 w-4 shrink-0 opacity-60" />
      </Button>
    </PopoverTrigger>

    <PopoverContent class="w-(--reka-popper-anchor-width) min-w-[320px] p-0" align="start">
      <div class="border-b px-3 py-3">
        <div class="relative">
          <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            v-model="searchTerm"
            class="pl-9"
            placeholder="Cari bank_name dari payment catalog"
          />
        </div>
      </div>

      <div class="max-h-72 overflow-y-auto p-2">
        <div v-if="loading" class="flex items-center justify-center gap-2 px-3 py-8 text-sm text-muted-foreground">
          <Spinner class="h-4 w-4" />
          <span>Memuat bank options...</span>
        </div>

        <div v-else-if="errorMessage" class="flex items-start gap-2 rounded-lg border border-destructive/20 bg-destructive/5 px-3 py-3 text-sm text-destructive">
          <TriangleAlert class="mt-0.5 h-4 w-4 shrink-0" />
          <span>{{ errorMessage }}</span>
        </div>

        <div v-else-if="options.length === 0" class="px-3 py-8 text-center text-sm text-muted-foreground">
          Tidak ada bank yang cocok dengan pencarian ini.
        </div>

        <div v-else class="space-y-1">
          <button
            v-for="option in options"
            :key="option.id"
            type="button"
            class="flex w-full items-center justify-between rounded-lg px-3 py-2 text-left text-sm transition-colors hover:bg-accent hover:text-accent-foreground"
            @click="handleSelect(option)"
          >
            <span class="truncate pr-3">{{ option.bank_name }}</span>
            <Check v-if="option.id === modelValue" class="h-4 w-4 shrink-0 text-primary" />
          </button>
        </div>
      </div>
    </PopoverContent>
  </Popover>
</template>
