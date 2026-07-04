import type { Config, DashboardSnapshot, Event, PlanDetailPayload, SchedulerPayload, StatusPayload } from '@/api/generated'
import type {
  BadgeVariant,
  ConfigForm,
  ConfigPayload,
  PlanDetailView,
  PlanSummary,
  RuntimeState,
  ThreadCounts,
  ThreadRow,
} from '@/types'

export const emptyStatus: StatusPayload = {
  running: false,
  state: 'stopped',
  total_bytes: '0 B',
  speed: '0 B/s',
  download_count: 0,
  thread_count: 0,
  thread_status: {},
  url_usage_stats: [],
}

export const emptyScheduler: SchedulerPayload = {
  next_run_time: '',
  job_details: '',
  plans: [],
}

export const emptyDashboard: DashboardSnapshot = {
  configs: [],
  plans: [],
  scheduler: emptyScheduler,
  status: emptyStatus,
}

export function normalizeStatus(status?: StatusPayload): StatusPayload {
  return {
    ...emptyStatus,
    ...(status || {}),
    thread_status: status?.thread_status || {},
    url_usage_stats: status?.url_usage_stats || [],
  }
}

export function normalizeScheduler(scheduler?: SchedulerPayload, plans: PlanSummary[] = []): SchedulerPayload {
  return {
    ...emptyScheduler,
    ...(scheduler || {}),
    plans: scheduler?.plans || plans,
  }
}

export function normalizeDashboard(response?: DashboardSnapshot): DashboardSnapshot {
  if (!response) return emptyDashboard
  const plans = response.plans || response.scheduler?.plans || []
  return {
    configs: response.configs || [],
    plans,
    scheduler: normalizeScheduler(response.scheduler, plans),
    status: normalizeStatus(response.status),
  }
}

export function configNames(snapshot: DashboardSnapshot): string[] {
  return (snapshot.configs || []).map((config) => config.name)
}

export function statusState(status: StatusPayload): RuntimeState {
  if (status.state === 'running' || status.state === 'scheduled') return status.state
  return 'stopped'
}

export function statusLabel(status: StatusPayload): string {
  const state = statusState(status)
  if (state === 'running') return '运行中'
  if (state === 'scheduled') return '等待调度'
  return '已停止'
}

export function statusVariant(status: StatusPayload): BadgeVariant {
  const state = statusState(status)
  if (state === 'running') return 'default'
  if (state === 'scheduled') return 'secondary'
  return 'outline'
}

export function currentConfigName(status: StatusPayload, selectedConfigName: string): string {
  return status.config || selectedConfigName || '未配置'
}

export function threadRows(status: StatusPayload): ThreadRow[] {
  const map = status.thread_status || {}
  const total = Number(status.thread_count || Object.keys(map).length || 0)
  return Array.from({ length: total }, (_, index) => {
    const id = index + 1
    const url = map[String(id)] || '等待分配'
    const failed = url.includes('已失效') || url.includes('无可用')
    const active = !failed && url !== '等待分配' && url !== '等待中...'
    return {
      id,
      url,
      state: failed ? 'error' : active ? 'active' : 'idle',
      label: failed ? (url.includes('无可用') ? '耗尽' : '失效') : active ? '运行中' : '待命',
    }
  })
}

export function threadCounts(rows: ThreadRow[]): ThreadCounts {
  const active = rows.filter((row) => row.state === 'active').length
  const errored = rows.filter((row) => row.state === 'error').length
  const idle = Math.max(rows.length - active - errored, 0)
  return { total: rows.length, active, idle, errored }
}

export function parseSpeed(value?: string): number {
  if (!value) return 0
  const match = value.match(/(\d+(?:\.\d+)?)\s*MB\/s/i)
  return match ? Number.parseFloat(match[1]) : 0
}

export function headersToText(headers: ConfigPayload['request_headers']): string {
  if (!headers) return ''
  return Object.entries(headers)
    .map(([name, value]) => `${name}: ${value}`)
    .join('\n')
}

export function applyConfigToForm(form: ConfigForm, name = '', config: ConfigPayload = { urls: [] }) {
  form.name = name
  form.urls = Array.isArray(config.urls) ? config.urls.join('\n') : ''
  form.threads = config.threads
  form.url_strategy = config.url_strategy || 'random'
  form.limit_speed = config.limit_speed
  form.traffic_limit = config.traffic_limit
  form.duration = config.duration
  form.count = config.count
  form.cron_expr = config.cron_expr || ''
  form.interval = config.interval
  form.auto_remove_failed_url = Boolean(config.auto_remove_failed_url)
  form.auto_start = Boolean(config.auto_start)
  form.user_agent = config.user_agent || ''
  form.request_headers = headersToText(config.request_headers)
  form.url_switch_interval = config.url_switch_interval
  form.thread_start_delay = config.thread_start_delay
}

function normalizeHeaders(value: string): Record<string, string> | undefined {
  const result: Record<string, string> = {}
  value.split(/\r?\n/).forEach((line) => {
    const index = line.indexOf(':')
    if (index <= 0) return
    const name = line.slice(0, index).trim()
    const headerValue = line.slice(index + 1).trim()
    if (name && headerValue) result[name] = headerValue
  })
  return Object.keys(result).length ? result : undefined
}

function assignNumber(
  target: ConfigPayload,
  key: 'threads' | 'limit_speed' | 'traffic_limit' | 'duration' | 'count' | 'interval' | 'url_switch_interval' | 'thread_start_delay',
  value: number | undefined,
) {
  if (typeof value === 'number' && Number.isFinite(value)) {
    target[key] = value
  }
}

