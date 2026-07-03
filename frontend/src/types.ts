export type RuntimeState = 'running' | 'scheduled' | 'stopped'

export interface ConfigPayload {
  urls: string[]
  url?: string
  url_strategy?: string | null
  threads?: number | null
  limit_speed?: number | null
  duration?: number | null
  count?: number | null
  cron_expr?: string | null
  traffic_limit?: number | null
  interval?: number | null
  auto_remove_failed_url?: boolean
  auto_start?: boolean
  user_agent?: string | null
  request_headers?: Record<string, string> | null
  url_switch_interval?: number | null
  thread_start_delay?: number | null
  config_name?: string | null
  name?: string | null
}

export interface StatusPayload {
  total_bytes?: string
  speed?: string
  download_count?: number
  running: boolean
  state: RuntimeState
  config?: string
  thread_count?: number
  thread_status?: Record<string, string>
  url_usage_stats?: URLUsageStat[]
  message?: string
}

export interface URLUsageStat {
  url: string
  count: number
  percentage: number
}

export interface ThreadRow {
  id: number
  url: string
  state: 'active' | 'idle' | 'error'
  label: string
}

export interface ThreadCounts {
  total: number
  active: number
  idle: number
  errored: number
}

export interface SchedulerPayload {
  next_run_time?: string
  job_details?: string
  plans?: PlanSummary[]
  message?: string
}

export interface PlanSummary {
  name: string
  running: boolean
  scheduler_running: boolean
  next_run_time?: string
  total_bytes: string
  total_bytes_raw: number
  download_count: number
  threads: number
  url_count: number
  cron_expr?: string
  interval?: number | null
  auto_start: boolean
  url_strategy: string
  stats_history?: HistoryRecord[]
}

export interface HistoryRecord {
  timestamp: string
  result: string
  bytes_consumed?: string
  download_count: number
  bytes?: number
  speed?: number
  elapsed_seconds?: number
}

export interface LogEntry {
  seq: number
  timestamp: string
  level: string
  message: string
  color?: string
  config?: string
}

export interface RuntimeEvent {
  seq: number
  type: string
  event_type?: string
  level?: string
  title?: string
  variant?: string
  message?: string
  error?: string
  url?: string
  config?: string
  retries?: number
  all_invalid?: boolean
  history_record?: HistoryRecord
}

export interface PlanDetailPayload {
  name: string
  config: ConfigPayload
  summary: PlanSummary
  history: HistoryRecord[]
}

export type BadgeVariant = 'default' | 'secondary' | 'destructive' | 'outline'

export interface ConfigForm {
  name: string
  urls: string
  threads: string
  url_strategy: string
  limit_speed: string
  traffic_limit: string
  duration: string
  count: string
  cron_expr: string
  interval: string
  auto_remove_failed_url: boolean
  auto_start: boolean
  user_agent: string
  request_headers: string
  url_switch_interval: string
  thread_start_delay: string
}
