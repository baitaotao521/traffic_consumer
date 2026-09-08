<script setup lang="ts">
import {
  CalendarClockIcon,
  PlusIcon,
  SquareIcon,
} from '@lucide/vue'
import { computed } from 'vue'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { SidebarTrigger } from '@/components/ui/sidebar'
import { Separator } from '@/components/ui/separator'
import { cn } from '@/lib/utils'
import type { BadgeVariant } from '@/types'

const props = defineProps<{
  connectionError: boolean
  pageLabel: string
  loading: boolean
  statusLabel: string
  statusVariant: BadgeVariant
  isStopped: boolean
  hasScheduler: boolean
}>()

const emit = defineEmits<{
  createConfig: []
  stop: []
  stopScheduler: []
}>()

const statusDotClass = computed(() => {
  if (props.statusVariant === 'default') return 'bg-status-running'
  if (props.statusVariant === 'secondary') return 'bg-status-scheduled'
  if (props.statusVariant === 'destructive') return 'bg-destructive'
  return 'bg-status-stopped'
})
</script>

<template>
  <header class="sticky top-0 z-30 border-b bg-card/95 backdrop-blur">
    <div class="flex min-h-16 flex-wrap items-center justify-between gap-3 px-4 py-3 sm:px-7 lg:px-9">
      <div class="flex min-w-0 flex-wrap items-center gap-3">
        <SidebarTrigger aria-label="切换导航" /><Separator orientation="vertical" class="h-5" /><span class="text-sm font-medium">{{ pageLabel }}</span>
        <Badge v-if="!loading && !connectionError" :variant="statusVariant" class="gap-1.5">
          <span :class="cn('size-1.5 rounded-full', statusDotClass)" />
          {{ statusLabel }}
        </Badge>
        <Badge v-else-if="connectionError" variant="destructive">连接中断</Badge>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <Button id="create-config-btn" :disabled="loading" @click="emit('createConfig')">
          <PlusIcon data-icon="inline-start" />
          新建配置
        </Button>
        <Button
          id="stop-btn"
          variant="outline"
          :disabled="loading || isStopped"
          @click="emit('stop')"
        >
          <SquareIcon data-icon="inline-start" />
          停止全部
        </Button>
        <Button
          id="stop-scheduler-btn"
          variant="outline"
          :disabled="loading || !hasScheduler"
          @click="emit('stopScheduler')"
        >
          <CalendarClockIcon data-icon="inline-start" />
          停止调度
        </Button>
      </div>
    </div>
  </header>
</template>
