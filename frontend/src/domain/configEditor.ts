import type { ConfigForm, ConfigPayload, PlanSummary, StatusPayload } from '@/types'

export type ConfigEditorMode = 'create' | 'edit'
export type ConfigValidationMode = 'save' | 'start' | 'temp-test'

export type ConfigEditorField =
  | keyof ConfigForm
  | 'form'

export type ConfigValidationErrors = Partial<Record<ConfigEditorField, string[]>>

export type ConfigProjection =
  | {
      ok: true
      payload: ConfigPayload
      targetName: string
      errors: ConfigValidationErrors
    }
  | {
      ok: false
      payload: null
      targetName: string
      errors: ConfigValidationErrors
    }

const limits = {
  configNameRunes: 80,
  urls: 500,
  urlLength: 4096,
  threads: 512,
  limitSpeed: 10240,
  trafficLimit: 1073741824,
  duration: 31536000,
  count: 1000000000,
  interval: 525600,
  seconds: 86400,
  minSwitchSeconds: 0.001,
}

const monthNames = new Map([
  ['jan', 1],
  ['feb', 2],
  ['mar', 3],
  ['apr', 4],
  ['may', 5],
  ['jun', 6],
  ['jul', 7],
  ['aug', 8],
  ['sep', 9],
  ['oct', 10],
  ['nov', 11],
  ['dec', 12],
])

const dayNames = new Map([
  ['sun', 0],
  ['mon', 1],
  ['tue', 2],
  ['wed', 3],
  ['thu', 4],
  ['fri', 5],
  ['sat', 6],
])

function addError(errors: ConfigValidationErrors, field: ConfigEditorField, message: string) {
  errors[field] = [...(errors[field] || []), message]
}

export function firstConfigError(errors: ConfigValidationErrors): string | undefined {
  const order: ConfigEditorField[] = [
    'name',
    'urls',
    'threads',
    'url_strategy',
    'limit_speed',
    'traffic_limit',
    'duration',
    'count',
    'interval',
    'cron_expr',
    'user_agent',
    'request_headers',
    'url_switch_interval',
    'thread_start_delay',
    'form',
  ]
  for (const field of order) {
    const message = errors[field]?.[0]
    if (message) return message
  }
  return undefined
}

function hasErrors(errors: ConfigValidationErrors) {
  return Object.values(errors).some((messages) => messages && messages.length > 0)
}

export function normalizeConfigName(name: string) {
  const text = name.trim()
  return text || 'default'
}

export function sameConfigName(left: string, right: string) {
  return normalizeConfigName(left) === normalizeConfigName(right)
}

export function configNameExists(names: string[], targetName: string) {
  const target = normalizeConfigName(targetName)
  return names.some((name) => normalizeConfigName(name) === target)
}

function runeLength(value: string) {
  return Array.from(value).length
}

function validateName(name: string, errors: ConfigValidationErrors) {
  if (!name) {
    addError(errors, 'name', '请填写配置名称。')
    return
  }
  if (runeLength(name) > limits.configNameRunes) {
    addError(errors, 'name', `配置名称不能超过 ${limits.configNameRunes} 个字符。`)
  }
  if (/[\\/\u0000-\u001f\u007f\u2028\u2029]/u.test(name)) {
    addError(errors, 'name', '配置名称不能包含路径分隔符或控制字符。')
  }
}

