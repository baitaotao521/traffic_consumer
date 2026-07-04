<script setup lang="ts">
import type { EChartsOption } from 'echarts'
import { use } from 'echarts/core'
import { LineChart, PieChart } from 'echarts/charts'
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { computed } from 'vue'
import VChart from 'vue-echarts'

const fallbackColors: Record<string, string> = {
  '--background': 'oklch(1 0 0)',
  '--foreground': 'oklch(0.145 0 0)',
  '--primary': 'oklch(0.205 0 0)',
  '--muted': 'oklch(0.97 0 0)',
  '--muted-foreground': 'oklch(0.556 0 0)',
  '--border': 'oklch(0.922 0 0)',
  '--chart-1': 'oklch(0.87 0 0)',
  '--chart-2': 'oklch(0.556 0 0)',
  '--chart-3': 'oklch(0.439 0 0)',
  '--chart-4': 'oklch(0.371 0 0)',
  '--chart-5': 'oklch(0.269 0 0)',
}

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

use([CanvasRenderer, LineChart, PieChart, GridComponent, TooltipComponent, LegendComponent])

const initOptions = { renderer: 'canvas' } as const
const hasData = computed(() => props.values.some((value) => value > 0))

function resolveThemeColor(value: string) {
  if (!value.startsWith('--')) return value
  const fallback = fallbackColors[value] || fallbackColors['--foreground']
  if (typeof window === 'undefined') return fallback
  return window.getComputedStyle(document.documentElement).getPropertyValue(value).trim() || fallback
}

function withAlpha(value: string, alpha: number) {
  const color = resolveThemeColor(value)
  if (color.startsWith('oklch(') && !color.includes('/')) {
    return color.replace(/\)$/, ` / ${alpha})`)
  }
  return color
}

function chartLabels() {
  if (props.type === 'doughnut' && !hasData.value) {
    return ['无数据']
  }
  return [...props.labels]
}

function doughnutValues() {
  if (!hasData.value) {
    return [1]
  }
  return [...props.values]
}

function doughnutColors() {
  if (!hasData.value) {
    return [resolveThemeColor('--muted')]
  }
  return props.values.map((_, index) => resolveThemeColor(props.colors[index % props.colors.length]))
}

const currentOption = computed<EChartsOption>(() => {
  const mutedForeground = resolveThemeColor('--muted-foreground')
  const border = withAlpha('--border', 0.65)

  if (props.type === 'line') {
    return {
      animation: false,
      grid: {
        left: 0,
        right: 8,
        top: 8,
        bottom: 0,
        containLabel: true,
      },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: [...props.labels],
        show: false,
      },
      yAxis: {
        type: 'value',
        min: 0,
        splitLine: {
          lineStyle: {
            color: border,
          },
        },
        axisLabel: {
          color: mutedForeground,
        },
        axisLine: {
          lineStyle: {
            color: border,
          },
        },
      },
      tooltip: {
        trigger: 'axis',
        borderWidth: 0,
        backgroundColor: resolveThemeColor('--background'),
        textStyle: {
          color: resolveThemeColor('--foreground'),
        },
      },
      series: [
        {
          name: props.label || '实时速度',
          id: `${props.chartId}-series`,
          type: 'line',
          data: [...props.values],
          symbol: 'none',
          smooth: 0.35,
          lineStyle: {
            color: resolveThemeColor('--primary'),
            width: 2,
          },
          areaStyle: {
            color: withAlpha('--primary', 0.12),
          },
          emphasis: {
            disabled: true,
          },
        },
      ],
    }
  }

  return {
    animation: false,
    tooltip: {
      trigger: 'item',
      borderWidth: 0,
      backgroundColor: resolveThemeColor('--background'),
      textStyle: {
        color: resolveThemeColor('--foreground'),
      },
    },
    legend: {
      show: hasData.value,
      bottom: 0,
      textStyle: {
        color: mutedForeground,
      },
      itemWidth: 10,
      itemHeight: 10,
    },
    series: [
      {
        name: props.label || '占比',
        id: `${props.chartId}-series`,
        type: 'pie',
        radius: ['64%', '100%'],
        avoidLabelOverlap: true,
        label: {
          show: false,
        },
        emphasis: {
          disabled: true,
        },
        data: chartLabels().map((name, index) => ({
          name,
          value: doughnutValues()[index],
          itemStyle: {
            color: doughnutColors()[index],
          },
        })),
      },
    ],
  }
})
</script>

<template>
  <div class="relative w-full" :style="{ height: `${height}px` }">
    <VChart
      :id="chartId"
      :option="currentOption"
      :init-options="initOptions"
      class="h-full w-full"
    />
  </div>
</template>
