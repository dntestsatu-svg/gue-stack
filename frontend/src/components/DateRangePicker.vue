<template>
  <Popover v-model:open="open">
    <PopoverTrigger as-child>
      <Button
        variant="outline"
        class="w-full justify-between border-[var(--border)] bg-[var(--background-elevated)] text-left font-normal"
        :disabled="disabled"
      >
        <span class="flex min-w-0 items-center gap-2">
          <CalendarRange class="h-4 w-4 shrink-0 text-[var(--muted-foreground)]" />
          <span class="truncate" :class="!hasSelection ? 'text-[var(--muted-foreground)]' : ''">
            {{ triggerLabel }}
          </span>
        </span>
      </Button>
    </PopoverTrigger>
    <PopoverContent class="w-[min(94vw,28rem)] border-[var(--border)] bg-[var(--background-elevated)] p-0">
      <div class="space-y-4 p-4">
        <div class="space-y-1">
          <p class="text-sm font-medium text-[var(--foreground)]">Transaction Range</p>
          <p class="text-xs text-[var(--muted-foreground)]">
            Pilih tanggal menggunakan calendar shadcn-vue.
          </p>
        </div>

        <div class="grid grid-cols-2 gap-2">
          <Button size="sm" variant="outline" type="button" @click="setToday">Today</Button>
          <Button size="sm" variant="outline" type="button" @click="setLastDays(7)">Last 7 Days</Button>
          <Button size="sm" variant="outline" type="button" @click="setLastDays(30)">Last 30 Days</Button>
          <Button size="sm" variant="outline" type="button" @click="clearDraft">Clear</Button>
        </div>

        <RangeCalendar v-model="calendarRange" :disabled="disabled" :number-of-months="1" />

        <div class="flex items-center justify-between gap-2">
          <p class="text-xs text-[var(--muted-foreground)]">{{ rangeCaption }}</p>
          <div class="flex items-center gap-2">
            <Button size="sm" variant="ghost" type="button" @click="resetDraft">Cancel</Button>
            <Button size="sm" type="button" @click="applyDraft">Apply Range</Button>
          </div>
        </div>
      </div>
    </PopoverContent>
  </Popover>
</template>

<script setup lang="ts">
import type { DateRange, DateValue } from 'reka-ui'
import { parseDate } from '@internationalized/date'
import { computed, reactive, ref, watch } from 'vue'
import { CalendarRange } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { RangeCalendar } from '@/components/ui/range-calendar'

type DateRangeValue = {
  from: string
  to: string
}

const props = withDefaults(defineProps<{
  modelValue: DateRangeValue
  disabled?: boolean
}>(), {
  disabled: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: DateRangeValue]
}>()

const open = ref(false)
const draft = reactive<DateRangeValue>({
  from: '',
  to: '',
})

const formatter = new Intl.DateTimeFormat('id-ID', {
  day: '2-digit',
  month: 'short',
  year: 'numeric',
})

const hasSelection = computed(() => draft.from !== '' || draft.to !== '')

const triggerLabel = computed(() => {
  if (draft.from && draft.to) {
    return `${formatDateLabel(draft.from)} - ${formatDateLabel(draft.to)}`
  }
  if (draft.from) {
    return `From ${formatDateLabel(draft.from)}`
  }
  if (draft.to) {
    return `Until ${formatDateLabel(draft.to)}`
  }
  return 'Pick a date range'
})

const rangeCaption = computed(() => {
  if (!draft.from && !draft.to) {
    return 'No date range selected'
  }
  if (draft.from && !draft.to) {
    return `Start: ${formatDateLabel(draft.from)}`
  }
  if (!draft.from && draft.to) {
    return `End: ${formatDateLabel(draft.to)}`
  }
  return `${formatDateLabel(draft.from)} - ${formatDateLabel(draft.to)}`
})

const calendarRange = computed<DateRange | undefined>({
  get: () => {
    const start = parseDateSafe(draft.from)
    const end = parseDateSafe(draft.to)
    if (!start && !end) {
      return undefined
    }
    return {
      start: start ?? end ?? undefined,
      end: end ?? start ?? undefined,
    }
  },
  set: (value) => {
    draft.from = toDateString(value?.start)
    draft.to = toDateString(value?.end)
  },
})

const syncDraft = (value: DateRangeValue) => {
  draft.from = value.from ?? ''
  draft.to = value.to ?? ''
}

watch(
  () => props.modelValue,
  (value) => syncDraft(value),
  { immediate: true, deep: true },
)

function parseDateSafe(value: string): DateValue | undefined {
  if (!value) {
    return undefined
  }
  try {
    return parseDate(value)
  } catch {
    return undefined
  }
}

function toDateString(value?: DateValue): string {
  return value ? value.toString() : ''
}

function formatDateLabel(value: string): string {
  const date = new Date(`${value}T00:00:00`)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return formatter.format(date)
}

function formatInputDate(date: Date): string {
  const year = date.getFullYear()
  const month = `${date.getMonth() + 1}`.padStart(2, '0')
  const day = `${date.getDate()}`.padStart(2, '0')
  return `${year}-${month}-${day}`
}

function setToday() {
  const today = formatInputDate(new Date())
  draft.from = today
  draft.to = today
}

function setLastDays(days: number) {
  const today = new Date()
  const from = new Date(today)
  from.setDate(today.getDate() - (days - 1))
  draft.from = formatInputDate(from)
  draft.to = formatInputDate(today)
}

function clearDraft() {
  draft.from = ''
  draft.to = ''
}

function resetDraft() {
  syncDraft(props.modelValue)
  open.value = false
}

function applyDraft() {
  emit('update:modelValue', {
    from: draft.from,
    to: draft.to,
  })
  open.value = false
}
</script>

