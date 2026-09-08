<script setup lang="ts">
import { ArrowUpRightIcon, CalendarDaysIcon, ListChecksIcon, PencilIcon, PlayIcon, PlusIcon, SearchIcon, SquareIcon, Trash2Icon, XIcon } from '@lucide/vue'
import { computed, ref } from 'vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardAction, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty'
import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupInput } from '@/components/ui/input-group'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import type { BadgeVariant, PlanSummary } from '@/types'

const props = defineProps<{
  compact?: boolean
  formatDateTime: (value?: string) => string
  formatPlanSchedule: (plan: PlanSummary) => string
  plans: PlanSummary[]
  planState: (plan: PlanSummary) => { label: string; variant: BadgeVariant }
}>()
const emit = defineEmits<{
  createConfig: []
  deletePlan: [plan: PlanSummary]
  editPlan: [plan: PlanSummary]
  openPlanDetail: [plan: PlanSummary]
  startPlan: [plan: PlanSummary]
  stopPlan: [plan: PlanSummary]
}>()
const search = ref('')
const stateFilter = ref('all')
const filters = [{ value: 'all', label: '全部' }, { value: 'running', label: '运行中' }, { value: 'scheduled', label: '调度中' }, { value: 'idle', label: '未启动' }]
const filteredPlans = computed(() => props.plans.filter((plan) => {
  if (props.compact) return true
  const matchesName = plan.name.toLocaleLowerCase().includes(search.value.trim().toLocaleLowerCase())
  const matchesState = stateFilter.value === 'all' || (stateFilter.value === 'running' && plan.running) || (stateFilter.value === 'scheduled' && plan.scheduler_running) || (stateFilter.value === 'idle' && !plan.running && !plan.scheduler_running)
  return matchesName && matchesState
}))
const visiblePlans = computed(() => props.compact ? filteredPlans.value.slice(0, 4) : filteredPlans.value)
function resetFilters() { search.value = ''; stateFilter.value = 'all' }
const actions = [
  { event: 'openPlanDetail', label: '查看详情', icon: ListChecksIcon },
  { event: 'editPlan', label: '编辑配置', icon: PencilIcon },
  { event: 'deletePlan', label: '删除配置', icon: Trash2Icon },
] as const
function runRowAction(action: typeof actions[number]['event'], plan: PlanSummary) {
  if (action === 'openPlanDetail') emit('openPlanDetail', plan)
  else if (action === 'editPlan') emit('editPlan', plan)
  else emit('deletePlan', plan)
}
</script>

