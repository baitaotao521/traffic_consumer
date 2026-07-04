<script setup lang="ts">
import { Clock3Icon, RotateCcwIcon, SaveIcon, XIcon } from '@lucide/vue'
import { NumberFieldInput, NumberFieldRoot } from 'reka-ui'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogClose, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Field, FieldContent, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
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
import type { ConfigForm } from '@/types'

const numberFieldInputClass = 'dark:bg-input/30 border-input focus-visible:border-ring focus-visible:ring-ring/50 aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive dark:aria-invalid:border-destructive/50 disabled:bg-input/50 dark:disabled:bg-input/80 h-8 rounded-lg border bg-transparent px-2.5 py-1 text-base transition-colors file:h-6 file:text-sm file:font-medium focus-visible:ring-3 aria-invalid:ring-3 md:text-sm w-full min-w-0 outline-none file:inline-flex file:border-0 file:bg-transparent file:text-foreground placeholder:text-muted-foreground disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50'

const integerNumberFieldProps = {
  stepSnapping: false,
  formatOptions: {
    maximumFractionDigits: 0,
  },
}

const decimalNumberFieldProps = {
  stepSnapping: false,
  formatOptions: {
    maximumFractionDigits: 3,
  },
}

defineProps<{
  cronPreview: string[]
  form: ConfigForm
  formatDateTime: (value?: string) => string
}>()

const open = defineModel<boolean>('open', { required: true })

defineEmits<{
  reset: []
  save: []
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
                    <Input id="config-name" v-model="form.name" placeholder="输入测试任务标识" />
                  </Field>

                  <div class="grid gap-4 sm:grid-cols-2">
                    <Field>
                      <FieldLabel for="threads">线程数量</FieldLabel>
                      <NumberFieldRoot
                        id="threads"
                        v-model="form.threads"
                        v-bind="integerNumberFieldProps"
                        class="w-full"
                      >
                        <NumberFieldInput :class="numberFieldInputClass" />
                      </NumberFieldRoot>
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
                    <FieldLabel for="limit-speed">限速 (MB/s)</FieldLabel>
                    <NumberFieldRoot
                      id="limit-speed"
                      v-model="form.limit_speed"
                      v-bind="decimalNumberFieldProps"
                      class="w-full"
                    >
                      <NumberFieldInput :class="numberFieldInputClass" />
                    </NumberFieldRoot>
                  </Field>
                  <Field>
                    <FieldLabel for="traffic-limit">总流量 (MB)</FieldLabel>
                    <NumberFieldRoot
                      id="traffic-limit"
                      v-model="form.traffic_limit"
                      v-bind="integerNumberFieldProps"
                      class="w-full"
                    >
                      <NumberFieldInput :class="numberFieldInputClass" />
                    </NumberFieldRoot>
                  </Field>
                  <Field>
                    <FieldLabel for="duration">时长 (秒)</FieldLabel>
                    <NumberFieldRoot
                      id="duration"
                      v-model="form.duration"
                      v-bind="integerNumberFieldProps"
                      class="w-full"
                    >
                      <NumberFieldInput :class="numberFieldInputClass" />
                    </NumberFieldRoot>
                  </Field>
                  <Field>
                    <FieldLabel for="count">下载次数</FieldLabel>
                    <NumberFieldRoot
                      id="count"
                      v-model="form.count"
                      v-bind="integerNumberFieldProps"
                      class="w-full"
                    >
                      <NumberFieldInput :class="numberFieldInputClass" />
                    </NumberFieldRoot>
                  </Field>
                  <Field>
                    <FieldLabel for="url-switch-interval">URL 切换时限 (秒)</FieldLabel>
                    <NumberFieldRoot
                      id="url-switch-interval"
                      v-model="form.url_switch_interval"
                      v-bind="decimalNumberFieldProps"
                      class="w-full"
                    >
                      <NumberFieldInput :class="numberFieldInputClass" />
                    </NumberFieldRoot>
                  </Field>
                  <Field>
                    <FieldLabel for="thread-start-delay">线程启动间隔 (秒)</FieldLabel>
                    <NumberFieldRoot
                      id="thread-start-delay"
                      v-model="form.thread_start_delay"
                      v-bind="decimalNumberFieldProps"
                      class="w-full"
                    >
                      <NumberFieldInput :class="numberFieldInputClass" />
                    </NumberFieldRoot>
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
                    <FieldLabel for="interval">间隔 (分钟)</FieldLabel>
                    <NumberFieldRoot
                      id="interval"
                      v-model="form.interval"
                      v-bind="integerNumberFieldProps"
                      class="w-full"
                    >
                      <NumberFieldInput :class="numberFieldInputClass" />
                    </NumberFieldRoot>
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
                      v-model:checked="form.auto_remove_failed_url"
                    />
                  </Field>

                  <Field orientation="horizontal" class="justify-between rounded-lg border bg-muted px-4 py-3">
                    <FieldContent>
                      <FieldLabel for="auto-start">启动自动运行</FieldLabel>
                    </FieldContent>
                    <Switch
                      id="auto-start"
                      v-model:checked="form.auto_start"
                    />
                  </Field>
                </FieldGroup>
              </CardContent>
            </Card>
          </div>
        </div>
      </ScrollArea>

      <DialogFooter class="m-0 px-6 py-4">
        <Button id="reset-config-btn" type="button" variant="outline" @click="$emit('reset')">
          <RotateCcwIcon />
          清空
        </Button>
        <Button id="save-config-btn" type="button" @click="$emit('save')">
          <SaveIcon />
          保存配置
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
