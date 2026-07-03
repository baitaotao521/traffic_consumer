<script setup lang="ts">
import { RefreshCwIcon } from '@lucide/vue'

import DashboardChart from '@/components/DashboardChart.vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

defineProps<{
  labels: string[]
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
          <Button size="icon-sm" variant="ghost" @click="$emit('refresh')">
            <RefreshCwIcon />
          </Button>
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
