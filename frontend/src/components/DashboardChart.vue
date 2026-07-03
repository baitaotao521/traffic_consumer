<script setup lang="ts">
import {
  ArcElement,
  CategoryScale,
  Chart,
  DoughnutController,
  Filler,
  Legend,
  LinearScale,
  LineController,
  LineElement,
  PointElement,
  Tooltip,
} from 'chart.js'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'

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

Chart.register(
  LineController,
  DoughnutController,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  ArcElement,
  Filler,
  Tooltip,
  Legend,
)

const canvas = ref<HTMLCanvasElement | null>(null)
let chart: Chart | null = null

const hasData = computed(() => props.values.some((value) => value > 0))

function chartValues() {
  if (props.type === 'doughnut' && !hasData.value) {
    return [1]
  }
  return props.values
}

function chartLabels() {
  if (props.type === 'doughnut' && !hasData.value) {
    return ['无数据']
  }
  return props.labels
}

function chartColors() {
  if (props.type === 'doughnut' && !hasData.value) {
    return [resolveThemeColor('--muted')]
  }
  return props.values.map((_, index) => resolveThemeColor(props.colors[index % props.colors.length]))
}

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

function renderChart() {
  if (!canvas.value) return
  const primary = resolveThemeColor('--primary')
  const mutedForeground = resolveThemeColor('--muted-foreground')
  const border = withAlpha('--border', 0.65)

  chart?.destroy()
  chart = new Chart(canvas.value, {
    type: props.type,
    data: {
      labels: chartLabels(),
      datasets: [
        props.type === 'line'
          ? {
              label: props.label || '实时速度',
              data: props.values,
              borderColor: primary,
              backgroundColor: withAlpha('--primary', 0.12),
              fill: true,
              pointRadius: 0,
              tension: 0.35,
            }
          : {
              label: props.label || '占比',
              data: chartValues(),
              backgroundColor: chartColors(),
              borderWidth: 0,
            },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      animation: {
        duration: props.type === 'line' ? 280 : 420,
        easing: 'easeOutCubic',
      },
      cutout: props.type === 'doughnut' ? '64%' : undefined,
      plugins: {
        legend: {
          display: props.type === 'doughnut',
          position: 'bottom',
          labels: { boxWidth: 10, boxHeight: 10, color: mutedForeground, usePointStyle: true },
        },
      },
      scales:
        props.type === 'line'
          ? {
              x: { display: false },
              y: { beginAtZero: true, grid: { color: border }, ticks: { color: mutedForeground }, border: { color: border } },
            }
          : undefined,
    },
  })
}

watch(() => [props.type, props.labels, props.values, props.colors], renderChart, { deep: true })

onMounted(renderChart)

onBeforeUnmount(() => chart?.destroy())
</script>

<template>
  <div class="relative w-full" :style="{ height: `${height}px` }">
    <canvas :id="chartId" ref="canvas" />
  </div>
</template>
