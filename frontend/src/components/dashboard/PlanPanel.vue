<script setup lang="ts">
import { CalendarDaysIcon } from '@lucide/vue'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty'
import { HoverCard, HoverCardContent, HoverCardTrigger } from '@/components/ui/hover-card'
import type { BadgeVariant, PlanSummary } from '@/types'

defineProps<{
  formatDateTime: (value?: string) => string
  formatPlanSchedule: (plan: PlanSummary) => string
  jobDetails: string
  plans: PlanSummary[]
  planState: (plan: PlanSummary) => { label: string; variant: BadgeVariant }
}>()

defineEmits<{
  openPlanDetail: [plan: PlanSummary]
  stopPlan: [plan: PlanSummary]
}>()
</script>

<template>
  <Card class="min-h-[320px]">
    <CardHeader class="border-b py-3">
      <div class="flex items-center justify-between gap-3">
        <CardTitle>运行计划</CardTitle>
        <HoverCard v-if="jobDetails" :open-delay="120">
          <HoverCardTrigger as-child>
            <span class="truncate text-xs text-muted-foreground">{{ jobDetails }}</span>
          </HoverCardTrigger>
          <HoverCardContent class="w-80 break-all">
            {{ jobDetails }}
          </HoverCardContent>
        </HoverCard>
        <span v-else class="text-xs text-muted-foreground">无</span>
      </div>
    </CardHeader>
    <CardContent class="p-4">
      <div v-if="!plans.length" class="flex min-h-[236px] flex-col items-center justify-center text-center">
        <Empty class="p-0">
          <EmptyHeader>
            <EmptyMedia variant="icon" class="size-12 [&_svg]:size-5">
              <CalendarDaysIcon />
            </EmptyMedia>
            <EmptyTitle>暂无运行计划</EmptyTitle>
          </EmptyHeader>
        </Empty>
      </div>
      <div v-else id="runtime-plan-list" class="grid gap-3">
        <Alert v-for="plan in plans" :key="plan.name">
          <AlertTitle class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <HoverCard :open-delay="120">
                <HoverCardTrigger as-child>
                  <div class="truncate font-medium">{{ plan.name }}</div>
                </HoverCardTrigger>
                <HoverCardContent class="w-64 break-all">
                  {{ plan.name }}
                </HoverCardContent>
              </HoverCard>
              <div class="mt-1 text-xs font-normal text-muted-foreground">
                {{ formatPlanSchedule(plan) }} · {{ formatDateTime(plan.next_run_time) }}
              </div>
            </div>
            <Badge :variant="planState(plan).variant">{{ planState(plan).label }}</Badge>
          </AlertTitle>
          <AlertDescription class="mt-3">
            <div class="grid grid-cols-2 gap-2 text-xs">
              <div>流量 <span class="ml-1 font-semibold">{{ plan.total_bytes }}</span></div>
              <div>下载 <span class="ml-1 font-semibold">{{ plan.download_count }}</span></div>
              <div>线程 <span class="ml-1 font-semibold">{{ plan.threads }}</span></div>
              <div>链接 <span class="ml-1 font-semibold">{{ plan.url_count }}</span></div>
            </div>
            <div class="mt-3 flex justify-end gap-2">
              <Button size="xs" variant="outline" @click="$emit('openPlanDetail', plan)">
                详情
              </Button>
              <Button
                size="xs"
                variant="destructive"
                :disabled="!plan.running && !plan.scheduler_running"
                @click="$emit('stopPlan', plan)"
              >
                停止
              </Button>
            </div>
          </AlertDescription>
        </Alert>
      </div>
    </CardContent>
  </Card>
</template>