<template>
  <Card class="min-w-0">
    <CardHeader>
      <CardTitle>{{ compact ? '配置概览' : '配置列表' }}</CardTitle>
      <CardDescription>共 {{ plans.length }} 个配置{{ compact ? '' : ` · 当前显示 ${filteredPlans.length} 个` }}</CardDescription>
      <CardAction><Button v-if="compact" variant="ghost" size="sm" as-child><a href="#plans">查看全部<ArrowUpRightIcon data-icon="inline-end" /></a></Button><Badge v-else variant="secondary">{{ plans.length }}</Badge></CardAction>
    </CardHeader>
    <CardContent class="flex min-w-0 flex-col gap-5">
      <div v-if="!compact" class="flex flex-wrap items-center justify-between gap-3">
        <InputGroup class="w-full sm:w-72">
          <InputGroupAddon><SearchIcon /></InputGroupAddon>
          <InputGroupInput v-model="search" placeholder="搜索配置名称…" aria-label="搜索配置名称" />
          <InputGroupAddon v-if="search" align="inline-end"><InputGroupButton size="icon-xs" aria-label="清除搜索" @click="search = ''"><XIcon /></InputGroupButton></InputGroupAddon>
        </InputGroup>
        <ToggleGroup :model-value="stateFilter" type="single" variant="outline" size="sm" aria-label="筛选配置状态" @update:model-value="stateFilter = String($event || 'all')">
          <ToggleGroupItem v-for="filter in filters" :key="filter.value" :value="filter.value">{{ filter.label }}</ToggleGroupItem>
        </ToggleGroup>
      </div>
      <Empty v-if="!visiblePlans.length" class="min-h-[260px]">
        <EmptyHeader><EmptyMedia variant="icon"><CalendarDaysIcon /></EmptyMedia><EmptyTitle>{{ plans.length ? '没有匹配的配置' : '创建你的第一个配置' }}</EmptyTitle><EmptyDescription>{{ plans.length ? '试试其他名称或运行状态。' : '添加下载地址，设置执行策略，开始管理流量。' }}</EmptyDescription></EmptyHeader>
        <EmptyContent><Button v-if="plans.length" variant="outline" @click="resetFilters">清除筛选</Button><Button v-else @click="$emit('createConfig')"><PlusIcon data-icon="inline-start" />新建配置</Button></EmptyContent>
      </Empty>
      <template v-else>
        <div class="grid gap-4 lg:hidden">
          <Card v-for="plan in visiblePlans" :key="`mobile-${plan.name}`" size="sm">
            <CardHeader><CardTitle><Button variant="link" class="max-w-full justify-start" @click="$emit('openPlanDetail', plan)"><span class="truncate">{{ plan.name }}</span></Button></CardTitle><CardDescription>{{ formatPlanSchedule(plan) }}</CardDescription><CardAction><Badge :variant="planState(plan).variant">{{ planState(plan).label }}</Badge></CardAction></CardHeader>
            <CardContent><dl class="grid grid-cols-3 gap-3 text-xs"><div><dt class="text-muted-foreground">速度</dt><dd class="mt-1 font-medium tabular-nums">{{ plan.speed || '0 B/s' }}</dd></div><div><dt class="text-muted-foreground">流量</dt><dd class="mt-1 font-medium tabular-nums">{{ plan.total_bytes || '0 B' }}</dd></div><div><dt class="text-muted-foreground">下载次数</dt><dd class="mt-1 font-medium tabular-nums">{{ plan.download_count || 0 }}</dd></div></dl><p class="mt-3 text-xs text-muted-foreground">{{ plan.threads || 0 }} 线程 · {{ plan.url_count || 0 }} 个 URL</p><p v-if="plan.next_run_time" class="mt-2 text-xs text-muted-foreground">下次执行 {{ formatDateTime(plan.next_run_time) }}</p></CardContent>
            <CardFooter class="flex-wrap gap-2">
              <Button size="sm" :variant="plan.running || plan.scheduler_running ? 'outline' : 'default'" @click="plan.running || plan.scheduler_running ? $emit('stopPlan', plan) : $emit('startPlan', plan)"><SquareIcon v-if="plan.running || plan.scheduler_running" data-icon="inline-start" /><PlayIcon v-else data-icon="inline-start" />{{ plan.running || plan.scheduler_running ? '停止' : '启动' }}</Button>
              <Button v-for="action in actions" :key="action.event" variant="ghost" size="icon-sm" :aria-label="`${action.label} ${plan.name}`" @click="runRowAction(action.event, plan)"><component :is="action.icon" data-icon="inline-start" /></Button>
            </CardFooter>
          </Card>
        </div>
        <div class="hidden lg:block">
          <Table>
            <TableHeader><TableRow><TableHead>配置名称</TableHead><TableHead>状态</TableHead><TableHead>实时速率 / 流量</TableHead><TableHead>执行安排</TableHead><TableHead v-if="!compact">上次运行</TableHead><TableHead class="text-right">操作</TableHead></TableRow></TableHeader>
            <TableBody id="runtime-plan-list">
              <TableRow v-for="plan in visiblePlans" :key="plan.name">
                <TableCell><Tooltip><TooltipTrigger as-child><Button variant="link" class="max-w-[180px] justify-start" @click="$emit('openPlanDetail', plan)"><span class="truncate">{{ plan.name }}</span></Button></TooltipTrigger><TooltipContent>{{ plan.name }}</TooltipContent></Tooltip><p class="text-xs text-muted-foreground">{{ plan.threads || 0 }} 线程 · {{ plan.url_count || 0 }} 个 URL</p></TableCell>
                <TableCell><Badge :variant="planState(plan).variant">{{ planState(plan).label }}</Badge></TableCell>
                <TableCell><div class="font-medium tabular-nums">{{ plan.speed || '0 B/s' }}</div><div class="mt-1 text-xs text-muted-foreground">{{ plan.total_bytes || '0 B' }} · {{ plan.download_count || 0 }} 次</div></TableCell>
                <TableCell><div class="max-w-[180px] truncate">{{ formatPlanSchedule(plan) }}</div><div class="mt-1 text-xs text-muted-foreground">{{ plan.next_run_time ? formatDateTime(plan.next_run_time) : '暂无下次执行' }}</div></TableCell>
                <TableCell v-if="!compact"><div>{{ plan.last_run?.result || '暂无记录' }}</div><div v-if="plan.last_run" class="mt-1 text-xs text-muted-foreground">{{ plan.last_run.bytes_consumed || '0 B' }} · {{ formatDateTime(plan.last_run.timestamp) }}</div></TableCell>
                <TableCell><div class="flex justify-end gap-1">
                  <Button size="sm" :variant="plan.running || plan.scheduler_running ? 'outline' : 'default'" :aria-label="`${plan.running || plan.scheduler_running ? '停止' : '启动'} ${plan.name}`" @click="plan.running || plan.scheduler_running ? $emit('stopPlan', plan) : $emit('startPlan', plan)"><SquareIcon v-if="plan.running || plan.scheduler_running" data-icon="inline-start" /><PlayIcon v-else data-icon="inline-start" />{{ plan.running || plan.scheduler_running ? '停止' : '启动' }}</Button>
                  <Tooltip v-for="action in actions" :key="action.event"><TooltipTrigger as-child><Button variant="ghost" size="icon-sm" :aria-label="`${action.label} ${plan.name}`" @click="runRowAction(action.event, plan)"><component :is="action.icon" data-icon="inline-start" /></Button></TooltipTrigger><TooltipContent>{{ action.label }}</TooltipContent></Tooltip>
                </div></TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>
      </template>
    </CardContent>
  </Card>
</template>