function parseURLs(value: string, mode: ConfigValidationMode, errors: ConfigValidationErrors) {
  const urls: string[] = []
  for (const rawLine of value.split(/\r?\n/)) {
    const text = rawLine.trim()
    if (!text) continue
    if (text.length > limits.urlLength) {
      addError(errors, 'urls', `URL 不能超过 ${limits.urlLength} 个字符。`)
      continue
    }
    if (/\s/u.test(text)) {
      addError(errors, 'urls', 'URL 必须是有效的 http 或 https 地址。')
      continue
    }
    try {
      const parsed = new URL(text)
      if ((parsed.protocol !== 'http:' && parsed.protocol !== 'https:') || !parsed.host) {
        addError(errors, 'urls', 'URL 必须是有效的 http 或 https 地址。')
        continue
      }
    } catch {
      addError(errors, 'urls', 'URL 必须是有效的 http 或 https 地址。')
      continue
    }
    // 重复 URL 保留为用户输入的多条下载目标，前端只负责过滤空行和拦截非法地址。
    urls.push(text)
  }
  if (urls.length > limits.urls) {
    addError(errors, 'urls', `URL 数量不能超过 ${limits.urls} 条。`)
  }
  if (mode !== 'save' && urls.length === 0) {
    addError(errors, 'urls', '启动或测试配置前至少需要 1 条 URL。')
  }
  return urls
}

function invalidHeaderName(name: string) {
  return /[:\s\u0000-\u001f\u007f]/u.test(name)
}

function parseHeaders(value: string, userAgent: string, errors: ConfigValidationErrors) {
  const headers: Record<string, string> = {}
  const seen = new Set<string>()
  for (const rawLine of value.split(/\r?\n/)) {
    const line = rawLine.trim()
    if (!line) continue
    const separator = line.indexOf(':')
    if (separator <= 0) {
      addError(errors, 'request_headers', '自定义请求头必须使用“名称: 值”格式。')
      continue
    }
    const name = line.slice(0, separator).trim()
    const headerValue = line.slice(separator + 1).trim()
    if (!name || !headerValue) {
      addError(errors, 'request_headers', '自定义请求头必须使用非空名称和值。')
      continue
    }
    if (invalidHeaderName(name)) {
      addError(errors, 'request_headers', '自定义请求头名称不能包含冒号、空白或控制字符。')
      continue
    }
    const key = name.toLowerCase()
    if (seen.has(key)) {
      addError(errors, 'request_headers', '自定义请求头名称不能重复。')
      continue
    }
    if (userAgent && key === 'user-agent') {
      addError(errors, 'request_headers', 'User-Agent 请只在专用字段中配置。')
      continue
    }
    seen.add(key)
    headers[name] = headerValue
  }
  return Object.keys(headers).length ? headers : undefined
}

function integerValue(value: number | undefined) {
  if (typeof value !== 'number' || !Number.isFinite(value) || !Number.isInteger(value)) return null
  return value
}

function finiteValue(value: number | undefined) {
  if (typeof value !== 'number' || !Number.isFinite(value)) return null
  return value
}

function optionalInteger(
  value: number | undefined,
  field: ConfigEditorField,
  label: string,
  min: number,
  max: number,
  errors: ConfigValidationErrors,
) {
  if (value === undefined || value === null || value === 0) return undefined
  const parsed = integerValue(value)
  if (parsed === null || parsed < min || parsed > max) {
    addError(errors, field, `${label}必须在 ${min} 到 ${max} 之间。`)
    return undefined
  }
  return parsed
}

function optionalNumber(
  value: number | undefined,
  field: ConfigEditorField,
  label: string,
  min: number,
  max: number,
  errors: ConfigValidationErrors,
) {
  if (value === undefined || value === null || value === 0) return undefined
  const parsed = finiteValue(value)
  if (parsed === null || parsed < min || parsed > max) {
    addError(errors, field, `${label}必须在 ${min} 到 ${max} 之间。`)
    return undefined
  }
  return parsed
}

function validateCronField(field: string, min: number, max: number, names?: Map<string, number>) {
  if (!field) return false
  return field.split(',').every((part) => validateCronRange(part, min, max, names))
}

function cronValue(value: string, names?: Map<string, number>) {
  const named = names?.get(value.toLowerCase())
  if (named !== undefined) return named
  if (!/^\d+$/.test(value)) return null
  return Number(value)
}

