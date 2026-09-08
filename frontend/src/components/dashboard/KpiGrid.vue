<script setup lang="ts">
import { ArrowDownToLineIcon, FilesIcon, GaugeIcon, LayersIcon } from '@lucide/vue'
import { computed } from 'vue'
import AnimatedMetric from '@/components/dashboard/AnimatedMetric.vue'
import { Card, CardAction, CardContent, CardDescription, CardHeader } from '@/components/ui/card'

const props = defineProps<{ activeConfigCount: number; downloadCount: number; speed: string; totalBytes: string }>()
const metrics = computed(() => [
  { id: 'speed-text', label: '实时速率', value: props.speed || '0 B/s', icon: GaugeIcon, note: '所有运行任务的聚合速率' },
  { id: 'total-bytes', label: '累计消耗', value: props.totalBytes || '0 B', icon: LayersIcon, note: '当前运行累计下载流量' },
  { id: 'download-count', label: '下载次数', value: props.downloadCount || 0, icon: ArrowDownToLineIcon, note: '当前运行累计下载次数' },
  { id: 'current-config', label: '活动配置', value: props.activeConfigCount || 0, icon: FilesIcon, note: '运行中与调度中的配置' },
])
</script>

<template>
  <div class="grid grid-cols-1 gap-4 min-[420px]:grid-cols-2 xl:grid-cols-4">
    <Card v-for="metric in metrics" :key="metric.id" class="min-w-0">
      <CardHeader>
        <CardDescription>{{ metric.label }}</CardDescription>
        <CardAction><div class="flex size-9 items-center justify-center rounded-lg bg-secondary text-primary"><component :is="metric.icon" class="size-[18px]" /></div></CardAction>
      </CardHeader>
      <CardContent>
        <div v-if="metric.id === 'speed-text'" :id="metric.id" class="truncate text-3xl font-semibold tracking-tight tabular-nums">{{ metric.value }}</div>
        <AnimatedMetric v-else :id="metric.id" class="block truncate text-3xl font-semibold tracking-tight tabular-nums" :value="metric.value" :decimals="typeof metric.value === 'number' ? 0 : undefined" />
        <p class="mt-3 text-xs text-muted-foreground">{{ metric.note }}</p>
      </CardContent>
    </Card>
  </div>
</template>
