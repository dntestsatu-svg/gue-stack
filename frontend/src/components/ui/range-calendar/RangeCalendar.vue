<script setup lang="ts">
import { ChevronLeft, ChevronRight } from 'lucide-vue-next'
import { computed } from 'vue'
import {
  type DateRange,
  RangeCalendarCell,
  RangeCalendarCellTrigger,
  RangeCalendarGrid,
  RangeCalendarGridBody,
  RangeCalendarGridHead,
  RangeCalendarGridRow,
  RangeCalendarHeadCell,
  RangeCalendarHeader,
  RangeCalendarHeading,
  RangeCalendarNext,
  RangeCalendarPrev,
  RangeCalendarRoot,
} from 'reka-ui'
import { cn } from '@/lib/utils'

const props = withDefaults(
  defineProps<{
    modelValue?: DateRange | null
    numberOfMonths?: number
    disabled?: boolean
    class?: string
  }>(),
  {
    modelValue: null,
    numberOfMonths: 1,
    disabled: false,
    class: '',
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: DateRange | undefined]
}>()

const value = computed<DateRange | undefined>({
  get: () => props.modelValue ?? undefined,
  set: (next) => emit('update:modelValue', next),
})
</script>

<template>
  <RangeCalendarRoot
    v-model="value"
    :number-of-months="props.numberOfMonths"
    :disabled="props.disabled"
    fixed-weeks
    :class="
      cn(
        'rounded-md border border-[var(--border)] bg-[var(--background-elevated)] p-3',
        props.class,
      )
    "
  >
    <template #default="{ grid, weekDays }">
      <div class="flex flex-col gap-4 sm:flex-row">
        <div
          v-for="(month, monthIndex) in grid"
          :key="month.value.toString()"
          class="space-y-3"
        >
          <RangeCalendarHeader class="relative flex w-full items-center justify-center">
            <RangeCalendarPrev
              v-if="monthIndex === 0"
              class="absolute left-1 inline-flex h-7 w-7 items-center justify-center rounded-md border border-[var(--border)] bg-[var(--background)] text-[var(--foreground)] hover:bg-[var(--background-muted)]"
            >
              <ChevronLeft class="h-4 w-4" />
            </RangeCalendarPrev>
            <RangeCalendarHeading class="text-sm font-medium text-[var(--foreground)]" />
            <RangeCalendarNext
              v-if="monthIndex === grid.length - 1"
              class="absolute right-1 inline-flex h-7 w-7 items-center justify-center rounded-md border border-[var(--border)] bg-[var(--background)] text-[var(--foreground)] hover:bg-[var(--background-muted)]"
            >
              <ChevronRight class="h-4 w-4" />
            </RangeCalendarNext>
          </RangeCalendarHeader>

          <RangeCalendarGrid :month="month.value" class="w-full border-collapse">
            <RangeCalendarGridHead>
              <RangeCalendarGridRow class="flex">
                <RangeCalendarHeadCell
                  v-for="day in weekDays"
                  :key="day"
                  class="h-9 w-9 text-xs font-medium text-[var(--muted-foreground)]"
                >
                  {{ day }}
                </RangeCalendarHeadCell>
              </RangeCalendarGridRow>
            </RangeCalendarGridHead>
            <RangeCalendarGridBody>
              <RangeCalendarGridRow
                v-for="(weekDates, index) in month.rows"
                :key="`week-${monthIndex}-${index}`"
                class="mt-1 flex w-full"
              >
                <RangeCalendarCell
                  v-for="weekDate in weekDates"
                  :key="weekDate.toString()"
                  :date="weekDate"
                  class="relative h-9 w-9 p-0 text-center text-sm"
                >
                  <RangeCalendarCellTrigger
                    :day="weekDate"
                    :month="month.value"
                    class="inline-flex h-9 w-9 items-center justify-center rounded-md text-sm text-[var(--foreground)] outline-none transition-colors hover:bg-[var(--background-muted)] focus-visible:ring-2 focus-visible:ring-[var(--ring)] data-[outside-view]:text-[var(--muted-foreground)] data-[outside-view]:opacity-60 data-[disabled]:cursor-not-allowed data-[disabled]:opacity-30 data-[unavailable]:cursor-not-allowed data-[unavailable]:opacity-30 data-[highlighted]:bg-[color-mix(in_oklab,var(--primary)_15%,transparent)] data-[selected]:bg-[var(--primary)] data-[selected]:text-[var(--primary-foreground)] data-[selection-start]:rounded-r-none data-[selection-end]:rounded-l-none"
                  />
                </RangeCalendarCell>
              </RangeCalendarGridRow>
            </RangeCalendarGridBody>
          </RangeCalendarGrid>
        </div>
      </div>
    </template>
  </RangeCalendarRoot>
</template>