function validateCronRange(expression: string, min: number, max: number, names?: Map<string, number>) {
  const [range, step, extraStep] = expression.split('/')
  if (!range || extraStep !== undefined) return false
  if (step !== undefined && (!/^\d+$/.test(step) || Number(step) <= 0)) return false
  const [startText, endText, extraRange] = range.split('-')
  if (!startText || extraRange !== undefined) return false
  if (startText === '*' || startText === '?') {
    return endText === undefined
  }
  const start = cronValue(startText, names)
  if (start === null || start < min || start > max) return false
  if (endText === undefined) return true
  const end = cronValue(endText, names)
  return end !== null && end >= start && end <= max
}

function validateEveryDescriptor(expression: string) {
  const duration = expression.slice('@every '.length).trim()
  if (!duration) return false
  const matches = duration.matchAll(/(\d+(?:\.\d+)?)(ns|us|µs|ms|s|m|h)/g)
  let consumed = ''
  let total = 0
  for (const match of matches) {
    consumed += match[0]
    total += Number(match[1])
  }
  return consumed === duration && total > 0
}

function validateCronExpression(expression: string) {
  if (!expression) return true
  const descriptor = expression.toLowerCase()
  if (['@yearly', '@annually', '@monthly', '@weekly', '@daily', '@midnight', '@hourly'].includes(descriptor)) {
    return true
  }
  if (descriptor.startsWith('@every ')) return validateEveryDescriptor(descriptor)
  const fields = expression.split(/\s+/)
  if (fields.length !== 5) return false
  return (
    validateCronField(fields[0], 0, 59) &&
    validateCronField(fields[1], 0, 23) &&
    validateCronField(fields[2], 1, 31) &&
    validateCronField(fields[3], 1, 12, monthNames) &&
    validateCronField(fields[4], 0, 6, dayNames)
  )
}

export function projectConfigForm(form: ConfigForm, mode: ConfigValidationMode): ConfigProjection {
  const errors: ConfigValidationErrors = {}
  const targetName = form.name.trim()
  const cronExpr = form.cron_expr.trim()
  const userAgent = form.user_agent.trim()
  const urls = parseURLs(form.urls, mode, errors)
  const headers = parseHeaders(form.request_headers, userAgent, errors)

  validateName(targetName, errors)

  const threads = integerValue(form.threads)
  if (threads === null || threads < 1 || threads > limits.threads) {
    addError(errors, 'threads', `线程数量必须在 1 到 ${limits.threads} 之间。`)
  }

  const limitSpeed = finiteValue(form.limit_speed)
  if (limitSpeed !== null && (limitSpeed < 0 || limitSpeed > limits.limitSpeed)) {
    addError(errors, 'limit_speed', `限速必须在 0 到 ${limits.limitSpeed} MB/s 之间。`)
  }

  if (form.url_strategy && !['random', 'round_robin'].includes(form.url_strategy)) {
    addError(errors, 'url_strategy', 'URL 策略只能是随机均衡或轮询顺序。')
  }

  if (userAgent && /[\r\n]/.test(userAgent)) {
    addError(errors, 'user_agent', 'User-Agent 不能包含换行。')
  }

  const interval = optionalInteger(form.interval, 'interval', '间隔', 1, limits.interval, errors)
  if (cronExpr && interval !== undefined) {
    addError(errors, 'cron_expr', 'Cron 表达式和间隔调度不能同时设置。')
    addError(errors, 'interval', 'Cron 表达式和间隔调度不能同时设置。')
  }
  if (cronExpr && !validateCronExpression(cronExpr)) {
    addError(errors, 'cron_expr', '无效的Cron表达式')
  }

  const payload: ConfigPayload = {
    name: targetName,
    config_name: targetName,
    // 显式空 URL 列表表示用户刻意保存空链接，保存模式不能退回默认 URL。
    urls,
    url_strategy: form.url_strategy || 'random',
    auto_remove_failed_url: form.auto_remove_failed_url,
    auto_start: form.auto_start,
  }

  if (threads !== null) payload.threads = threads
  if (limitSpeed !== null) payload.limit_speed = limitSpeed

  const trafficLimit = optionalInteger(form.traffic_limit, 'traffic_limit', '总流量', 1, limits.trafficLimit, errors)
  const duration = optionalInteger(form.duration, 'duration', '时长', 1, limits.duration, errors)
  const count = optionalInteger(form.count, 'count', '下载次数', 1, limits.count, errors)
  const urlSwitchInterval = optionalNumber(
    form.url_switch_interval,
    'url_switch_interval',
    'URL 切换时限',
    limits.minSwitchSeconds,
    limits.seconds,
    errors,
  )
  const threadStartDelay = optionalNumber(
    form.thread_start_delay,
    'thread_start_delay',
    '线程启动间隔',
    0,
    limits.seconds,
    errors,
  )

  if (trafficLimit !== undefined) payload.traffic_limit = trafficLimit
  if (duration !== undefined) payload.duration = duration
  if (count !== undefined) payload.count = count
  if (interval !== undefined) payload.interval = interval
  if (urlSwitchInterval !== undefined) payload.url_switch_interval = urlSwitchInterval
  if (threadStartDelay !== undefined) payload.thread_start_delay = threadStartDelay
  if (cronExpr) payload.cron_expr = cronExpr
  if (userAgent) payload.user_agent = userAgent
  if (headers) payload.request_headers = headers

  if (hasErrors(errors)) {
    return { ok: false, payload: null, targetName, errors }
  }
  return { ok: true, payload, targetName, errors }
}

