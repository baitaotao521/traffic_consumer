<script setup lang="ts">
import { ActivityIcon, FileTextIcon, GaugeIcon, ListChecksIcon } from '@lucide/vue'

import { Card, CardAction, CardContent, CardDescription, CardHeader } from '@/components/ui/card'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

defineProps<{
  currentConfig: string
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
        <div id="speed-text" class="text-2xl font-bold">{{ speed || '0 B/s' }}</div>
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
        <div id="total-bytes" class="text-2xl font-bold">{{ totalBytes || '0 B' }}</div>
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
        <div id="download-count" class="text-2xl font-bold">{{ downloadCount || 0 }}</div>
      </CardContent>
    </Card>

    <Card size="sm" class="min-h-[116px]">
      <CardHeader>
        <CardDescription>当前配置</CardDescription>
        <CardAction>
          <FileTextIcon class="size-4 text-muted-foreground" />
        </CardAction>
      </CardHeader>
      <CardContent>
        <Tooltip>
          <TooltipTrigger as-child>
            <div id="current-config" class="truncate text-2xl font-bold">
              {{ currentConfig }}
            </div>
          </TooltipTrigger>
          <TooltipContent>{{ currentConfig }}</TooltipContent>
        </Tooltip>
      </CardContent>
    </Card>
  </div>
</template>
