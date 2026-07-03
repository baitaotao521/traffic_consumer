import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { toast } from 'vue-sonner'

import {
  deleteConfig,
  getConfig,
  getEvents,
  getLogs,
  getPlanDetail,
  getScheduler,
  getStatus,
  listConfigs,
  previewCron,
  saveConfig,
  startRuntime,
  startTempTest,
  stopPlan,
  stopRuntime,
  stopScheduler,
} from '@/api'
import type {
  BadgeVariant,
  ConfigForm,
  ConfigPayload,
  PlanDetailPayload,
  PlanSummary,
  RuntimeEvent,
  SchedulerPayload,
  StatusPayload,
  ThreadCounts,
  ThreadRow,
} from '@/types'

const emptyStatus: StatusPayload = {
  running: false,
  state: 'stopped',
  total_bytes: '0 B',
  speed: '0 B/s',
  download_count: 0,
  thread_count: 0,
  thread_status: {},
  url_usage_stats: [],
}

const emptyScheduler: SchedulerPayload = {
  next_run_time: '',
  job_details: '',
  plans: [],
}

export function useDashboard() {
  const configs = ref<string[]>([])
  const selectedConfigName = ref('')
  const selectedConfigDetail = ref<ConfigPayload | null>(null)
  const status = ref<StatusPayload>(emptyStatus)
  const scheduler = ref<SchedulerPayload>(emptyScheduler)
  const logs = ref<Array<{ seq: number; timestamp: string; level: string; message: string }>>([])
  const logEnabled = ref(false)
  const logCursor = ref(0)
  const eventCursor = ref(0)
  const editorOpen = ref(false)
  const planDialogOpen = ref(false)
  const selectedPlanDetail = ref<PlanDetailPayload | null>(null)
  const speedSamples = ref<{ label: string; value: number }[]>([])
  const cronPreview = ref<string[]>([])
  const confirmDialogOpen = ref(false)
  const confirmDialogTitle = ref('')
  const confirmDialogDescription = ref('')
  const confirmDialogActionLabel = ref('确认')
  const confirmDialogVariant = ref<'default' | 'destructive'>('default')
  const recentAlertKeys = new Map<string, number>()
  let pendingConfirmAction: null | (() => Promise<void>) = null

  const form = reactive<ConfigForm>({
    name: '',
    urls: '',
    threads: '',
    url_strategy: 'random',
    limit_speed: '',
    traffic_limit: '',
    duration: '',
    count: '',
    cron_expr: '',
    interval: '',
    auto_remove_failed_url: false,
    auto_start: false,
    user_agent: '',
    request_headers: '',
    url_switch_interval: '',
    thread_start_delay: '',
  })

  const statusLabel = computed(() => {
    if (status.value.state === 'running') return '运行中'
    if (status.value.state === 'scheduled') return '等待调度'
    return '已停止'
  })

  const statusVariant = computed<BadgeVariant>(() => {
    if (status.value.state === 'running') return 'default'
    if (status.value.state === 'scheduled') return 'secondary'
    return 'outline'
  })

  const currentConfig = computed(() => status.value.config || selectedConfigName.value || '未配置')
  const plans = computed(() => scheduler.value.plans || [])
  const urlStats = computed(() => status.value.url_usage_stats || [])

  const threadRows = computed<ThreadRow[]>(() => {
    const map = status.value.thread_status || {}
    const total = Number(status.value.thread_count || Object.keys(map).length || 0)
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
  })

  const threadCounts = computed<ThreadCounts>(() => {
    const active = threadRows.value.filter((row) => row.state === 'active').length
    const errored = threadRows.value.filter((row) => row.state === 'error').length
    const idle = Math.max(threadRows.value.length - active - errored, 0)
    return { total: threadRows.value.length, active, idle, errored }
  })

  const speedValues = computed(() => speedSamples.value.map((sample) => sample.value))
  const speedLabels = computed(() => speedSamples.value.map((sample) => sample.label))
  const urlChartValues = computed(() => urlStats.value.map((item) => item.count))
  const urlChartLabels = computed(() => urlStats.value.map((item, index) => item.url || `链接 ${index + 1}`))

  function pushAlert(message: string, title = '状态更新', variant: 'default' | 'destructive' = 'default', detail?: string) {
    const now = Date.now()
    const dedupeKey = [variant, title, message, detail || ''].join('::')
    const dedupeWindow = variant === 'destructive' ? 6000 : 4000
    const lastShownAt = recentAlertKeys.get(dedupeKey)

    for (const [key, timestamp] of recentAlertKeys.entries()) {
      if (now - timestamp > 15000) {
        recentAlertKeys.delete(key)
      }
    }

    if (lastShownAt && now - lastShownAt < dedupeWindow) {
      return
    }

    recentAlertKeys.set(dedupeKey, now)
    const description = [title, detail].filter((item) => item && item !== '状态更新').join(' · ') || undefined
    if (variant === 'destructive') {
      toast.error(message, { description, duration: 8000 })
      return
    }
    toast(message, { description, duration: 5000 })
  }

  function openConfirmation(options: {
    title: string
    description: string
    actionLabel: string
    variant?: 'default' | 'destructive'
    action: () => Promise<void>
  }) {
    confirmDialogTitle.value = options.title
    confirmDialogDescription.value = options.description
    confirmDialogActionLabel.value = options.actionLabel
    confirmDialogVariant.value = options.variant || 'default'
    pendingConfirmAction = options.action
    confirmDialogOpen.value = true
  }

  function cancelPendingAction() {
    pendingConfirmAction = null
    confirmDialogOpen.value = false
  }

  async function confirmPendingAction() {
    const action = pendingConfirmAction
    pendingConfirmAction = null
    confirmDialogOpen.value = false
    if (!action) return
    await runAction(action)
  }

  function clearLogs() {
    logs.value = []
  }

  function parseInteger(value: string): number | null {
    if (!value.trim()) return null
    const parsed = Number.parseInt(value, 10)
    return Number.isFinite(parsed) ? parsed : null
  }

  function parseFloatOrNull(value: string): number | null {
    if (!value.trim()) return null
    const parsed = Number.parseFloat(value)
    return Number.isFinite(parsed) ? parsed : null
  }

  function normalizeHeaders(value: string): Record<string, string> | null {
    const result: Record<string, string> = {}
    value.split(/\r?\n/).forEach((line) => {
      const index = line.indexOf(':')
      if (index <= 0) return
      const name = line.slice(0, index).trim()
      const headerValue = line.slice(index + 1).trim()
      if (name && headerValue) result[name] = headerValue
    })
    return Object.keys(result).length ? result : null
  }

  function headersToText(headers: ConfigPayload['request_headers']): string {
    if (!headers) return ''
    return Object.entries(headers)
      .map(([name, value]) => `${name}: ${value}`)
      .join('\n')
  }

  function formFromConfig(name = '', config: ConfigPayload = { urls: [] }) {
    form.name = name
    form.urls = Array.isArray(config.urls) ? config.urls.join('\n') : ''
    form.threads = config.threads ? String(config.threads) : ''
    form.url_strategy = config.url_strategy || 'random'
    form.limit_speed = config.limit_speed ? String(config.limit_speed) : ''
    form.traffic_limit = config.traffic_limit ? String(config.traffic_limit) : ''
    form.duration = config.duration ? String(config.duration) : ''
    form.count = config.count ? String(config.count) : ''
    form.cron_expr = config.cron_expr || ''
    form.interval = config.interval ? String(config.interval) : ''
    form.auto_remove_failed_url = Boolean(config.auto_remove_failed_url)
    form.auto_start = Boolean(config.auto_start)
    form.user_agent = config.user_agent || ''
    form.request_headers = headersToText(config.request_headers)
    form.url_switch_interval = config.url_switch_interval ? String(config.url_switch_interval) : ''
    form.thread_start_delay = config.thread_start_delay ? String(config.thread_start_delay) : ''
  }

  function configFromForm(): ConfigPayload {
    const name = form.name.trim()
    return {
      name,
      config_name: name,
      urls: form.urls.split(/\r?\n/).map((url) => url.trim()).filter(Boolean),
      url_strategy: form.url_strategy || 'random',
      threads: parseInteger(form.threads),
      limit_speed: parseFloatOrNull(form.limit_speed),
      traffic_limit: parseInteger(form.traffic_limit),
      duration: parseInteger(form.duration),
      count: parseInteger(form.count),
      cron_expr: form.cron_expr.trim() || null,
      interval: parseInteger(form.interval),
      auto_remove_failed_url: form.auto_remove_failed_url,
      auto_start: form.auto_start,
      user_agent: form.user_agent.trim() || null,
      request_headers: normalizeHeaders(form.request_headers),
      url_switch_interval: parseFloatOrNull(form.url_switch_interval),
      thread_start_delay: parseFloatOrNull(form.thread_start_delay),
    }
  }

  function parseSpeed(value?: string): number {
    if (!value) return 0
    const match = value.match(/(\d+(?:\.\d+)?)\s*MB\/s/i)
    return match ? Number.parseFloat(match[1]) : 0
  }

  function applyStatus(nextStatus: StatusPayload) {
    status.value = {
      ...emptyStatus,
      ...nextStatus,
      thread_status: nextStatus.thread_status || {},
      url_usage_stats: nextStatus.url_usage_stats || [],
    }
    if (nextStatus.message) {
      pushAlert(nextStatus.message)
    }
    speedSamples.value.push({ label: new Date().toLocaleTimeString(), value: parseSpeed(nextStatus.speed) })
    speedSamples.value = speedSamples.value.slice(-30)
  }

  async function refreshConfigs() {
    const data = await listConfigs()
    configs.value = data.configs || []
    if (!configs.value.length) {
      selectedConfigName.value = ''
      selectedConfigDetail.value = null
      formFromConfig()
      return
    }
    if (!selectedConfigName.value || !configs.value.includes(selectedConfigName.value)) {
      selectedConfigName.value = configs.value[0]
    }
    await loadConfigDetail(selectedConfigName.value)
  }

  async function loadConfigDetail(name: string) {
    if (!name) return
    const data = await getConfig(name)
    selectedConfigName.value = data.name
    selectedConfigDetail.value = { ...data.config, name: data.name, config_name: data.name }
  }

  async function pollStatus() {
    try {
      applyStatus(await getStatus())
    } catch (error) {
      pushAlert((error as Error).message, '操作失败', 'destructive')
    }
  }

  async function pollScheduler() {
    try {
      scheduler.value = { ...emptyScheduler, ...(await getScheduler()) }
    } catch (error) {
      pushAlert((error as Error).message, '操作失败', 'destructive')
    }
  }

  async function pollLogs() {
    if (!logEnabled.value) return
    try {
      const data = await getLogs(logCursor.value)
      logCursor.value = Number(data.latest || logCursor.value)
      logs.value = [...logs.value, ...(data.logs || [])].slice(-200)
    } catch (error) {
      pushAlert((error as Error).message, '日志错误', 'destructive')
    }
  }

  async function pollEvents() {
    try {
      const data = await getEvents(eventCursor.value)
      eventCursor.value = Number(data.latest || eventCursor.value)
      for (const event of data.events || []) {
        await handleRuntimeEvent(event)
      }
    } catch (error) {
      pushAlert((error as Error).message, '操作失败', 'destructive')
    }
  }

  async function handleRuntimeEvent(event: RuntimeEvent) {
    if (event.type === 'history_update') {
      await pollScheduler()
      await pollStatus()
      return
    }
    if (['status_update', 'error', 'download_failure'].includes(event.type) || event.event_type === 'invalid_url') {
      pushAlert(
        event.message || '发生未知事件。',
        event.title || (event.type === 'error' ? '操作失败' : '状态更新'),
        event.type === 'error' || event.variant === 'danger' ? 'destructive' : 'default',
        event.error,
      )
    }
  }

  async function runAction(action: () => Promise<unknown>) {
    try {
      await action()
    } catch (error) {
      pushAlert((error as Error).message, '操作失败', 'destructive')
    }
  }

  function ensureSelectedConfig(): ConfigPayload | null {
    if (!selectedConfigName.value || !selectedConfigDetail.value) {
      pushAlert('请选择运行配置。', '提示')
      return null
    }
    return {
      ...selectedConfigDetail.value,
      name: selectedConfigName.value,
      config_name: selectedConfigName.value,
      urls: selectedConfigDetail.value.urls || [],
    }
  }

  async function startSelected() {
    const payload = ensureSelectedConfig()
    if (!payload) return
    applyStatus(await startRuntime(payload))
    await pollScheduler()
  }

  async function startSelectedTempTest() {
    const payload = ensureSelectedConfig()
    if (!payload) return
    applyStatus(await startTempTest(payload))
    await pollScheduler()
  }

  async function stopSelected() {
    await stopRuntime()
    await pollStatus()
    await pollScheduler()
  }

  async function stopSelectedScheduler() {
    await stopScheduler()
    await pollStatus()
    await pollScheduler()
  }

  function openEditor() {
    if (selectedConfigName.value && selectedConfigDetail.value) {
      formFromConfig(selectedConfigName.value, selectedConfigDetail.value)
    } else {
      formFromConfig()
    }
    editorOpen.value = true
  }

  async function saveEditorConfig() {
    const payload = configFromForm()
    const name = payload.name?.trim() || ''
    if (!name) {
      pushAlert('请填写配置名称。', '提示')
      return
    }
    await saveConfig(name, payload)
    selectedConfigName.value = name
    await refreshConfigs()
    pushAlert(`配置 "${name}" 已保存。`)
    editorOpen.value = false
  }

  async function deleteSelectedConfig(target: string) {
    await deleteConfig(target)
    selectedConfigName.value = ''
    await refreshConfigs()
    await pollScheduler()
    pushAlert(`配置 "${target}" 已删除。`)
  }

  function requestDeleteSelectedConfig() {
    const target = form.name.trim() || selectedConfigName.value
    if (!target) {
      pushAlert('请选择配置。', '提示')
      return
    }
    openConfirmation({
      title: '删除配置',
      description: `配置 "${target}" 会从当前列表中移除。`,
      actionLabel: '删除',
      variant: 'destructive',
      action: () => deleteSelectedConfig(target),
    })
  }

  async function openPlanDetail(plan: PlanSummary) {
    selectedPlanDetail.value = await getPlanDetail(plan.name)
    planDialogOpen.value = true
  }

  async function stopNamedPlan(plan: PlanSummary) {
    await stopPlan(plan.name)
    await pollStatus()
    await pollScheduler()
    pushAlert(`计划 "${plan.name}" 已停止。`)
  }

  function requestStopNamedPlan(plan: PlanSummary) {
    openConfirmation({
      title: '停止计划',
      description: `计划 "${plan.name}" 会立即停止当前执行和后续调度。`,
      actionLabel: '停止',
      variant: 'destructive',
      action: () => stopNamedPlan(plan),
    })
  }

  function resetEditor() {
    formFromConfig()
    cronPreview.value = []
  }

  function formatDateTime(value?: string): string {
    if (!value) return '无'
    const date = new Date(value.includes('T') ? value : value.replace(' ', 'T'))
    if (Number.isNaN(date.getTime())) return value
    return date.toLocaleString()
  }

  function formatPlanSchedule(plan: PlanSummary): string {
    if (plan.cron_expr) return `Cron ${plan.cron_expr}`
    if (plan.interval) return `每 ${plan.interval} 分钟`
    return '手动任务'
  }

  function planState(plan: PlanSummary): { label: string; variant: BadgeVariant } {
    if (plan.running) return { label: '运行中', variant: 'default' }
    if (plan.scheduler_running) return { label: '计划中', variant: 'secondary' }
    return { label: '已停止', variant: 'outline' }
  }

  function logLevelClass(level: string) {
    if (level === 'error') return 'text-destructive'
    if (level === 'warning') return 'text-primary'
    if (level === 'success') return ''
    return 'text-muted-foreground'
  }

  let statusTimer = 0
  let schedulerTimer = 0
  let logTimer = 0
  let eventTimer = 0
  let cronTimer = 0

  watch(selectedConfigName, async (name) => {
    if (name) {
      await runAction(() => loadConfigDetail(name))
    }
  })

  watch(
    () => form.cron_expr,
    (value) => {
      window.clearTimeout(cronTimer)
      cronTimer = window.setTimeout(async () => {
        if (!value.trim()) {
          cronPreview.value = []
          return
        }
        await runAction(async () => {
          cronPreview.value = await previewCron(value.trim())
        })
      }, 500)
    },
  )

  onMounted(async () => {
    await runAction(refreshConfigs)
    await Promise.all([pollStatus(), pollScheduler(), pollEvents()])
    statusTimer = window.setInterval(pollStatus, 1000)
    schedulerTimer = window.setInterval(pollScheduler, 2000)
    logTimer = window.setInterval(pollLogs, 1000)
    eventTimer = window.setInterval(pollEvents, 1000)
  })

  onBeforeUnmount(() => {
    window.clearInterval(statusTimer)
    window.clearInterval(schedulerTimer)
    window.clearInterval(logTimer)
    window.clearInterval(eventTimer)
    window.clearTimeout(cronTimer)
  })

  return {
    cancelPendingAction,
    clearLogs,
    confirmDialogActionLabel,
    confirmDialogDescription,
    confirmDialogOpen,
    confirmDialogTitle,
    confirmDialogVariant,
    confirmPendingAction,
    configs,
    cronPreview,
    currentConfig,
    editorOpen,
    form,
    formatDateTime,
    formatPlanSchedule,
    logEnabled,
    logLevelClass,
    logs,
    openEditor,
    openPlanDetail,
    planDialogOpen,
    planState,
    plans,
    pollStatus,
    refreshConfigs,
    requestDeleteSelectedConfig,
    requestStopNamedPlan,
    resetEditor,
    saveEditorConfig,
    scheduler,
    selectedConfigName,
    selectedPlanDetail,
    speedLabels,
    speedValues,
    startSelected,
    startSelectedTempTest,
    status,
    statusLabel,
    statusVariant,
    stopNamedPlan,
    stopSelected,
    stopSelectedScheduler,
    threadCounts,
    threadRows,
    urlChartLabels,
    urlChartValues,
    urlStats,
    runAction,
  }
}
