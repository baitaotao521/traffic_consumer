<script setup lang="ts">
import {
  CalendarClockIcon,
  PlusIcon,
  SquareIcon,
} from '@lucide/vue'
import { computed } from 'vue'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import type { BadgeVariant } from '@/types'

const props = defineProps<{
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
  <header class="sticky top-0 z-30 border-b bg-background/95 backdrop-blur">
    <div class="mx-auto flex max-w-[1600px] flex-col gap-3 px-6 py-3 lg:flex-row lg:items-center lg:justify-between">
      <div class="flex min-w-0 flex-wrap items-center gap-3">
        <h1 class="text-base font-semibold">Traffic Consumer</h1>
        <Badge :variant="statusVariant" class="gap-1.5">
          <span :class="cn('size-1.5 rounded-full', statusDotClass)" />
          {{ statusLabel }}
        </Badge>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <Button id="create-config-btn" variant="outline" @click="emit('createConfig')">
          <PlusIcon data-icon="inline-start" />
          新建配置
        </Button>
        <Button
          id="stop-btn"
          variant="destructive"
          :disabled="isStopped"
          @click="emit('stop')"
        >
          <SquareIcon data-icon="inline-start" />
          停止全部
        </Button>
        <Button
          id="stop-scheduler-btn"
          variant="outline"
          :disabled="!hasScheduler"
          @click="emit('stopScheduler')"
        >
          <CalendarClockIcon data-icon="inline-start" />
          停止调度
        </Button>
      </div>
    </div>
  </header>
</template>
