<script setup lang="ts">
import { computed } from 'vue'
import type { ChartConfig } from '@/components/ui/chart'
import {
  ChartContainer,
  ChartCrosshair,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  componentToString,
} from '@/components/ui/chart'
import type { DashboardStatusSeriesPoint } from '@/services/types'
import { VisArea, VisAxis, VisLine, VisXYContainer } from '@unovis/vue'

type StatusAreaPoint = {
  bucketIndex: number
  bucketLabel: string
  success: number
  failedExpired: number
}

const props = defineProps<{
  series: DashboardStatusSeriesPoint[]
}>()

const chartConfig = {
  failedExpired: {
    label: 'Failed / Expired',
    color: 'var(--chart-failed)',
  },
  success: {
    label: 'Success',
    color: 'var(--chart-success)',
  },
} satisfies ChartConfig

const chartData = computed<StatusAreaPoint[]>(() =>
  props.series.map((point, index) => {
    const bucketDate = new Date(point.bucket)
    return {
      bucketIndex: index + 1,
      bucketLabel: bucketDate.toLocaleTimeString('id-ID', {
        hour: '2-digit',
        minute: '2-digit',
        timeZone: 'UTC',
      }),
      success: point.success_count,
      failedExpired: point.failed_expired_count,
    }
  }),
)

const tickValues = computed(() => chartData.value.map(point => point.bucketIndex))

function tickLabel(_value: number, index: number) {
  return chartData.value[index]?.bucketLabel ?? ''
}

function yTickLabel(value: number) {
  return value.toLocaleString('id-ID')
}
</script>

<template>
  <ChartContainer
    :config="chartConfig"
    cursor
    class="min-h-80 w-full rounded-xl border border-border/70 bg-(--background-muted)/35 p-4"
    data-testid="dashboard-status-chart"
  >
    <ChartLegendContent class="justify-start pt-0 pb-4" />
    <VisXYContainer :data="chartData">
      <VisArea
        :x="(d: StatusAreaPoint) => d.bucketIndex"
        :y="[(d: StatusAreaPoint) => d.failedExpired, (d: StatusAreaPoint) => d.success]"
        :color="(_d: StatusAreaPoint, index: number) => [chartConfig.failedExpired.color, chartConfig.success.color][index]"
        :opacity="0.3"
      />
      <VisLine
        :x="(d: StatusAreaPoint) => d.bucketIndex"
        :y="[(d: StatusAreaPoint) => d.failedExpired, (d: StatusAreaPoint) => d.failedExpired + d.success]"
        :color="(_d: StatusAreaPoint, index: number) => [chartConfig.failedExpired.color, chartConfig.success.color][index]"
        :line-width="2"
      />
      <VisAxis
        type="x"
        :x="(d: StatusAreaPoint) => d.bucketIndex"
        :tick-line="false"
        :domain-line="false"
        :grid-line="false"
        :num-ticks="chartData.length"
        :tick-values="tickValues"
        :tick-format="tickLabel"
      />
      <VisAxis
        type="y"
        :num-ticks="4"
        :tick-line="false"
        :domain-line="false"
        :grid-line="true"
        :tick-format="yTickLabel"
      />
      <ChartTooltip />
      <ChartCrosshair
        :template="componentToString(chartConfig, ChartTooltipContent, { labelKey: 'bucketLabel' })"
        :color="(_d: StatusAreaPoint, index: number) => [chartConfig.failedExpired.color, chartConfig.success.color][index % 2]"
      />
    </VisXYContainer>
  </ChartContainer>
</template>
