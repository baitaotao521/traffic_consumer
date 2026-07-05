<script setup lang="ts">
import {
  CalendarClockIcon,
  ChevronsUpDownIcon,
  PlusIcon,
  SquareIcon,
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
  hasScheduler: boolean
}>()

const emit = defineEmits<{
  createConfig: []
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
const statusDotClass = computed(() => {
  if (props.statusVariant === 'default') return 'bg-status-running'
  if (props.statusVariant === 'secondary') return 'bg-status-scheduled'
  if (props.statusVariant === 'destructive') return 'bg-destructive'
  return 'bg-status-stopped'
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
          <span :class="cn('size-1.5 rounded-full', statusDotClass)" />
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
