<script setup lang="ts">
import { RefreshCwIcon } from '@lucide/vue'

import DashboardChart from '@/components/DashboardChart.vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

defineProps<{
  labels: string[]
  refreshing?: boolean
  values: number[]
}>()

defineEmits<{
  refresh: []
}>()
</script>

<template>
  <Card class="min-h-[360px]">
    <CardHeader class="border-b py-3">
      <div class="flex items-center justify-between">
        <CardTitle>实时速率趋势</CardTitle>
        <div class="flex items-center gap-2">
          <div class="flex items-center gap-2">
            <span class="size-2.5 rounded-full bg-primary" />
            <span class="text-xs text-muted-foreground">Download</span>
          </div>
          <Tooltip>
            <TooltipTrigger as-child>
              <Button
                size="icon-sm"
                variant="ghost"
                :aria-label="refreshing ? '刷新中' : '刷新趋势'"
                :disabled="refreshing"
                @click="$emit('refresh')"
              >
                <Spinner v-if="refreshing" />
                <RefreshCwIcon v-else />
              </Button>
            </TooltipTrigger>
            <TooltipContent>{{ refreshing ? '刷新中' : '刷新趋势' }}</TooltipContent>
          </Tooltip>
        </div>
      </div>
    </CardHeader>
    <CardContent class="py-4">
      <DashboardChart
        chart-id="speed-chart"
        type="line"
        :labels="labels"
        :values="values"
        label="MB/s"
        :height="220"
      />
    </CardContent>
  </Card>
</template>
