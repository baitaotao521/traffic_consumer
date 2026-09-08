<script setup lang="ts">
import { ListIcon } from '@lucide/vue'

import { Badge } from '@/components/ui/badge'
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty'
import { HoverCard, HoverCardContent, HoverCardTrigger } from '@/components/ui/hover-card'
import { Progress } from '@/components/ui/progress'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Table, TableBody, TableCell, TableEmpty, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import type { ThreadCounts, ThreadRow } from '@/types'

defineProps<{
  threadCounts: ThreadCounts
  threadRows: ThreadRow[]
}>()
</script>

<template>
  <Card>
    <CardHeader>
      <CardTitle>工作线程</CardTitle>
      <CardDescription>共 {{ threadCounts.total }} 个线程 · {{ threadCounts.active }} 个活跃 · {{ threadCounts.idle }} 个待命</CardDescription>
      <CardAction><Badge :variant="threadCounts.errored ? 'destructive' : 'secondary'">{{ threadCounts.errored }} 个异常</Badge></CardAction>
    </CardHeader>
    <CardContent class="p-0">
      <div class="md:hidden">
        <div v-if="!threadRows.length" class="p-6">
          <Empty class="p-0">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <ListIcon />
              </EmptyMedia>
              <EmptyTitle>暂无线程数据</EmptyTitle>
            </EmptyHeader>
          </Empty>
        </div>
        <div v-else class="divide-y">
          <div v-for="row in threadRows" :key="`mobile-thread-${row.id}`" class="grid gap-3 px-4 py-3">
            <div class="flex items-center justify-between gap-3">
              <div class="font-medium">线程 {{ row.id }}</div>
              <Badge :variant="row.state === 'active' ? 'default' : row.state === 'error' ? 'destructive' : 'secondary'">
                {{ row.label }}
              </Badge>
            </div>
            <div class="break-all text-xs text-muted-foreground">{{ row.url }}</div>
            <div class="grid grid-cols-2 gap-2 text-xs">
              <div>
                <div class="text-muted-foreground">响应时间</div>
                <div class="font-mono font-medium tabular-nums">{{ row.responseTime }}</div>
              </div>
              <div>
                <div class="text-muted-foreground">进度</div>
                <div class="font-mono font-medium tabular-nums">{{ row.progressLabel }}</div>
              </div>
            </div>
            <Progress class="h-1.5" :model-value="row.progress" />
          </div>
        </div>
      </div>

      <ScrollArea class="hidden h-[min(60vh,640px)] md:block">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead class="w-[96px]">线程ID</TableHead>
              <TableHead class="w-[100px]">状态</TableHead>
              <TableHead>当前任务</TableHead>
              <TableHead class="w-[120px]">响应时间</TableHead>
              <TableHead class="w-[140px]">进度</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody id="thread-status-list">
            <TableEmpty v-if="!threadRows.length" :colspan="5" class="py-0">
              <Empty class="p-0">
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <ListIcon />
                  </EmptyMedia>
                  <EmptyTitle>暂无线程数据</EmptyTitle>
                </EmptyHeader>
              </Empty>
            </TableEmpty>
            <TableRow v-for="row in threadRows" :key="row.id">
              <TableCell>线程 {{ row.id }}</TableCell>
              <TableCell>
                <Badge :variant="row.state === 'active' ? 'default' : row.state === 'error' ? 'destructive' : 'secondary'">
                  {{ row.label }}
                </Badge>
              </TableCell>
              <TableCell class="max-w-[320px]">
                <HoverCard :open-delay="120">
                  <HoverCardTrigger as-child>
                    <span class="block truncate">{{ row.url }}</span>
                  </HoverCardTrigger>
                  <HoverCardContent class="w-96 break-all">
                    {{ row.url }}
                  </HoverCardContent>
                </HoverCard>
              </TableCell>
              <TableCell>{{ row.responseTime }}</TableCell>
              <TableCell>
                <div class="flex items-center gap-2">
                  <Progress class="h-1.5 flex-1" :model-value="row.progress" />
                  <span class="w-12 text-right text-xs text-muted-foreground">{{ row.progressLabel }}</span>
                </div>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </ScrollArea>
      <div class="grid grid-cols-2 gap-3 border-t px-4 py-3 text-xs sm:grid-cols-4">
        <div>总线程 <span class="ml-1 font-semibold">{{ threadCounts.total }}</span></div>
        <div>活跃 <span id="active-thread-count" class="ml-1 font-semibold">{{ threadCounts.active }}</span></div>
        <div>待命 <span id="idle-thread-count" class="ml-1 font-semibold">{{ threadCounts.idle }}</span></div>
        <div>异常 <span id="errored-thread-count" class="ml-1 font-semibold text-destructive">{{ threadCounts.errored }}</span></div>
      </div>
    </CardContent>
  </Card>
</template>
