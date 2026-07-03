<script setup lang="ts">
import {
  BoltIcon,
  CalendarClockIcon,
  PlayIcon,
  PlusIcon,
  Settings2Icon,
  SquareIcon,
  Trash2Icon,
} from '@lucide/vue'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { BadgeVariant } from '@/types'

defineProps<{
  configs: string[]
  selectedConfigName: string
  statusLabel: string
  statusVariant: BadgeVariant
  isStopped: boolean
  isRunning: boolean
  hasScheduler: boolean
}>()

defineEmits<{
  createConfig: []
  deleteConfig: []
  openEditor: []
  start: []
  startTemp: []
  stop: []
  stopScheduler: []
  'update:selectedConfigName': [value: string]
}>()
</script>

<template>
  <header class="sticky top-0 z-30 border-b bg-background/95 backdrop-blur">
    <div class="mx-auto flex max-w-[1600px] flex-col gap-3 px-6 py-3 lg:flex-row lg:items-center lg:justify-between">
      <div class="flex min-w-0 flex-wrap items-center gap-3">
        <h1 class="text-base font-semibold">Traffic Consumer</h1>
        <Badge :variant="statusVariant" class="gap-1.5">
          <span class="size-1.5 rounded-full bg-destructive" />
          {{ statusLabel }}
        </Badge>
        <Select
          :model-value="selectedConfigName"
          @update:model-value="$emit('update:selectedConfigName', String($event))"
        >
          <SelectTrigger id="config-select" class="w-[180px]">
            <SelectValue placeholder="默认配置架构" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem v-for="name in configs" :key="name" :value="name">
              {{ name }}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <Button id="start-btn" :disabled="!isStopped" @click="$emit('start')">
          <PlayIcon />
          开始
        </Button>
        <Button
          id="temp-test-btn"
          variant="secondary"
          :disabled="isRunning"
          @click="$emit('startTemp')"
        >
          <BoltIcon />
          临时测试
        </Button>
        <Button
          id="stop-btn"
          variant="destructive"
          :disabled="isStopped"
          @click="$emit('stop')"
        >
          <SquareIcon />
          停止
        </Button>
        <Button
          id="stop-scheduler-btn"
          variant="outline"
          :disabled="!hasScheduler"
          @click="$emit('stopScheduler')"
        >
          <CalendarClockIcon />
          停止调度
        </Button>
        <Button id="create-config-btn" variant="outline" @click="$emit('createConfig')">
          <PlusIcon />
          新建配置
        </Button>
        <Button id="edit-config-btn" variant="ghost" @click="$emit('openEditor')">
          <Settings2Icon />
          编辑配置
        </Button>
        <Button id="delete-config-btn" variant="destructive" @click="$emit('deleteConfig')">
          <Trash2Icon />
          删除
        </Button>
      </div>
    </div>
  </header>
</template>
