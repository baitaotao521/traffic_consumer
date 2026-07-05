<script setup lang="ts">
import { ActivityIcon, FileTextIcon, GaugeIcon, ListChecksIcon } from '@lucide/vue'

import AnimatedMetric from '@/components/dashboard/AnimatedMetric.vue'
import { Card, CardAction, CardContent, CardDescription, CardHeader } from '@/components/ui/card'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

defineProps<{
  activeConfigCount: number
  downloadCount: number
  speed: string
  totalBytes: string
}>()
</script>

<template>
  <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
    <Card size="sm" class="min-h-[116px]">
      <CardHeader>
        <CardDescription>实时速率</CardDescription>
        <CardAction>
          <GaugeIcon class="size-4 text-muted-foreground" />
        </CardAction>
      </CardHeader>
      <CardContent>
        <div id="speed-text" class="truncate text-2xl font-bold">{{ speed || '0 B/s' }}</div>
      </CardContent>
    </Card>

    <Card size="sm" class="min-h-[116px]">
      <CardHeader>
        <CardDescription>总计流量</CardDescription>
        <CardAction>
          <ActivityIcon class="size-4 text-muted-foreground" />
        </CardAction>
      </CardHeader>
      <CardContent>
        <AnimatedMetric
          id="total-bytes"
          class="block truncate text-2xl font-bold tabular-nums"
          :value="totalBytes || '0 B'"
        />
      </CardContent>
    </Card>

    <Card size="sm" class="min-h-[116px]">
      <CardHeader>
        <CardDescription>下载次数</CardDescription>
        <CardAction>
          <ListChecksIcon class="size-4 text-muted-foreground" />
        </CardAction>
      </CardHeader>
      <CardContent>
        <AnimatedMetric
          id="download-count"
          class="block truncate text-2xl font-bold tabular-nums"
          :decimals="0"
          :value="downloadCount || 0"
        />
      </CardContent>
    </Card>

    <Card size="sm" class="min-h-[116px]">
      <CardHeader>
        <CardDescription>活动配置</CardDescription>
        <CardAction>
          <FileTextIcon class="size-4 text-muted-foreground" />
        </CardAction>
      </CardHeader>
      <CardContent>
        <Tooltip>
          <TooltipTrigger as-child>
            <AnimatedMetric
              id="current-config"
              class="block truncate text-2xl font-bold tabular-nums"
              :decimals="0"
              :value="activeConfigCount || 0"
            />
          </TooltipTrigger>
          <TooltipContent>运行与调度中的配置</TooltipContent>
        </Tooltip>
      </CardContent>
    </Card>
  </div>
</template>
