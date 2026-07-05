import type {
  ConfigWritable,
  HistoryRecord,
  LogEntry,
  PlanDetailPayload,
  PlanSummary,
  SchedulerPayload,
  StatusPayload,
  ThreadMetric,
  UrlUsageStat,
} from '@/api/generated'

export type RuntimeState = 'running' | 'scheduled' | 'stopped'

export type ConfigPayload = ConfigWritable
export type ThreadMetricPayload = ThreadMetric
export type URLUsageStat = UrlUsageStat
export type { HistoryRecord, LogEntry, PlanSummary, SchedulerPayload, StatusPayload }

export type PlanDetailView = Omit<PlanDetailPayload, 'history'> & {
  history: HistoryRecord[]
}

export interface ThreadRow {
  /** 线程编号，对应后端下载 goroutine 的业务序号。 */
  id: number
  /** 当前任务展示旧状态字符串中的 URL 或异常状态。 */
  url: string
  /** 线程状态决定徽标文案和异常计数。 */
  state: 'active' | 'idle' | 'error'
  /** 线程状态徽标直接展示给用户的短文案。 */
  label: string
  /** 响应头返回耗时的展示值，尚未发起请求时为空占位。 */
  responseTime: string
  /** 响应体读取进度百分比，未知长度时保持 0。 */
  progress: number
  /** 进度列直接展示给用户的短文案。 */
  progressLabel: string
}

export interface ThreadCounts {
  total: number
  active: number
  idle: number
  errored: number
}

export type BadgeVariant = 'default' | 'secondary' | 'destructive' | 'outline'

export interface ConfigForm {
  name: string
  urls: string
  threads: number | undefined
  url_strategy: string
  limit_speed: number | undefined
  traffic_limit: number | undefined
  duration: number | undefined
  count: number | undefined
  cron_expr: string
  interval: number | undefined
  auto_remove_failed_url: boolean
  auto_start: boolean
  user_agent: string
  request_headers: string
  url_switch_interval: number | undefined
  thread_start_delay: number | undefined
}
