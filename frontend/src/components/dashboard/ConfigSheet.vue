<script setup lang="ts">
import { BoltIcon, Clock3Icon, PlusIcon, RotateCcwIcon, SaveIcon, Trash2Icon, XIcon } from '@lucide/vue'
import { computed } from 'vue'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Field, FieldContent, FieldDescription, FieldError, FieldGroup, FieldLabel } from '@/components/ui/field'
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
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'
import { Spinner } from '@/components/ui/spinner'
import type { ConfigEditorField, ConfigValidationErrors } from '@/domain/configEditor'
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

const props = withDefaults(defineProps<{
  cronPreview: string[]
  errors?: ConfigValidationErrors
  form: ConfigForm
  formatDateTime: (value?: string) => string
  lastRun?: HistoryRecord | null
  saving?: boolean
  testDisabled?: boolean
  testing?: boolean
}>(), {
  errors: () => ({}),
})

const open = defineModel<boolean>('open', { required: true })

const emit = defineEmits<{
  closeRequest: []
  reset: []
  save: []
  test: []
}>()

function hasError(field: ConfigEditorField) {
  return Boolean(props.errors[field]?.length)
}

// URL 表格只改变编辑体验，保存/测试前仍由 projectConfigForm 统一拆分空行并校验格式。
const urlRows = computed({
  get: () => props.form.urls.split(/\r?\n/),
  set: (rows: string[]) => {
    props.form.urls = rows.length ? rows.join('\n') : ''
  },
})

function updateURLRow(index: number, value: string | number) {
  const nextRows = [...urlRows.value]
  nextRows[index] = String(value)
  urlRows.value = nextRows
}

function addURLRow() {
  urlRows.value = [...urlRows.value, '']
}

function removeURLRow(index: number) {
  const nextRows = [...urlRows.value]
  nextRows.splice(index, 1)
  urlRows.value = nextRows.length ? nextRows : ['']
}

function handleOpenUpdate(nextOpen: boolean) {
  if (nextOpen) {
    open.value = true
    return
  }
  emit('closeRequest')
}
</script>

