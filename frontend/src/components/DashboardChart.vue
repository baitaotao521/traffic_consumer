<script setup lang="ts">
import type { ChartConfig } from '@/components/ui/chart'
import { VisDonut, VisDonutSelectors, VisLine, VisLineSelectors, VisSingleContainer, VisXYContainer, VisAxis } from '@unovis/vue'
import { computed } from 'vue'
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  componentToString,
} from '@/components/ui/chart'

type LineDatum = {
  label: string
  value: number
}

type DonutDatum = {
  key: string
  label: string
  value: number
} & Record<string, number | string>

const props = withDefaults(
  defineProps<{
    chartId: string
    type: 'line' | 'doughnut'
    labels: string[]
    values: number[]
    colors?: string[]
    label?: string
    height?: number
  }>(),
  {
    colors: () => ['--chart-1', '--chart-2', '--chart-3', '--chart-4', '--chart-5'],
    label: '',
    height: 240,
  },
)
const hasData = computed(() => props.values.some((value) => value > 0))

function resolveChartColor(value: string) {
  return value.startsWith('--') ? `var(${value})` : value
}

const lineData = computed<LineDatum[]>(() => {
  return props.labels.map((label, index) => ({
    label,
    value: props.values[index] ?? 0,
  }))
})

const lineChartConfig = computed<ChartConfig>(() => ({
  value: {
    label: props.label || '实时速度',
    color: resolveChartColor(props.colors[0]),
  },
}))
const lineDomainMax = computed(() => Math.max(1, ...lineData.value.map((item) => item.value)))
const lineTooltip = computed(() => {
  return componentToString(lineChartConfig.value, ChartTooltipContent, {
    indicator: 'line',
    labelKey: 'label',
  })
})
const lineTooltipTriggers = computed(() => {
  if (!lineTooltip.value) return {}
  return {
    [VisLineSelectors.line]: lineTooltip.value,
    [VisLineSelectors.lineSelectionHelper]: lineTooltip.value,
  }
})

const donutLabels = computed(() => {
  if (!hasData.value) return ['无数据']
  return [...props.labels]
})
const donutValues = computed(() => {
  if (!hasData.value) return [1]
  return [...props.values]
})
const donutChartConfig = computed<ChartConfig>(() => {
  return Object.fromEntries(
    donutLabels.value.map((label, index) => [
      `segment${index}`,
      {
        label,
        color: hasData.value
          ? resolveChartColor(props.colors[index % props.colors.length])
          : 'var(--muted)',
      },
    ]),
  )
})
const donutData = computed<DonutDatum[]>(() => {
  return donutLabels.value.map((label, index) => {
    const key = `segment${index}`
    return {
      key,
      label,
      value: donutValues.value[index] ?? 0,
      [key]: donutValues.value[index] ?? 0,
    }
  })
})
const donutTooltip = computed(() => {
  return componentToString(donutChartConfig.value, ChartTooltipContent, {
    labelKey: 'label',
  })
})
const donutTooltipTriggers = computed(() => {
  if (!donutTooltip.value) return {}
  return {
    [VisDonutSelectors.segment]: donutTooltip.value,
  }
})

function lineX(_: LineDatum, index: number) {
  return index
}

function lineY(datum: LineDatum) {
  return datum.value
}

function formatLineTick(tick: number | Date) {
  const index = Math.round(Number(tick))
  return lineData.value[index]?.label ?? ''
}

function donutValue(datum: DonutDatum) {
  return Number(datum.value)
}

function donutColor(datum: DonutDatum) {
  return `var(--color-${datum.key})`
}
</script>

<template>
  <div class="relative w-full" :style="{ height: `${height}px` }">
    <ChartContainer
      v-if="type === 'line'"
      :config="lineChartConfig"
      class="h-full w-full"
    >
      <VisXYContainer
        :data="lineData"
        :x-domain="[0, Math.max(lineData.length - 1, 0)]"
        :y-domain="[0, lineDomainMax]"
      >
        <VisLine
          :color="() => 'var(--color-value)'"
          :line-width="2"
          :x="lineX"
          :y="lineY"
        />
        <VisAxis
          :domain-line="false"
          :grid-line="false"
          :min-max-ticks-only="true"
          :tick-format="formatLineTick"
          :tick-line="false"
          type="x"
        />
        <VisAxis
          :domain-line="false"
          :num-ticks="4"
          :tick-line="false"
          type="y"
        />
        <ChartTooltip :triggers="lineTooltipTriggers" />
      </VisXYContainer>
    </ChartContainer>

    <ChartContainer
      v-else
      :config="donutChartConfig"
      class="h-full w-full"
    >
      <VisSingleContainer :data="donutData">
        <ChartTooltip :triggers="donutTooltipTriggers" />
        <VisDonut
          :arc-width="26"
          :central-label="hasData ? (label || 'URL') : '无数据'"
          :central-sub-label="hasData ? '负载分布' : '暂无下载记录'"
          :color="donutColor"
          :value="donutValue"
        />
      </VisSingleContainer>
    </ChartContainer>
  </div>
</template>
