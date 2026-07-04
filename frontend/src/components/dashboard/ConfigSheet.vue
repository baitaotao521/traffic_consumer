<script setup lang="ts">
import { BoltIcon, Clock3Icon, RotateCcwIcon, SaveIcon, XIcon } from '@lucide/vue'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogClose, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Field, FieldContent, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  NumberField,
  NumberFieldContent,
  NumberFieldDecrement,
  NumberFieldIncrement,
  NumberFieldInput,
} from '@/components/ui/number-field'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { Spinner } from '@/components/ui/spinner'
import type { ConfigForm, HistoryRecord } from '@/types'

const numberFieldInputClass = 'px-8 text-left'

const integerNumberFieldProps = {
  min: 0,
  stepSnapping: false,
  formatOptions: {
    maximumFractionDigits: 0,
  },
}

const decimalNumberFieldProps = {
  min: 0,
  stepSnapping: false,
  formatOptions: {
    maximumFractionDigits: 3,
  },
}

defineProps<{
  cronPreview: string[]
  form: ConfigForm
  formatDateTime: (value?: string) => string
  lastRun?: HistoryRecord | null
  saving?: boolean
  testDisabled?: boolean
  testing?: boolean
}>()

const open = defineModel<boolean>('open', { required: true })

defineEmits<{
  reset: []
  save: []
  test: []
}>()
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent
      :show-close-button="false"
      class="h-[min(90vh,860px)] max-w-[calc(100vw-2rem)] gap-0 overflow-hidden p-0 sm:max-w-[1100px]"
    >
      <DialogHeader class="flex-row items-center justify-between border-b px-6 py-4">
        <DialogTitle>任务配置</DialogTitle>
        <DialogClose as-child>
          <Button
            variant="ghost"
            size="icon-sm"
            class="shrink-0"
          >
            <XIcon />
            <span class="sr-only">关闭</span>
          </Button>
        </DialogClose>
      </DialogHeader>

      <ScrollArea class="min-h-0 flex-1">
        <div class="grid gap-4 px-6 py-5 xl:grid-cols-[minmax(0,1.35fr)_minmax(0,1fr)]">
          <div class="grid gap-4">
            <Card>
              <CardHeader class="border-b py-3">
                <CardTitle>基础配置</CardTitle>
              </CardHeader>
              <CardContent class="py-4">
                <FieldGroup>
                  <Field>
                    <FieldLabel for="config-name">配置名称</FieldLabel>
                    <Input id="config-name" v-model="form.name" placeholder="输入配置名称" />
                  </Field>

                  <div class="grid gap-4 sm:grid-cols-2">
                    <Field>
                      <FieldLabel>线程数量</FieldLabel>
                      <NumberField
                        v-model="form.threads"
                        v-bind="integerNumberFieldProps"
                        class="w-full gap-0"
                      >
                        <NumberFieldContent class="w-full">
                          <NumberFieldDecrement />
                          <NumberFieldInput :class="numberFieldInputClass" />
                          <NumberFieldIncrement />
                        </NumberFieldContent>
                      </NumberField>
                    </Field>
                    <Field>
                      <FieldLabel for="url-strategy">URL 策略</FieldLabel>
                      <Select v-model="form.url_strategy">
                        <SelectTrigger id="url-strategy">
                          <SelectValue placeholder="随机均衡" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="random">随机均衡</SelectItem>
                          <SelectItem value="round_robin">轮询顺序</SelectItem>
                        </SelectContent>
                      </Select>
                    </Field>
                  </div>

                  <Field>
                    <FieldLabel for="urls">URL 列表</FieldLabel>
                    <Textarea id="urls" v-model="form.urls" rows="10" placeholder="每行一个 URL" />
                  </Field>
                </FieldGroup>
              </CardContent>
            </Card>

            <Card>
              <CardHeader class="border-b py-3">
                <CardTitle>执行限制</CardTitle>
              </CardHeader>
              <CardContent class="py-4">
                <FieldGroup class="grid gap-4 sm:grid-cols-2">
                  <Field>
                      <FieldLabel>限速 (MB/s)</FieldLabel>
                      <NumberField
                        v-model="form.limit_speed"
                        v-bind="decimalNumberFieldProps"
                        class="w-full gap-0"
                      >
                        <NumberFieldContent class="w-full">
                          <NumberFieldDecrement />
                          <NumberFieldInput :class="numberFieldInputClass" />
                          <NumberFieldIncrement />
                        </NumberFieldContent>
                      </NumberField>
                    </Field>
                  <Field>
                    <FieldLabel>总流量 (MB)</FieldLabel>
                    <NumberField
                      v-model="form.traffic_limit"
                      v-bind="integerNumberFieldProps"
                      class="w-full gap-0"
                    >
                      <NumberFieldContent class="w-full">
                        <NumberFieldDecrement />
                        <NumberFieldInput :class="numberFieldInputClass" />
                        <NumberFieldIncrement />
                      </NumberFieldContent>
                    </NumberField>
                  </Field>
                  <Field>
                    <FieldLabel>时长 (秒)</FieldLabel>
                    <NumberField
                      v-model="form.duration"
                      v-bind="integerNumberFieldProps"
                      class="w-full gap-0"
                    >
                      <NumberFieldContent class="w-full">
                        <NumberFieldDecrement />
                        <NumberFieldInput :class="numberFieldInputClass" />
                        <NumberFieldIncrement />
                      </NumberFieldContent>
                    </NumberField>
                  </Field>
                  <Field>
                    <FieldLabel>下载次数</FieldLabel>
                    <NumberField
                      v-model="form.count"
                      v-bind="integerNumberFieldProps"
                      class="w-full gap-0"
                    >
                      <NumberFieldContent class="w-full">
                        <NumberFieldDecrement />
                        <NumberFieldInput :class="numberFieldInputClass" />
                        <NumberFieldIncrement />
                      </NumberFieldContent>
                    </NumberField>
                  </Field>
                  <Field>
                    <FieldLabel>URL 切换时限 (秒)</FieldLabel>
                    <NumberField
                      v-model="form.url_switch_interval"
                      v-bind="decimalNumberFieldProps"
                      class="w-full gap-0"
                    >
                      <NumberFieldContent class="w-full">
                        <NumberFieldDecrement />
                        <NumberFieldInput :class="numberFieldInputClass" />
                        <NumberFieldIncrement />
                      </NumberFieldContent>
                    </NumberField>
                  </Field>
                  <Field>
                    <FieldLabel>线程启动间隔 (秒)</FieldLabel>
                    <NumberField
                      v-model="form.thread_start_delay"
                      v-bind="decimalNumberFieldProps"
                      class="w-full gap-0"
                    >
                      <NumberFieldContent class="w-full">
                        <NumberFieldDecrement />
                        <NumberFieldInput :class="numberFieldInputClass" />
                        <NumberFieldIncrement />
                      </NumberFieldContent>
                    </NumberField>
                  </Field>
                </FieldGroup>
              </CardContent>
            </Card>
          </div>

          <div class="grid gap-4">
            <Card>
              <CardHeader class="border-b py-3">
                <CardTitle>调度设置</CardTitle>
              </CardHeader>
              <CardContent class="grid gap-4 py-4">
                <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-1">
                  <Field>
                    <FieldLabel>间隔 (分钟)</FieldLabel>
                    <NumberField
                      v-model="form.interval"
                      v-bind="integerNumberFieldProps"
                      class="w-full gap-0"
                    >
                      <NumberFieldContent class="w-full">
                        <NumberFieldDecrement />
                        <NumberFieldInput :class="numberFieldInputClass" />
                        <NumberFieldIncrement />
                      </NumberFieldContent>
                    </NumberField>
                  </Field>
                  <Field>
                    <FieldLabel for="cron-expr">Cron 表达式</FieldLabel>
                    <Input id="cron-expr" v-model="form.cron_expr" placeholder="*/5 * * * *" />
                  </Field>
                </div>

                <Alert id="cron-preview" class="border-dashed">
                  <Clock3Icon class="text-muted-foreground" />
                  <AlertTitle>执行预览</AlertTitle>
                  <AlertDescription class="text-xs">
                    <ul v-if="cronPreview.length" class="grid gap-1">
                      <li v-for="time in cronPreview" :key="time">{{ formatDateTime(time) }}</li>
                    </ul>
                    <span v-else>暂无预览</span>
                  </AlertDescription>
                </Alert>
              </CardContent>
            </Card>

            <Card>
              <CardHeader class="border-b py-3">
                <CardTitle>请求设置</CardTitle>
              </CardHeader>
              <CardContent class="py-4">
                <FieldGroup>
                  <Field>
                    <FieldLabel for="user-agent">User-Agent</FieldLabel>
                    <Input id="user-agent" v-model="form.user_agent" />
                  </Field>

                  <Field>
                    <FieldLabel for="request-headers">自定义请求头</FieldLabel>
                    <Textarea
                      id="request-headers"
                      v-model="form.request_headers"
                      rows="6"
                      placeholder="Referer: https://example.com"
                    />
                  </Field>
                </FieldGroup>
              </CardContent>
            </Card>

            <Card>
              <CardHeader class="border-b py-3">
                <CardTitle>运行行为</CardTitle>
              </CardHeader>
              <CardContent class="py-4">
                <FieldGroup>
                  <Field orientation="horizontal" class="justify-between rounded-lg border bg-muted px-4 py-3">
                    <FieldContent>
                      <FieldLabel for="auto-remove-failed-url">失败链接自动移除</FieldLabel>
                    </FieldContent>
                    <Switch
                      id="auto-remove-failed-url"
                      v-model="form.auto_remove_failed_url"
                    />
                  </Field>

                  <Field orientation="horizontal" class="justify-between rounded-lg border bg-muted px-4 py-3">
                    <FieldContent>
                      <FieldLabel for="auto-start">启动自动运行</FieldLabel>
                    </FieldContent>
                    <Switch
                      id="auto-start"
                      v-model="form.auto_start"
                    />
                  </Field>
                </FieldGroup>
              </CardContent>
            </Card>

            <Card>
              <CardHeader class="border-b py-3">
                <CardTitle>上次运行结果</CardTitle>
              </CardHeader>
              <CardContent class="grid gap-3 py-4 text-sm">
                <div class="grid grid-cols-2 gap-3">
                  <div>
                    <div class="text-xs text-muted-foreground">状态</div>
                    <div class="font-medium">{{ lastRun?.result || '无' }}</div>
                  </div>
                  <div>
                    <div class="text-xs text-muted-foreground">流量</div>
                    <div class="font-medium">{{ lastRun?.bytes_consumed || '0 B' }}</div>
                  </div>
                  <div>
                    <div class="text-xs text-muted-foreground">下载</div>
                    <div class="font-medium">{{ lastRun?.download_count || 0 }}</div>
                  </div>
                  <div>
                    <div class="text-xs text-muted-foreground">时间</div>
                    <div class="truncate font-medium">{{ formatDateTime(lastRun?.timestamp) }}</div>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>
        </div>
      </ScrollArea>

      <DialogFooter class="m-0 px-6 py-4">
        <Button id="reset-config-btn" type="button" variant="outline" :disabled="saving" @click="$emit('reset')">
          <RotateCcwIcon />
          清空
        </Button>
        <Button
          id="test-config-btn"
          type="button"
          variant="secondary"
          :disabled="saving || testing || testDisabled"
          @click="$emit('test')"
        >
          <Spinner v-if="testing" />
          <BoltIcon v-else data-icon="inline-start" />
          测试
        </Button>
        <Button id="save-config-btn" type="button" :disabled="saving" @click="$emit('save')">
          <Spinner v-if="saving" />
          <SaveIcon v-else />
          保存配置
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