<template>
  <Dialog :open="open" @update:open="handleOpenUpdate">
    <DialogContent
      :show-close-button="false"
      class="h-[min(90vh,860px)] max-w-[calc(100vw-2rem)] gap-0 overflow-hidden p-0 sm:max-w-[1100px]"
    >
      <DialogHeader class="flex-row items-center justify-between border-b px-6 py-4">
        <DialogTitle>任务配置</DialogTitle>
        <Button
          variant="ghost"
          size="icon-sm"
          class="shrink-0"
          @click="$emit('closeRequest')"
        >
          <XIcon data-icon="inline-start" />
          <span class="sr-only">关闭</span>
        </Button>
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
                  <Field :data-invalid="hasError('name') ? true : undefined">
                    <FieldLabel for="config-name">配置名称</FieldLabel>
                    <Input
                      id="config-name"
                      v-model="form.name"
                      :aria-invalid="hasError('name') || undefined"
                      placeholder="输入配置名称"
                    />
                    <FieldError :errors="errors.name" />
                  </Field>

                  <div class="grid gap-4 sm:grid-cols-2">
                    <Field :data-invalid="hasError('threads') ? true : undefined">
                      <FieldLabel>线程数量</FieldLabel>
                      <NumberField
                        v-model="form.threads"
                        v-bind="integerNumberFieldProps"
                        class="w-full gap-0"
                      >
                        <NumberFieldContent class="w-full">
                          <NumberFieldDecrement />
                          <NumberFieldInput
                            :aria-invalid="hasError('threads') || undefined"
                            :class="numberFieldInputClass"
                          />
                          <NumberFieldIncrement />
                        </NumberFieldContent>
                      </NumberField>
                      <FieldError :errors="errors.threads" />
                    </Field>
                    <Field :data-invalid="hasError('url_strategy') ? true : undefined">
                      <FieldLabel for="url-strategy">URL 策略</FieldLabel>
                      <Select v-model="form.url_strategy">
                        <SelectTrigger id="url-strategy" :aria-invalid="hasError('url_strategy') || undefined">
                          <SelectValue placeholder="随机均衡" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectGroup>
                            <SelectItem value="random">随机均衡</SelectItem>
                            <SelectItem value="round_robin">轮询顺序</SelectItem>
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                      <FieldError :errors="errors.url_strategy" />
                    </Field>
                  </div>

                  <Field :data-invalid="hasError('urls') ? true : undefined">
                    <div class="flex items-center justify-between gap-3">
                      <FieldLabel>URL 列表</FieldLabel>
                      <Button type="button" variant="outline" size="sm" @click="addURLRow">
                        <PlusIcon data-icon="inline-start" />
                        添加 URL
                      </Button>
                    </div>
                    <div class="overflow-hidden rounded-lg border">
                      <Table class="min-w-[520px]">
                        <TableHeader class="bg-muted">
                          <TableRow>
                            <TableHead class="w-[56px] text-right">序号</TableHead>
                            <TableHead>URL</TableHead>
                            <TableHead class="w-[48px]">
                              <span class="sr-only">操作</span>
                            </TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          <TableRow v-for="(row, index) in urlRows" :key="`url-row-${index}`">
                            <TableCell class="text-right text-muted-foreground">
                              {{ index + 1 }}
                            </TableCell>
                            <TableCell class="min-w-[360px]">
                              <Input
                                :id="`url-row-input-${index}`"
                                :model-value="row"
                                :aria-invalid="hasError('urls') || undefined"
                                placeholder="https://example.com/file"
                                @update:model-value="updateURLRow(index, $event)"
                              />
                            </TableCell>
                            <TableCell class="text-right">
                              <Button
                                type="button"
                                variant="ghost"
                                size="icon-sm"
                                :disabled="urlRows.length === 1 && !row"
                                @click="removeURLRow(index)"
                              >
                                <Trash2Icon data-icon="inline-start" />
                                <span class="sr-only">删除 URL</span>
                              </Button>
                            </TableCell>
                          </TableRow>
                        </TableBody>
                      </Table>
                    </div>
                    <FieldError :errors="errors.urls" />
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
                  <Field :data-invalid="hasError('limit_speed') ? true : undefined">
                    <FieldLabel>限速 (MB/s)</FieldLabel>
                    <NumberField
                      v-model="form.limit_speed"
                      v-bind="decimalNumberFieldProps"
                      class="w-full gap-0"
                    >
                      <NumberFieldContent class="w-full">
                        <NumberFieldDecrement />
                        <NumberFieldInput
                          :aria-invalid="hasError('limit_speed') || undefined"
                          :class="numberFieldInputClass"
                        />
                        <NumberFieldIncrement />
                      </NumberFieldContent>
                    </NumberField>
                    <FieldError :errors="errors.limit_speed" />
                  </Field>
                  <Field :data-invalid="hasError('traffic_limit') ? true : undefined">
                    <FieldLabel>总流量 (MB)</FieldLabel>
                    <NumberField
                      v-model="form.traffic_limit"
                      v-bind="integerNumberFieldProps"
                      class="w-full gap-0"
                    >
                      <NumberFieldContent class="w-full">
                        <NumberFieldDecrement />
                        <NumberFieldInput
                          :aria-invalid="hasError('traffic_limit') || undefined"
                          :class="numberFieldInputClass"
                        />
                        <NumberFieldIncrement />
                      </NumberFieldContent>
                    </NumberField>
                    <FieldError :errors="errors.traffic_limit" />
                  </Field>
                  <Field :data-invalid="hasError('duration') ? true : undefined">
                    <FieldLabel>时长 (秒)</FieldLabel>
                    <NumberField
                      v-model="form.duration"
                      v-bind="integerNumberFieldProps"
                      class="w-full gap-0"
                    >
                      <NumberFieldContent class="w-full">
                        <NumberFieldDecrement />
                        <NumberFieldInput
                          :aria-invalid="hasError('duration') || undefined"
                          :class="numberFieldInputClass"
                        />
                        <NumberFieldIncrement />
                      </NumberFieldContent>
                    </NumberField>
                    <FieldError :errors="errors.duration" />
                  </Field>
                  <Field :data-invalid="hasError('count') ? true : undefined">
                    <FieldLabel>下载次数</FieldLabel>
                    <NumberField
                      v-model="form.count"
                      v-bind="integerNumberFieldProps"
                      class="w-full gap-0"
                    >
                      <NumberFieldContent class="w-full">
                        <NumberFieldDecrement />
                        <NumberFieldInput
                          :aria-invalid="hasError('count') || undefined"
                          :class="numberFieldInputClass"
                        />
                        <NumberFieldIncrement />
                      </NumberFieldContent>
                    </NumberField>
                    <FieldError :errors="errors.count" />
                  </Field>
                  <Field :data-invalid="hasError('url_switch_interval') ? true : undefined">
                    <FieldLabel>URL 切换时限 (秒)</FieldLabel>
                    <NumberField
                      v-model="form.url_switch_interval"
                      v-bind="decimalNumberFieldProps"
                      class="w-full gap-0"
                    >
                      <NumberFieldContent class="w-full">
                        <NumberFieldDecrement />
                        <NumberFieldInput
                          :aria-invalid="hasError('url_switch_interval') || undefined"
                          :class="numberFieldInputClass"
                        />
                        <NumberFieldIncrement />
                      </NumberFieldContent>
                    </NumberField>
                    <FieldDescription>
                      单条 URL 连续下载达到该秒数后切换下一条，留空为不按时长切换。
                    </FieldDescription>
                    <FieldError :errors="errors.url_switch_interval" />
                  </Field>
                  <Field :data-invalid="hasError('thread_start_delay') ? true : undefined">
                    <FieldLabel>线程启动间隔 (秒)</FieldLabel>
                    <NumberField
                      v-model="form.thread_start_delay"
                      v-bind="decimalNumberFieldProps"
                      class="w-full gap-0"
                    >
                      <NumberFieldContent class="w-full">
                        <NumberFieldDecrement />
                        <NumberFieldInput
                          :aria-invalid="hasError('thread_start_delay') || undefined"
                          :class="numberFieldInputClass"
                        />
                        <NumberFieldIncrement />
                      </NumberFieldContent>
                    </NumberField>
                    <FieldDescription>
                      每个线程启动前等待的秒数，用于错峰发起连接，留空为同时启动。
                    </FieldDescription>
                    <FieldError :errors="errors.thread_start_delay" />
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
                  <Field :data-invalid="hasError('interval') ? true : undefined">
                    <FieldLabel>间隔 (分钟)</FieldLabel>
                    <NumberField
                      v-model="form.interval"
                      v-bind="integerNumberFieldProps"
                      class="w-full gap-0"
                    >
                      <NumberFieldContent class="w-full">
                        <NumberFieldDecrement />
                        <NumberFieldInput
                          :aria-invalid="hasError('interval') || undefined"
                          :class="numberFieldInputClass"
                        />
                        <NumberFieldIncrement />
                      </NumberFieldContent>
                    </NumberField>
                    <FieldError :errors="errors.interval" />
                  </Field>
                  <Field :data-invalid="hasError('cron_expr') ? true : undefined">
                    <FieldLabel for="cron-expr">Cron 表达式</FieldLabel>
                    <Input
                      id="cron-expr"
                      v-model="form.cron_expr"
                      :aria-invalid="hasError('cron_expr') || undefined"
                      placeholder="*/5 * * * *"
                    />
                    <FieldError :errors="errors.cron_expr" />
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
                  <Field :data-invalid="hasError('user_agent') ? true : undefined">
                    <FieldLabel for="user-agent">User-Agent</FieldLabel>
                    <Input
                      id="user-agent"
                      v-model="form.user_agent"
                      :aria-invalid="hasError('user_agent') || undefined"
                    />
                    <FieldError :errors="errors.user_agent" />
                  </Field>

                  <Field :data-invalid="hasError('request_headers') ? true : undefined">
                    <FieldLabel for="request-headers">自定义请求头</FieldLabel>
                    <Textarea
                      id="request-headers"
                      v-model="form.request_headers"
                      :aria-invalid="hasError('request_headers') || undefined"
                      rows="6"
                      placeholder="Referer: https://example.com"
                    />
                    <FieldError :errors="errors.request_headers" />
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
          <RotateCcwIcon data-icon="inline-start" />
          清空
        </Button>
        <Button
          id="test-config-btn"
          type="button"
          variant="secondary"
          :disabled="saving || testing || testDisabled"
          @click="$emit('test')"
        >
          <Spinner v-if="testing" data-icon="inline-start" />
          <BoltIcon v-else data-icon="inline-start" />
          测试
        </Button>
        <Button id="save-config-btn" type="button" :disabled="saving" @click="$emit('save')">
          <Spinner v-if="saving" data-icon="inline-start" />
          <SaveIcon v-else data-icon="inline-start" />
          保存配置
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
