<script setup lang="ts">
import { HistoryIcon } from '@lucide/vue'

import { Card, CardContent, CardDescription } from '@/components/ui/card'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Table, TableBody, TableCell, TableEmpty, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import type { HistoryRecord, PlanDetailView } from '@/types'

defineProps<{
  formatDateTime: (value?: string) => string
  planDetail: PlanDetailView | null
}>()

const open = defineModel<boolean>('open', { required: true })
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent id="plan-detail-modal" class="max-w-3xl p-0 sm:max-w-3xl">
      <DialogHeader class="border-b px-6 py-4">
        <DialogTitle>运行计划详情</DialogTitle>
      </DialogHeader>
      <div v-if="planDetail" class="grid gap-4 px-6 py-5">
        <div id="plan-detail-summary" class="grid gap-4 md:grid-cols-3">
          <Card size="sm">
            <CardContent>
              <CardDescription>计划名称</CardDescription>
              <div id="plan-detail-name" class="truncate font-semibold">{{ planDetail.name }}</div>
            </CardContent>
          </Card>
          <Card size="sm">
            <CardContent>
              <CardDescription>下次执行</CardDescription>
              <div id="plan-detail-next-run" class="font-semibold">
                {{ formatDateTime(planDetail.summary.next_run_time) }}
              </div>
            </CardContent>
          </Card>
          <Card size="sm">
            <CardContent>
              <CardDescription>任务累计流量 / 下载数</CardDescription>
              <div id="plan-detail-total" class="font-semibold">
                {{ planDetail.summary.total_bytes }} / {{ planDetail.summary.download_count }}
              </div>
            </CardContent>
          </Card>
        </div>

        <ScrollArea class="max-h-[360px] rounded-lg border">
          <Table>
            <TableHeader class="bg-muted">
              <TableRow>
                <TableHead>执行时间</TableHead>
                <TableHead>结果</TableHead>
                <TableHead>流量</TableHead>
                <TableHead class="text-right">下载数</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody id="plan-detail-history-body">
              <TableEmpty v-if="!planDetail.history.length" :colspan="4" class="py-0">
                <Empty class="p-0">
                  <EmptyHeader>
                    <EmptyMedia variant="icon">
                      <HistoryIcon />
                    </EmptyMedia>
                    <EmptyTitle>暂无历史调度数据</EmptyTitle>
                  </EmptyHeader>
                </Empty>
              </TableEmpty>
              <TableRow
                v-for="item in planDetail.history as HistoryRecord[]"
                :key="`${item.timestamp}-${item.download_count}`"
              >
                <TableCell>{{ formatDateTime(item.timestamp) }}</TableCell>
                <TableCell>{{ item.result || '未知' }}</TableCell>
                <TableCell>{{ item.bytes_consumed || '0 B' }}</TableCell>
                <TableCell class="text-right">{{ item.download_count || 0 }}</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </ScrollArea>
      </div>
    </DialogContent>
  </Dialog>
</template>
