<script setup lang="ts">
import type { DateValue } from 'reka-ui'
import { ChevronLeft, ChevronRight } from 'lucide-vue-next'
import { computed } from 'vue'
import {
  CalendarCell,
  CalendarCellTrigger,
  CalendarGrid,
  CalendarGridBody,
  CalendarGridHead,
  CalendarGridRow,
  CalendarHeadCell,
  CalendarHeader,
  CalendarHeading,
  CalendarNext,
  CalendarPrev,
  CalendarRoot,
} from 'reka-ui'
import { cn } from '@/lib/utils'

defineOptions({
  inheritAttrs: false,
})

const props = withDefaults(
  defineProps<{
    modelValue?: DateValue | null
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
  'update:modelValue': [value: DateValue | undefined]
}>()

const value = computed<DateValue | undefined>({
  get: () => props.modelValue ?? undefined,
  set: (next) => emit('update:modelValue', next),
})
</script>

<template>
  <CalendarRoot
    v-model="value"
    v-bind="$attrs"
    :number-of-months="props.numberOfMonths"
    :disabled="props.disabled"
    fixed-weeks
    :class="
      cn(
        'rounded-xl border border-border bg-(--background-elevated) p-3 shadow-[0_18px_44px_-34px_color-mix(in_oklab,#000_28%,transparent)]',
        props.class,
      )
    "
  >
    <template #default="{ grid, weekDays }">
      <div class="flex flex-col gap-4">
        <div
          v-for="month in grid"
          :key="month.value.toString()"
          class="space-y-3"
        >
          <CalendarHeader class="relative flex items-center justify-center">
            <CalendarPrev
              class="absolute left-0 inline-flex h-8 w-8 items-center justify-center rounded-lg border border-border bg-background text-foreground transition-colors hover:bg-(--background-muted)"
            >
              <ChevronLeft class="h-4 w-4" />
            </CalendarPrev>
            <CalendarHeading class="text-sm font-semibold text-foreground" />
            <CalendarNext
              class="absolute right-0 inline-flex h-8 w-8 items-center justify-center rounded-lg border border-border bg-background text-foreground transition-colors hover:bg-(--background-muted)"
            >
              <ChevronRight class="h-4 w-4" />
            </CalendarNext>
          </CalendarHeader>

          <CalendarGrid :month="month.value" class="w-full border-collapse">
            <CalendarGridHead>
              <CalendarGridRow class="flex">
                <CalendarHeadCell
                  v-for="day in weekDays"
                  :key="day"
                  class="h-9 w-9 text-center text-xs font-medium text-muted-foreground"
                >
                  {{ day }}
                </CalendarHeadCell>
              </CalendarGridRow>
            </CalendarGridHead>
            <CalendarGridBody>
              <CalendarGridRow
                v-for="(weekDates, index) in month.rows"
                :key="`${month.value.toString()}-${index}`"
                class="mt-1 flex w-full"
              >
                <CalendarCell
                  v-for="weekDate in weekDates"
                  :key="weekDate.toString()"
                  :date="weekDate"
                  class="relative h-9 w-9 p-0 text-center text-sm"
                >
                  <CalendarCellTrigger
                    :day="weekDate"
                    :month="month.value"
                    class="inline-flex h-9 w-9 items-center justify-center rounded-lg text-sm text-foreground outline-none transition-colors hover:bg-(--background-muted) focus-visible:ring-2 focus-visible:ring-ring data-outside-view:text-muted-foreground data-outside-view:opacity-45 data-disabled:cursor-not-allowed data-disabled:opacity-30 data-unavailable:cursor-not-allowed data-unavailable:opacity-30 data-today:bg-[color-mix(in_oklab,var(--brand)_10%,transparent)] data-selected:bg-(--brand) data-selected:text-white"
                  />
                </CalendarCell>
              </CalendarGridRow>
            </CalendarGridBody>
          </CalendarGrid>
        </div>
      </div>
    </template>
  </CalendarRoot>
</template>
