<script setup lang="ts">
import { FileTextIcon, HistoryIcon } from '@lucide/vue'
import { nextTick, ref, watch } from 'vue'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty'
import { HoverCard, HoverCardContent, HoverCardTrigger } from '@/components/ui/hover-card'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Switch } from '@/components/ui/switch'
import { Table, TableBody, TableCell, TableEmpty, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import type { LogEntry, PlanSummary } from '@/types'

const props = defineProps<{
  formatDateTime: (value?: string) => string
  logEnabled: boolean
  logLevelClass: (level: string) => string
  logLevelOptions: Array<{ value: string; label: string }>
  logSourceLabel: (source?: string) => string
  logSourceOptions: Array<{ value: string; label: string }>
  logSourceVariant: (source?: string) => 'default' | 'secondary' | 'destructive' | 'outline'
  logs: LogEntry[]
  plans: PlanSummary[]
  selectedLogLevels: string[]
  selectedLogSources: string[]
}>()

const logScrollArea = ref<{ scrollBottom: () => void } | null>(null)

defineEmits<{
  clearLogs: []
  'update:logEnabled': [value: boolean]
  'update:selectedLogLevels': [value: string[]]
  'update:selectedLogSources': [value: string[]]
}>()

watch(
  () => [props.logEnabled, props.logs[props.logs.length - 1]?.seq],
  async ([enabled, lastSeq], [previousEnabled, previousLastSeq]) => {
    if (!enabled || !lastSeq) return
    if (enabled === previousEnabled && lastSeq === previousLastSeq) return

    // 实时日志总是跟随最新一条，避免用户看到旧位置而误判“没有新日志”。
    await nextTick()
    logScrollArea.value?.scrollBottom()
  },
)
</script>

<template>
  <Card>
    <Tabs default-value="logs">
      <CardHeader class="border-b py-2">
        <TabsList>
          <TabsTrigger value="logs">实时日志</TabsTrigger>
          <TabsTrigger value="history">调度历史</TabsTrigger>
        </TabsList>
      </CardHeader>

      <CardContent class="p-0">
        <TabsContent value="logs" class="m-0">
          <div class="flex flex-col gap-3 px-4 py-3">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div class="flex items-center gap-2">
                <Switch
                  id="log-switch"
                  :model-value="logEnabled"
                  @update:model-value="$emit('update:logEnabled', Boolean($event))"
                />
                <Label for="log-switch">接收实时日志</Label>
              </div>
              <Button
                id="clear-log-btn"
                size="xs"
                variant="ghost"
                @click="$emit('clearLogs')"
              >
                清空
              </Button>
            </div>

            <div class="flex flex-col gap-2 border-t pt-3">
              <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
                <span class="text-xs text-muted-foreground">来源</span>
                <ToggleGroup
                  class="max-w-full flex-wrap"
                  :model-value="selectedLogSources"
                  size="sm"
                  :spacing="2"
                  type="multiple"
                  variant="outline"
                  @update:model-value="$emit('update:selectedLogSources', Array.isArray($event) ? $event : [])"
                >
                  <ToggleGroupItem
                    v-for="option in logSourceOptions"
                    :key="`source-${option.value}`"
                    :value="option.value"
                  >
                    {{ option.label }}
                  </ToggleGroupItem>
                </ToggleGroup>
              </div>
              <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
                <span class="text-xs text-muted-foreground">级别</span>
                <ToggleGroup
                  class="max-w-full flex-wrap"
                  :model-value="selectedLogLevels"
                  size="sm"
                  :spacing="2"
                  type="multiple"
                  variant="outline"
                  @update:model-value="$emit('update:selectedLogLevels', Array.isArray($event) ? $event : [])"
                >
                  <ToggleGroupItem
                    v-for="option in logLevelOptions"
                    :key="`level-${option.value}`"
                    :value="option.value"
                  >
                    {{ option.label }}
                  </ToggleGroupItem>
                </ToggleGroup>
              </div>
            </div>
          </div>

          <ScrollArea
            id="log-container"
            ref="logScrollArea"
            class="h-[200px] border-t bg-muted px-4 py-3 font-mono text-xs"
          >
            <div v-if="logs.length">
              <div v-for="entry in logs" :key="entry.seq" class="mb-2 flex items-center gap-2">
                <span class="shrink-0 text-muted-foreground">[{{ new Date(entry.timestamp || Date.now()).toLocaleTimeString() }}]</span>
                <div class="flex min-w-0 flex-1 items-center gap-2 overflow-hidden">
                  <Badge :variant="logSourceVariant(entry.source)">{{ logSourceLabel(entry.source) }}</Badge>
                  <span class="shrink-0 text-[10px] uppercase tracking-[0.18em] text-muted-foreground">{{ entry.level }}</span>
                  <HoverCard :open-delay="120">
                    <HoverCardTrigger as-child>
                      <span :class="['min-w-0 flex-1 truncate', logLevelClass(entry.level)]">{{ entry.message }}</span>
                    </HoverCardTrigger>
                    <HoverCardContent class="w-[28rem] break-all font-sans text-sm">
                      {{ entry.message }}
                    </HoverCardContent>
                  </HoverCard>
                </div>
              </div>
            </div>
            <Empty v-else class="min-h-[176px] p-0">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <FileTextIcon />
                </EmptyMedia>
                <EmptyTitle>{{ logEnabled ? '暂无日志' : '日志已暂停' }}</EmptyTitle>
              </EmptyHeader>
            </Empty>
          </ScrollArea>
        </TabsContent>

        <TabsContent value="history" class="m-0">
          <div class="md:hidden">
            <div v-if="!plans.length" class="p-6">
              <Empty class="p-0">
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <HistoryIcon />
                  </EmptyMedia>
                  <EmptyTitle>暂无调度历史</EmptyTitle>
                </EmptyHeader>
              </Empty>
            </div>
            <div v-else class="divide-y">
              <div v-for="plan in plans" :key="`mobile-history-${plan.name}`" class="grid gap-2 px-4 py-3 text-sm">
                <div class="truncate font-medium">{{ plan.name }}</div>
                <div class="grid grid-cols-2 gap-2 text-xs">
                  <div>
                    <div class="text-muted-foreground">下次执行</div>
                    <div class="font-medium">{{ formatDateTime(plan.next_run_time) }}</div>
                  </div>
                  <div>
                    <div class="text-muted-foreground">下载数</div>
                    <div class="font-mono font-medium tabular-nums">{{ plan.download_count || 0 }}</div>
                  </div>
                  <div class="col-span-2">
                    <div class="text-muted-foreground">任务流量</div>
                    <div class="font-mono font-medium tabular-nums">{{ plan.total_bytes || '0 B' }}</div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="hidden md:block">
            <Table>
              <TableHeader class="bg-muted">
                <TableRow>
                  <TableHead>计划</TableHead>
                  <TableHead>下次执行</TableHead>
                  <TableHead>任务流量</TableHead>
                  <TableHead class="text-right">下载数</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableEmpty v-if="!plans.length" :colspan="4" class="py-0">
                  <Empty class="p-0">
                    <EmptyHeader>
                      <EmptyMedia variant="icon">
                        <HistoryIcon />
                      </EmptyMedia>
                      <EmptyTitle>暂无调度历史</EmptyTitle>
                    </EmptyHeader>
                  </Empty>
                </TableEmpty>
                <TableRow v-for="plan in plans" :key="`history-${plan.name}`">
                  <TableCell>{{ plan.name }}</TableCell>
                  <TableCell>{{ formatDateTime(plan.next_run_time) }}</TableCell>
                  <TableCell>{{ plan.total_bytes }}</TableCell>
                  <TableCell class="text-right">{{ plan.download_count }}</TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>
        </TabsContent>
      </CardContent>
    </Tabs>
  </Card>
</template>
