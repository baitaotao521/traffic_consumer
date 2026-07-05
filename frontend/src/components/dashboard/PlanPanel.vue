<script setup lang="ts">
import { CalendarDaysIcon, ListChecksIcon, PencilIcon, PlayIcon, SquareIcon, Trash2Icon } from '@lucide/vue'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty'
import { HoverCard, HoverCardContent, HoverCardTrigger } from '@/components/ui/hover-card'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Table, TableBody, TableCell, TableEmpty, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import type { BadgeVariant, PlanSummary } from '@/types'

defineProps<{
  formatDateTime: (value?: string) => string
  formatPlanSchedule: (plan: PlanSummary) => string
  plans: PlanSummary[]
  planState: (plan: PlanSummary) => { label: string; variant: BadgeVariant }
}>()

defineEmits<{
  deletePlan: [plan: PlanSummary]
  editPlan: [plan: PlanSummary]
  openPlanDetail: [plan: PlanSummary]
  startPlan: [plan: PlanSummary]
  stopPlan: [plan: PlanSummary]
}>()
</script>

<template>
  <Card class="min-h-[420px]">
    <CardHeader class="border-b py-3">
      <div class="flex items-center justify-between gap-3">
        <CardTitle>配置列表</CardTitle>
        <Badge variant="secondary">{{ plans.length }}</Badge>
      </div>
    </CardHeader>
    <CardContent class="p-0">
      <div class="md:hidden">
        <div v-if="!plans.length" class="p-6">
          <Empty class="p-0">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <CalendarDaysIcon />
              </EmptyMedia>
              <EmptyTitle>暂无配置</EmptyTitle>
            </EmptyHeader>
          </Empty>
        </div>
        <div v-else class="divide-y">
          <div v-for="plan in plans" :key="`mobile-${plan.name}`" class="grid gap-3 px-4 py-3">
            <div class="flex min-w-0 items-start justify-between gap-3">
              <div class="min-w-0">
                <div class="truncate font-medium">{{ plan.name }}</div>
                <div class="mt-0.5 truncate text-xs text-muted-foreground">{{ formatPlanSchedule(plan) }}</div>
              </div>
              <Badge :variant="planState(plan).variant">{{ planState(plan).label }}</Badge>
            </div>

            <div class="grid grid-cols-3 gap-2 text-xs">
              <div>
                <div class="text-muted-foreground">速度</div>
                <div class="font-mono font-medium tabular-nums">{{ plan.speed || '0 B/s' }}</div>
              </div>
              <div>
                <div class="text-muted-foreground">流量</div>
                <div class="font-mono font-medium tabular-nums">{{ plan.total_bytes || '0 B' }}</div>
              </div>
              <div>
                <div class="text-muted-foreground">下载</div>
                <div class="font-mono font-medium tabular-nums">{{ plan.download_count || 0 }}</div>
              </div>
              <div>
                <div class="text-muted-foreground">线程</div>
                <div class="font-mono font-medium tabular-nums">{{ plan.threads || 0 }}</div>
              </div>
              <div>
                <div class="text-muted-foreground">链接</div>
                <div class="font-mono font-medium tabular-nums">{{ plan.url_count || 0 }}</div>
              </div>
              <div>
                <div class="text-muted-foreground">下次执行</div>
                <div class="truncate font-medium">{{ formatDateTime(plan.next_run_time) }}</div>
              </div>
            </div>

            <div class="text-xs">
              <span class="text-muted-foreground">上次运行</span>
              <span v-if="plan.last_run" class="ml-1">
                {{ plan.last_run.result || '成功' }} · {{ plan.last_run.bytes_consumed || '0 B' }}
              </span>
              <span v-else class="ml-1 text-muted-foreground">无</span>
            </div>

            <div class="flex flex-wrap gap-2">
              <Button class="flex-1 sm:flex-none" size="xs" variant="outline" @click="$emit('openPlanDetail', plan)">
                <ListChecksIcon data-icon="inline-start" />
                详情
              </Button>
              <Button class="flex-1 sm:flex-none" size="xs" variant="outline" @click="$emit('editPlan', plan)">
                <PencilIcon data-icon="inline-start" />
                编辑
              </Button>
              <Button
                class="flex-1 sm:flex-none"
                size="xs"
                :disabled="plan.running || plan.scheduler_running"
                @click="$emit('startPlan', plan)"
              >
                <PlayIcon data-icon="inline-start" />
                启动
              </Button>
              <Button
                class="flex-1 sm:flex-none"
                size="xs"
                variant="outline"
                :disabled="!plan.running && !plan.scheduler_running"
                @click="$emit('stopPlan', plan)"
              >
                <SquareIcon data-icon="inline-start" />
                停止
              </Button>
              <Button class="flex-1 sm:flex-none" size="xs" variant="destructive" @click="$emit('deletePlan', plan)">
                <Trash2Icon data-icon="inline-start" />
                删除
              </Button>
            </div>
          </div>
        </div>
      </div>

      <ScrollArea class="hidden h-[378px] md:block">
        <Table class="min-w-[1220px]">
          <TableHeader class="bg-muted">
            <TableRow>
              <TableHead class="min-w-[180px]">配置</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>速度</TableHead>
              <TableHead>流量</TableHead>
              <TableHead class="text-right">下载</TableHead>
              <TableHead class="text-right">线程</TableHead>
              <TableHead class="text-right">链接</TableHead>
              <TableHead class="min-w-[180px]">上次运行结果</TableHead>
              <TableHead class="min-w-[160px]">调度</TableHead>
              <TableHead class="min-w-[260px] text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody id="runtime-plan-list">
            <TableEmpty v-if="!plans.length" :colspan="10" class="py-0">
              <Empty class="p-0">
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <CalendarDaysIcon />
                  </EmptyMedia>
                  <EmptyTitle>暂无配置</EmptyTitle>
                </EmptyHeader>
              </Empty>
            </TableEmpty>
            <TableRow v-for="plan in plans" :key="plan.name">
              <TableCell class="font-medium">
                <HoverCard :open-delay="120">
                  <HoverCardTrigger as-child>
                    <div class="max-w-[220px] truncate">{{ plan.name }}</div>
                  </HoverCardTrigger>
                  <HoverCardContent class="w-64 break-all">
                    {{ plan.name }}
                  </HoverCardContent>
                </HoverCard>
              </TableCell>
              <TableCell>
                <Badge :variant="planState(plan).variant">{{ planState(plan).label }}</Badge>
              </TableCell>
              <TableCell>{{ plan.speed || '0 B/s' }}</TableCell>
              <TableCell>{{ plan.total_bytes || '0 B' }}</TableCell>
              <TableCell class="text-right">{{ plan.download_count || 0 }}</TableCell>
              <TableCell class="text-right">{{ plan.threads || 0 }}</TableCell>
              <TableCell class="text-right">{{ plan.url_count || 0 }}</TableCell>
              <TableCell>
                <div v-if="plan.last_run" class="max-w-[180px]">
                  <div class="truncate">
                    {{ plan.last_run.result || '成功' }} · {{ plan.last_run.bytes_consumed || '0 B' }}
                  </div>
                  <div class="text-xs text-muted-foreground">
                    {{ formatDateTime(plan.last_run.timestamp) }}
                  </div>
                </div>
                <span v-else class="text-muted-foreground">无</span>
              </TableCell>
              <TableCell>
                <div class="max-w-[180px] truncate">{{ formatPlanSchedule(plan) }}</div>
                <div v-if="plan.next_run_time" class="text-xs text-muted-foreground">
                  {{ formatDateTime(plan.next_run_time) }}
                </div>
              </TableCell>
              <TableCell>
                <div class="flex justify-end gap-2">
                  <Button size="xs" variant="outline" @click="$emit('openPlanDetail', plan)">
                    <ListChecksIcon data-icon="inline-start" />
                    详情
                  </Button>
                  <Button size="xs" variant="outline" @click="$emit('editPlan', plan)">
                    <PencilIcon data-icon="inline-start" />
                    编辑
                  </Button>
                  <Button
                    size="xs"
                    :disabled="plan.running || plan.scheduler_running"
                    @click="$emit('startPlan', plan)"
                  >
                    <PlayIcon data-icon="inline-start" />
                    启动
                  </Button>
                  <Button
                    size="xs"
                    variant="outline"
                    :disabled="!plan.running && !plan.scheduler_running"
                    @click="$emit('stopPlan', plan)"
                  >
                    <SquareIcon data-icon="inline-start" />
                    停止
                  </Button>
                  <Button size="xs" variant="destructive" @click="$emit('deletePlan', plan)">
                    <Trash2Icon data-icon="inline-start" />
                    删除
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </ScrollArea>
    </CardContent>
  </Card>
</template>
