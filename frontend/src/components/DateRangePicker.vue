<template>
  <div class="grid gap-3">
    <div class="grid gap-3 md:grid-cols-2">
      <div class="space-y-1.5">
        <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">From</p>
        <Popover v-model:open="openFrom">
          <PopoverTrigger as-child>
            <Button
              variant="outline"
              :disabled="disabled"
              :class="
                cn(
                  'h-11 w-full justify-start rounded-xl border-border bg-(--background-elevated) px-3 text-left font-medium shadow-none',
                  !props.modelValue.from && 'text-muted-foreground',
                )
              "
            >
              <CalendarIcon class="mr-2 h-4 w-4 shrink-0" />
              <span class="truncate">{{ fromLabel }}</span>
            </Button>
          </PopoverTrigger>
          <PopoverContent class="w-auto rounded-2xl border-border bg-(--background-elevated) p-2" align="start">
            <Calendar
              v-model="fromValue"
              :default-placeholder="defaultPlaceholder"
              layout="month-and-year"
              initial-focus
              @update:model-value="openFrom = false"
            />
          </PopoverContent>
        </Popover>
      </div>

      <div class="space-y-1.5">
        <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">To</p>
        <Popover v-model:open="openTo">
          <PopoverTrigger as-child>
            <Button
              variant="outline"
              :disabled="disabled"
              :class="
                cn(
                  'h-11 w-full justify-start rounded-xl border-border bg-(--background-elevated) px-3 text-left font-medium shadow-none',
                  !props.modelValue.to && 'text-muted-foreground',
                )
              "
            >
              <CalendarIcon class="mr-2 h-4 w-4 shrink-0" />
              <span class="truncate">{{ toLabel }}</span>
            </Button>
          </PopoverTrigger>
          <PopoverContent class="w-auto rounded-2xl border-border bg-(--background-elevated) p-2" align="start">
            <Calendar
              v-model="toValue"
              :default-placeholder="defaultPlaceholder"
              layout="month-and-year"
              initial-focus
              @update:model-value="openTo = false"
            />
          </PopoverContent>
        </Popover>
      </div>
    </div>

    <div class="flex flex-wrap items-center gap-2">
      <Button size="sm" variant="outline" class="rounded-lg" :disabled="disabled" @click="setToday">
        Today
      </Button>
      <Button size="sm" variant="outline" class="rounded-lg" :disabled="disabled" @click="setLastDays(7)">
        Last 7 Days
      </Button>
      <Button size="sm" variant="outline" class="rounded-lg" :disabled="disabled" @click="setLastDays(30)">
        Last 30 Days
      </Button>
      <Button size="sm" variant="ghost" class="rounded-lg text-muted-foreground" :disabled="disabled" @click="clearDates">
        Clear
      </Button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { DateValue } from '@internationalized/date'
import { DateFormatter, getLocalTimeZone, parseDate, today } from '@internationalized/date'
import { CalendarIcon } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'

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

const openFrom = ref(false)
const openTo = ref(false)
const defaultPlaceholder = today(getLocalTimeZone())
const formatter = new DateFormatter('id-ID', { dateStyle: 'medium' })

const fromLabel = computed(() => formatLabel(props.modelValue.from, 'Choose start date'))
const toLabel = computed(() => formatLabel(props.modelValue.to, 'Choose end date'))

const fromValue = computed<DateValue | undefined>({
  get: () => parseDateSafe(props.modelValue.from),
  set: (value) => updateField('from', toDateString(value)),
})

const toValue = computed<DateValue | undefined>({
  get: () => parseDateSafe(props.modelValue.to),
  set: (value) => updateField('to', toDateString(value)),
})

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

function formatLabel(value: string, fallback: string): string {
  const date = parseDateSafe(value)
  if (!date) {
    return fallback
  }
  return formatter.format(date.toDate(getLocalTimeZone()))
}

function emitRange(nextValue: DateRangeValue, changedField?: keyof DateRangeValue) {
  const normalized = normalizeRange(nextValue, changedField)
  emit('update:modelValue', normalized)
}

function normalizeRange(value: DateRangeValue, changedField?: keyof DateRangeValue): DateRangeValue {
  const next = {
    from: value.from ?? '',
    to: value.to ?? '',
  }

  if (next.from !== '' && next.to !== '' && next.from > next.to) {
    if (changedField === 'to') {
      next.from = next.to
    } else {
      next.to = next.from
    }
  }

  return next
}

function updateField(field: keyof DateRangeValue, value: string) {
  emitRange(
    {
      from: field === 'from' ? value : props.modelValue.from,
      to: field === 'to' ? value : props.modelValue.to,
    },
    field,
  )
}

function formatInputDate(date: Date): string {
  const year = date.getFullYear()
  const month = `${date.getMonth() + 1}`.padStart(2, '0')
  const day = `${date.getDate()}`.padStart(2, '0')
  return `${year}-${month}-${day}`
}

function setToday() {
  const currentDay = formatInputDate(new Date())
  emitRange({
    from: currentDay,
    to: currentDay,
  })
}

function setLastDays(days: number) {
  const currentDay = new Date()
  const start = new Date(currentDay)
  start.setDate(currentDay.getDate() - (days - 1))

  emitRange({
    from: formatInputDate(start),
    to: formatInputDate(currentDay),
  })
}

function clearDates() {
  emitRange({
    from: '',
    to: '',
  })
}
</script>