export function configFromForm(form: ConfigForm): ConfigPayload {
  const name = form.name.trim()
  const config: ConfigPayload = {
    name,
    config_name: name,
    // 显式空 URL 列表表示用户刻意保存空链接，不能省略成后端默认 URL。
    urls: form.urls
      .split(/\r?\n/)
      .map((url) => url.trim())
      .filter(Boolean),
    url_strategy: form.url_strategy || 'random',
    auto_remove_failed_url: form.auto_remove_failed_url,
    auto_start: form.auto_start,
  }

  assignNumber(config, 'threads', form.threads)
  assignNumber(config, 'limit_speed', form.limit_speed)
  assignNumber(config, 'traffic_limit', form.traffic_limit)
  assignNumber(config, 'duration', form.duration)
  assignNumber(config, 'count', form.count)
  assignNumber(config, 'interval', form.interval)
  assignNumber(config, 'url_switch_interval', form.url_switch_interval)
  assignNumber(config, 'thread_start_delay', form.thread_start_delay)

  const cronExpr = form.cron_expr.trim()
  if (cronExpr) config.cron_expr = cronExpr

  const userAgent = form.user_agent.trim()
  if (userAgent) config.user_agent = userAgent

  const headers = normalizeHeaders(form.request_headers)
  if (headers) config.request_headers = headers

  return config
}

export function configWithName(config: Config | ConfigPayload, name: string): ConfigPayload {
  return {
    ...config,
    name,
    config_name: name,
    urls: config.urls || [],
  }
}

export function normalizePlanDetail(detail: PlanDetailPayload | null | undefined): PlanDetailView | null {
  if (!detail) return null
  return {
    ...detail,
    history: detail.history || [],
  }
}

export function formatDateTime(value?: string): string {
  if (!value) return '无'
  const date = new Date(value.includes('T') ? value : value.replace(' ', 'T'))
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

export function formatPlanSchedule(plan: PlanSummary): string {
  if (plan.cron_expr) return `Cron ${plan.cron_expr}`
  if (plan.interval) return `每 ${plan.interval} 分钟`
  return '手动任务'
}

export function planState(plan: PlanSummary): { label: string; variant: BadgeVariant } {
  if (plan.running) return { label: '运行中', variant: 'default' }
  if (plan.scheduler_running) return { label: '计划中', variant: 'secondary' }
  return { label: '已停止', variant: 'outline' }
}

export function logLevelClass(level: string) {
  if (level === 'error') return 'text-destructive'
  if (level === 'warning') return 'text-primary'
  if (level === 'success') return 'text-emerald-600'
  return 'text-muted-foreground'
}

export type LogFilterOption = {
  value: string
  label: string
}

export const logSourceOptions: LogFilterOption[] = [
  { value: 'scheduler', label: '调度' },
  { value: 'runtime', label: '运行' },
  { value: 'download', label: '下载' },
]

export const logLevelOptions: LogFilterOption[] = [
  { value: 'info', label: '信息' },
  { value: 'warning', label: '警告' },
  { value: 'error', label: '错误' },
  { value: 'success', label: '完成' },
]

export function logSourceLabel(source?: string): string {
  if (source === 'scheduler') return '调度'
  if (source === 'download') return '下载'
  return '运行'
}

export function logSourceVariant(source?: string): BadgeVariant {
  if (source === 'scheduler') return 'secondary'
  if (source === 'download') return 'default'
  return 'outline'
}

export function normalizeFilterSelection(options: LogFilterOption[], values: string[], fallback: string[]): string[] {
  const requested = new Set(values)
  const ordered = options.map((option) => option.value).filter((value) => requested.has(value))
  return ordered.length ? ordered : fallback
}

export type ActivityToast = {
  title: string
  description?: string
  variant: 'default' | 'success' | 'info' | 'warning' | 'destructive'
}

export type ActivityProjection = {
  refreshDashboard: boolean
  toast?: ActivityToast
}

function activityToastVariant(event: Event): ActivityToast['variant'] {
  if (event.type === 'error' || event.variant === 'danger' || event.level === 'error') return 'destructive'
  if (event.type === 'download_failure' || event.event_type === 'invalid_url' || event.level === 'warning') return 'warning'
  if (event.variant === 'success' || event.level === 'success') return 'success'
  if (event.variant === 'info') return 'info'
  return 'default'
}

function activityToastTitle(event: Event, variant: ActivityToast['variant']): string {
  const title = event.title?.trim()
  if (title) return title

  const message = event.message?.trim()
  if (message) return message

  if (variant === 'destructive') return '操作失败'
  if (variant === 'warning') return '需要处理'
  if (variant === 'success') return '处理完成'
  return '状态更新'
}

function activityToastDescription(event: Event, title: string): string | undefined {
  // 活动提示只保留补充信息，避免把标题和正文原样重复两遍。
  const lines = [event.message?.trim(), event.error?.trim()]
    .filter((line, index, items): line is string => Boolean(line) && line !== title && items.indexOf(line) === index)

  return lines.length ? lines.join('\n') : undefined
}

export function projectActivityEvent(event: Event): ActivityProjection {
  if (event.type === 'history_update') {
    return { refreshDashboard: true }
  }
  if (event.type === 'status_update') {
    // 运行态成功提示统一走命令响应，避免和轮询活动里的状态事件重复弹出。
    return { refreshDashboard: false }
  }
  if (['error', 'download_failure'].includes(event.type) || event.event_type === 'invalid_url') {
    const variant = activityToastVariant(event)
    const title = activityToastTitle(event, variant)
    return {
      refreshDashboard: false,
      toast: {
        title,
        description: activityToastDescription(event, title),
        variant,
      },
    }
  }
  return { refreshDashboard: false }
}
