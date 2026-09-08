<script setup lang="ts">
import { computed, defineAsyncComponent } from 'vue'
import { ActivityIcon, RefreshCwIcon } from '@lucide/vue'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import type { URLUsageStat } from '@/types'

const DashboardChart = defineAsyncComponent(() => import('@/components/DashboardChart.vue'))

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
  <div class="grid min-w-0 gap-5 xl:grid-cols-[minmax(0,1.8fr)_minmax(300px,1fr)]">
    <Card class="min-w-0 xl:col-span-2">
      <CardHeader>
        <CardTitle>流量趋势</CardTitle>
        <CardDescription>所有任务的实时下载速率</CardDescription>
        <CardAction>
          <div class="flex items-center gap-3">
            <Badge variant="secondary"><ActivityIcon data-icon="inline-start" />实时</Badge>
            <Tooltip><TooltipTrigger as-child><Button size="icon-sm" variant="ghost" :aria-label="refreshing ? '刷新中' : '刷新趋势'" :disabled="refreshing" @click="$emit('refresh')"><Spinner v-if="refreshing" data-icon="inline-start" /><RefreshCwIcon v-else data-icon="inline-start" /></Button></TooltipTrigger><TooltipContent>刷新趋势</TooltipContent></Tooltip>
          </div>
        </CardAction>
      </CardHeader>
      <CardContent>
        <div class="mb-5 flex items-baseline gap-3"><span class="text-2xl font-semibold tracking-tight tabular-nums">{{ currentSpeedLabel }}</span><span class="text-xs text-muted-foreground">当前速率</span></div>
        <Suspense><DashboardChart chart-id="aggregate-speed-chart" type="line" label="下载速率" :labels="labels" :value-formatter="formatSpeedValue" :values="speedChartValues" :height="240" /><template #fallback><Skeleton class="h-[240px] w-full" /></template></Suspense>
      </CardContent>
    </Card>
    <Card class="min-w-0">
      <CardHeader><CardTitle>线程活跃度</CardTitle><CardDescription>活跃线程占总线程的比例</CardDescription><CardAction><Badge variant="outline">{{ threadActiveLabel }}</Badge></CardAction></CardHeader>
      <CardContent><Suspense><DashboardChart chart-id="aggregate-thread-chart" type="line" label="线程活跃" :colors="['--chart-2']" :labels="labels" :value-formatter="formatPercentValue" :values="threadActiveChartValues" :height="190" :y-max="100" /><template #fallback><Skeleton class="h-[190px] w-full" /></template></Suspense></CardContent>
    </Card>
    <Card class="min-w-0">
      <CardHeader><CardTitle>下载来源</CardTitle><CardDescription>各 URL 的下载次数与占比</CardDescription></CardHeader>
      <CardContent>
        <div class="flex min-w-0 flex-col items-center gap-4 min-[500px]:flex-row">
          <div class="w-[170px] shrink-0"><Suspense><DashboardChart chart-id="aggregate-url-pie-chart" type="doughnut" label="URL" sub-label="次数 / 占比" :labels="urlChartLabels" :value-formatter="formatDownloadCount" :values="urlChartValues" :height="190" /><template #fallback><Skeleton class="h-[190px]" /></template></Suspense></div>
          <div class="flex max-h-[190px] w-full min-w-0 flex-col gap-4 overflow-auto">
            <p v-if="!urlDistributionRows.length" class="text-sm text-muted-foreground">暂无下载记录</p>
            <div v-for="(item, index) in urlDistributionRows" :key="item.key" class="flex min-w-0 items-start gap-2">
              <span class="mt-1 size-2 shrink-0 rounded-full" :style="{ background: `var(--chart-${index % 5 + 1})` }" />
              <div class="min-w-0 flex-1"><Tooltip><TooltipTrigger as-child><p class="truncate text-xs" tabindex="0">{{ item.url }}</p></TooltipTrigger><TooltipContent class="max-w-[min(28rem,85vw)] break-all">{{ item.url }}</TooltipContent></Tooltip><div class="mt-1 flex items-center justify-between gap-2 text-xs text-muted-foreground"><span>{{ item.countLabel }}</span><span class="tabular-nums">{{ item.percentageLabel }}</span></div></div>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  </div>
</template>