export function projectConfigPayload(name: string, config: ConfigPayload, mode: ConfigValidationMode): ConfigProjection {
  const form: ConfigForm = {
    name,
    urls: Array.isArray(config.urls) ? config.urls.join('\n') : '',
    threads: config.threads,
    url_strategy: config.url_strategy || 'random',
    limit_speed: config.limit_speed,
    traffic_limit: config.traffic_limit,
    duration: config.duration,
    count: config.count,
    cron_expr: config.cron_expr || '',
    interval: config.interval,
    auto_remove_failed_url: Boolean(config.auto_remove_failed_url),
    auto_start: Boolean(config.auto_start),
    user_agent: config.user_agent || '',
    request_headers: config.request_headers
      ? Object.entries(config.request_headers).map(([headerName, value]) => `${headerName}: ${value}`).join('\n')
      : '',
    url_switch_interval: config.url_switch_interval,
    thread_start_delay: config.thread_start_delay,
  }
  return projectConfigForm(form, mode)
}

export function snapshotConfigForm(form: ConfigForm): string {
  return JSON.stringify({
    name: form.name.trim(),
    urls: form.urls
      .split(/\r?\n/)
      .map((url) => url.trim())
      .filter(Boolean),
    threads: form.threads ?? null,
    url_strategy: form.url_strategy || 'random',
    limit_speed: form.limit_speed ?? null,
    traffic_limit: form.traffic_limit ?? null,
    duration: form.duration ?? null,
    count: form.count ?? null,
    cron_expr: form.cron_expr.trim(),
    interval: form.interval ?? null,
    auto_remove_failed_url: form.auto_remove_failed_url,
    auto_start: form.auto_start,
    user_agent: form.user_agent.trim(),
    request_headers: form.request_headers
      .split(/\r?\n/)
      .map((line) => line.trim())
      .filter(Boolean),
    url_switch_interval: form.url_switch_interval ?? null,
    thread_start_delay: form.thread_start_delay ?? null,
  })
}

export function activeRuntimeNames(plans: PlanSummary[], status: StatusPayload, names: string[]) {
  const requested = new Set(names.map((name) => name.trim()).filter(Boolean).map(normalizeConfigName))
  const active = new Set<string>()
  for (const plan of plans) {
    if (requested.has(normalizeConfigName(plan.name)) && (plan.running || plan.scheduler_running)) {
      active.add(plan.name)
    }
  }
  if (status.config && requested.has(normalizeConfigName(status.config)) && status.state !== 'stopped') {
    active.add(status.config)
  }
  return Array.from(active)
}

export function scheduledPlans(plans: PlanSummary[]) {
  return plans.filter((plan) => plan.scheduler_running)
}
