import type {
  ConfigPayload,
  LogEntry,
  PlanDetailPayload,
  SchedulerPayload,
  StatusPayload,
  RuntimeEvent,
} from '@/types'

async function apiFetch<T>(url: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(url, {
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers || {}),
    },
    ...options,
  })
  const text = await response.text()
  const data = text ? JSON.parse(text) : {}
  if (!response.ok) {
    throw new Error(data.message || data.error || '请求失败')
  }
  return data as T
}

function encodedName(name: string): string {
  return encodeURIComponent(name)
}

export function listConfigs(): Promise<{ configs: string[] }> {
  return apiFetch('/api/configs')
}

export function getConfig(name: string): Promise<{ name: string; config: ConfigPayload }> {
  return apiFetch(`/api/configs/${encodedName(name)}`)
}

export function saveConfig(name: string, config: ConfigPayload): Promise<{ message: string }> {
  return apiFetch(`/api/configs/${encodedName(name)}`, {
    method: 'PUT',
    body: JSON.stringify(config),
  })
}

export function deleteConfig(name: string): Promise<{ message: string }> {
  return apiFetch(`/api/configs/${encodedName(name)}`, { method: 'DELETE' })
}

export function startRuntime(config: ConfigPayload): Promise<StatusPayload> {
  return apiFetch('/api/runtime/start', { method: 'POST', body: JSON.stringify(config) })
}

export function startTempTest(config: ConfigPayload): Promise<StatusPayload> {
  return apiFetch('/api/runtime/temp-test', { method: 'POST', body: JSON.stringify(config) })
}

export function stopRuntime(): Promise<{ message: string }> {
  return apiFetch('/api/runtime/stop', { method: 'POST', body: '{}' })
}

export function stopScheduler(): Promise<{ message: string }> {
  return apiFetch('/api/runtime/scheduler/stop', { method: 'POST', body: '{}' })
}

export function stopPlan(name: string): Promise<SchedulerPayload> {
  return apiFetch(`/api/runtime/plans/${encodedName(name)}/stop`, { method: 'POST', body: '{}' })
}

export function getStatus(): Promise<StatusPayload> {
  return apiFetch('/api/runtime/status')
}

export function getScheduler(): Promise<SchedulerPayload> {
  return apiFetch('/api/runtime/scheduler')
}

export function getPlanDetail(name: string): Promise<PlanDetailPayload> {
  return apiFetch(`/api/runtime/plans/${encodedName(name)}`)
}

export function getLogs(since: number): Promise<{ logs: LogEntry[]; latest: number }> {
  return apiFetch(`/api/runtime/logs?since=${since}`)
}

export function getEvents(since: number): Promise<{ events: RuntimeEvent[]; latest: number }> {
  return apiFetch(`/api/runtime/events?since=${since}`)
}

export function previewCron(cronExpr: string): Promise<string[]> {
  return apiFetch('/api/preview_cron', {
    method: 'POST',
    body: JSON.stringify({ cron_expr: cronExpr }),
  })
}
