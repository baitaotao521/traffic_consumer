<script setup lang="ts">
import {
  BoltIcon,
  CalendarClockIcon,
  ChevronsUpDownIcon,
  PlayIcon,
  PlusIcon,
  Settings2Icon,
  SquareIcon,
  Trash2Icon,
} from '@lucide/vue'
import { computed, ref } from 'vue'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { cn } from '@/lib/utils'
import type { BadgeVariant } from '@/types'

type ConfigOption = {
  label: string
  value: string
}

const props = defineProps<{
  configs: string[]
  selectedConfigName: string
  statusLabel: string
  statusVariant: BadgeVariant
  isStopped: boolean
  isRunning: boolean
  hasScheduler: boolean
}>()

const emit = defineEmits<{
  createConfig: []
  deleteConfig: []
  openEditor: []
  start: []
  startTemp: []
  stop: []
  stopScheduler: []
  'update:selectedConfigName': [value: string]
}>()

const open = ref(false)
const configOptions = computed<ConfigOption[]>(() => props.configs.map((name) => ({ label: name, value: name })))
const selectedConfigLabel = computed(() => {
  return configOptions.value.find((option) => option.value === props.selectedConfigName)?.label
    ?? props.selectedConfigName
})

function handleConfigSelect(value: string) {
  emit('update:selectedConfigName', value)
  open.value = false
}
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
        <Popover v-model:open="open">
          <PopoverTrigger as-child>
            <Button
              id="config-select"
              variant="outline"
              role="combobox"
              :aria-expanded="open"
              :disabled="!configOptions.length"
              class="w-[220px] justify-between font-normal"
            >
              <span :class="cn('truncate', !selectedConfigLabel && 'text-muted-foreground')">
                {{ selectedConfigLabel || '选择配置' }}
              </span>
              <ChevronsUpDownIcon data-icon="inline-end" class="opacity-50" />
            </Button>
          </PopoverTrigger>
          <PopoverContent align="start" class="w-[220px] p-0">
            <Command :model-value="selectedConfigName">
              <CommandInput placeholder="搜索配置" />
              <CommandList>
                <CommandEmpty>未找到配置</CommandEmpty>
                <CommandGroup>
                  <CommandItem
                    v-for="option in configOptions"
                    :key="option.value"
                    :value="option.value"
                    @select="handleConfigSelect(option.value)"
                  >
                    {{ option.label }}
                  </CommandItem>
                </CommandGroup>
              </CommandList>
            </Command>
          </PopoverContent>
        </Popover>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <Button id="start-btn" :disabled="!isStopped" @click="emit('start')">
          <PlayIcon />
          开始
        </Button>
        <Button
          id="temp-test-btn"
          variant="secondary"
          :disabled="isRunning"
          @click="emit('startTemp')"
        >
          <BoltIcon />
          临时测试
        </Button>
        <Button
          id="stop-btn"
          variant="destructive"
          :disabled="isStopped"
          @click="emit('stop')"
        >
          <SquareIcon />
          停止
        </Button>
        <Button
          id="stop-scheduler-btn"
          variant="outline"
          :disabled="!hasScheduler"
          @click="emit('stopScheduler')"
        >
          <CalendarClockIcon />
          停止调度
        </Button>
        <Button id="create-config-btn" variant="outline" @click="emit('createConfig')">
          <PlusIcon />
          新建配置
        </Button>
        <Button id="edit-config-btn" variant="ghost" @click="emit('openEditor')">
          <Settings2Icon />
          编辑配置
        </Button>
        <Button id="delete-config-btn" variant="destructive" @click="emit('deleteConfig')">
          <Trash2Icon />
          删除
        </Button>
      </div>
    </div>
  </header>
</template>
