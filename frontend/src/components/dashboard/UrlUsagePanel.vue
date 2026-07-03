<script setup lang="ts">
import { PieChartIcon } from '@lucide/vue'

import DashboardChart from '@/components/DashboardChart.vue'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty'
import { Progress } from '@/components/ui/progress'
import type { URLUsageStat } from '@/types'

defineProps<{
  labels: string[]
  stats: URLUsageStat[]
  values: number[]
}>()
</script>

<template>
  <Card class="min-h-[360px]">
    <CardHeader class="border-b py-3">
      <CardTitle>URL 负载分布</CardTitle>
    </CardHeader>
    <CardContent class="grid min-h-[260px] gap-4 p-4 xl:grid-cols-[160px_minmax(0,1fr)]">
      <div class="flex items-center justify-center">
        <DashboardChart
          chart-id="url-pie-chart"
          type="doughnut"
          :labels="labels"
          :values="values"
          :height="150"
        />
      </div>
      <div id="url-usage-list" class="grid content-start gap-3 overflow-auto pr-1 text-sm">
        <Empty v-if="!stats.length" class="p-0 text-left">
          <EmptyHeader class="items-start">
            <EmptyMedia variant="icon">
              <PieChartIcon />
            </EmptyMedia>
            <EmptyTitle>暂无下载数据</EmptyTitle>
          </EmptyHeader>
        </Empty>
        <div v-for="item in stats" :key="item.url" class="grid gap-1.5">
          <div class="flex items-center justify-between gap-3">
            <span class="truncate text-muted-foreground" :title="item.url">{{ item.url }}</span>
            <span class="shrink-0 text-xs font-semibold">{{ item.percentage.toFixed(1) }}%</span>
          </div>
          <Progress :model-value="Math.max(0, Math.min(100, item.percentage))" />
        </div>
      </div>
    </CardContent>
  </Card>
</template>
