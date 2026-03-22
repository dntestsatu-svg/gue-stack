<template>
  <div class="date-interval-picker" :class="{ 'pointer-events-none opacity-70': disabled }">
    <div class="date-interval-field">
      <span class="date-interval-label">From</span>
      <div class="date-interval-input-shell">
        <CalendarRange class="date-interval-icon" />
        <Input
          :disabled="disabled"
          :model-value="modelValue.from"
          type="date"
          class-name="date-interval-input"
          @update:model-value="updateField('from', $event)"
        />
      </div>
    </div>

    <div class="date-interval-field">
      <span class="date-interval-label">To</span>
      <div class="date-interval-input-shell">
        <CalendarRange class="date-interval-icon" />
        <Input
          :disabled="disabled"
          :model-value="modelValue.to"
          type="date"
          class-name="date-interval-input"
          @update:model-value="updateField('to', $event)"
        />
      </div>
    </div>

    <div class="date-interval-presets">
      <Button size="sm" variant="ghost" type="button" class="date-interval-preset" @click="setToday">
        Today
      </Button>
      <Button size="sm" variant="ghost" type="button" class="date-interval-preset" @click="setLastDays(7)">
        Last 7 Days
      </Button>
      <Button size="sm" variant="ghost" type="button" class="date-interval-preset" @click="setLastDays(30)">
        Last 30 Days
      </Button>
      <Button size="sm" variant="ghost" type="button" class="date-interval-preset" @click="clearDates">
        Clear
      </Button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { CalendarRange } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

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

const emitRange = (nextValue: DateRangeValue, changedField?: keyof DateRangeValue) => {
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
  const today = formatInputDate(new Date())
  emitRange({
    from: today,
    to: today,
  })
}

function setLastDays(days: number) {
  const today = new Date()
  const from = new Date(today)
  from.setDate(today.getDate() - (days - 1))

  emitRange({
    from: formatInputDate(from),
    to: formatInputDate(today),
  })
}

function clearDates() {
  emitRange({
    from: '',
    to: '',
  })
}
</script>
