<script setup lang="ts">
import { computed } from 'vue'
import { RefreshCwIcon } from '@lucide/vue'

import DashboardChart from '@/components/DashboardChart.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import type { URLUsageStat } from '@/types'

const props = defineProps<{
  /** 当前速度保留后端格式化文案，避免综合指数替代真实单位。 */
  currentSpeed: string
  /** 速率趋势采样时间标签，与 values 下标一一对应。 */
  labels: string[]
  /** 手动刷新 dashboard 快照时展示按钮加载状态。 */
  refreshing?: boolean
  /** 线程活跃度历史采样，独立使用百分比坐标轴展示。 */
  threadActiveValues: number[]
  /** URL 饼图标签与后端 URL 使用统计保持同一排序。 */
  urlChartLabels: string[]
  /** URL 饼图数值使用下载次数，单独表达 URL 分布。 */
  urlChartValues: number[]
  /** URL 分布明细包含后端计算的下载次数和占比。 */
  urlStats: URLUsageStat[]
  /** 实时速率采样值，单位为 MB/s。 */
  values: number[]
}>()

defineEmits<{
  refresh: []
}>()

const currentThreadActive = computed(() => props.threadActiveValues.at(-1) ?? 0)

function clampIndex(value: number) {
  if (!Number.isFinite(value) || value <= 0) return 0
  if (value >= 100) return 100
  return Math.round(value * 10) / 10
}

function formatSpeedValue(value: unknown) {
  const numericValue = Number(value)
  if (!Number.isFinite(numericValue) || numericValue <= 0) return '0 MB/s'
  if (numericValue < 10) return `${numericValue.toFixed(2)} MB/s`
  if (numericValue < 100) return `${numericValue.toFixed(1)} MB/s`
  return `${Math.round(numericValue)} MB/s`
}

function formatPercentValue(value: unknown) {
  const numericValue = Number(value)
  if (!Number.isFinite(numericValue) || numericValue <= 0) return '0.0%'
  return `${clampIndex(numericValue).toFixed(1)}%`
}

function formatDownloadCount(value: unknown) {
  const numericValue = Number(value)
  if (!Number.isFinite(numericValue) || numericValue <= 0) return '0 次'
  return `${Math.round(numericValue).toLocaleString()} 次`
}

const currentSpeedLabel = computed(() => props.currentSpeed || formatSpeedValue(props.values.at(-1)))

const speedChartValues = computed(() => {
  return props.values.map((value) => {
    if (!Number.isFinite(value) || value <= 0) return 0
    return Math.round(value * 100) / 100
  })
})

const threadActiveChartValues = computed(() => props.labels.map((_, index) => clampIndex(props.threadActiveValues[index] ?? 0)))
const threadActiveLabel = computed(() => formatPercentValue(currentThreadActive.value))

const urlDistributionRows = computed(() => {
  return props.urlStats.map((item, index) => {
    const url = item.url || `链接 ${index + 1}`
    return {
      key: `${url}-${index}`,
      url,
      countLabel: formatDownloadCount(item.count),
      percentageLabel: formatPercentValue(item.percentage),
    }
  })
})
</script>

<template>
  <Card class="min-h-[520px]">
    <CardHeader class="border-b py-3">
      <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <CardTitle>实时聚合图表</CardTitle>
        <div class="flex flex-wrap items-center gap-2">
          <Tooltip>
            <TooltipTrigger as-child>
              <Button
                size="icon-sm"
                variant="ghost"
                :aria-label="refreshing ? '刷新中' : '刷新趋势'"
                :disabled="refreshing"
                @click="$emit('refresh')"
              >
                <Spinner v-if="refreshing" data-icon="inline-start" />
                <RefreshCwIcon v-else data-icon="inline-start" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>{{ refreshing ? '刷新中' : '刷新趋势' }}</TooltipContent>
          </Tooltip>
        </div>
      </div>
    </CardHeader>
    <CardContent class="grid gap-5 p-4">
      <div class="grid min-w-0 gap-5 xl:grid-cols-[minmax(0,1fr)_280px]">
        <div class="grid min-w-0 gap-5">
          <section class="grid min-w-0 gap-2">
            <div class="flex flex-wrap items-baseline justify-between gap-2 text-sm">
              <span class="font-medium">当前速度</span>
              <span class="font-mono font-medium tabular-nums">{{ currentSpeedLabel }}</span>
            </div>
            <DashboardChart
              chart-id="aggregate-speed-chart"
              type="line"
              label="当前速度"
              :labels="labels"
              :value-formatter="formatSpeedValue"
              :values="speedChartValues"
              :height="210"
            />
          </section>

          <section class="grid min-w-0 gap-2">
            <div class="flex flex-wrap items-baseline justify-between gap-2 text-sm">
              <span class="font-medium">线程活跃</span>
              <span class="font-mono font-medium tabular-nums">{{ threadActiveLabel }}</span>
            </div>
            <DashboardChart
              chart-id="aggregate-thread-chart"
              type="line"
              label="线程活跃"
              :colors="['--chart-2']"
              :labels="labels"
              :value-formatter="formatPercentValue"
              :values="threadActiveChartValues"
              :height="150"
              :y-max="100"
            />
          </section>
        </div>

        <section class="grid min-w-0 content-start justify-items-center gap-3">
          <div class="w-full text-sm font-medium">
            URL 分布
          </div>
          <div class="w-full max-w-[240px]">
            <DashboardChart
              chart-id="aggregate-url-pie-chart"
              type="doughnut"
              label="URL"
              sub-label="次数 / 占比"
              :labels="urlChartLabels"
              :value-formatter="formatDownloadCount"
              :values="urlChartValues"
              :height="240"
            />
          </div>
          <div v-if="urlDistributionRows.length" class="grid max-h-40 w-full gap-2 overflow-auto pr-1 text-xs">
            <div
              v-for="item in urlDistributionRows"
              :key="item.key"
              class="grid gap-1 rounded-md border bg-card px-2 py-1.5"
            >
              <span class="truncate text-muted-foreground">{{ item.url }}</span>
              <div class="flex items-center justify-between gap-2">
                <span class="font-mono font-medium tabular-nums">{{ item.countLabel }}</span>
                <Badge variant="secondary">{{ item.percentageLabel }}</Badge>
              </div>
            </div>
          </div>
        </section>
      </div>
    </CardContent>
  </Card>
</template>
