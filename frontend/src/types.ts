import type {
  ConfigWritable,
  HistoryRecord,
  LogEntry,
  PlanDetailPayload,
  PlanSummary,
  SchedulerPayload,
  StatusPayload,
  UrlUsageStat,
} from '@/api/generated'

export type RuntimeState = 'running' | 'scheduled' | 'stopped'

export type ConfigPayload = ConfigWritable
export type URLUsageStat = UrlUsageStat
export type { HistoryRecord, LogEntry, PlanSummary, SchedulerPayload, StatusPayload }

export type PlanDetailView = Omit<PlanDetailPayload, 'history'> & {
  history: HistoryRecord[]
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
