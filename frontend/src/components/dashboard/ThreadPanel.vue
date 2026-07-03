<script setup lang="ts">
import { ListIcon } from '@lucide/vue'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty'
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
    <CardHeader class="border-b py-3">
      <CardTitle>线程状态监控</CardTitle>
    </CardHeader>
    <CardContent class="p-0">
      <ScrollArea class="h-[300px]">
        <Table>
          <TableHeader class="bg-muted">
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
              <TableCell class="max-w-[320px] truncate" :title="row.url">{{ row.url }}</TableCell>
              <TableCell>--</TableCell>
              <TableCell>
                <Progress :model-value="row.state === 'active' ? 100 : row.state === 'error' ? 20 : 0" />
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </ScrollArea>
      <div class="grid grid-cols-4 gap-3 border-t px-4 py-3 text-xs">
        <div>总线程 <span class="ml-1 font-semibold">{{ threadCounts.total }}</span></div>
        <div>活跃 <span id="active-thread-count" class="ml-1 font-semibold">{{ threadCounts.active }}</span></div>
        <div>待命 <span id="idle-thread-count" class="ml-1 font-semibold">{{ threadCounts.idle }}</span></div>
        <div>异常 <span id="errored-thread-count" class="ml-1 font-semibold text-destructive">{{ threadCounts.errored }}</span></div>
      </div>
    </CardContent>
  </Card>
</template>
