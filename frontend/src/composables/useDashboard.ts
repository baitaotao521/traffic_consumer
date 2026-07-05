import { useMutation } from '@tanstack/vue-query'
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { toast } from 'vue-sonner'

import { apiClient, ApiError } from '@/api/client'
import {
  useActivityQuery,
  useConfigCommandMutations,
  useDashboardCache,
  useDashboardQuery,
  useRuntimeCommandMutations,
} from '@/api/queries'
import {
  applyConfigToForm,
  configNames,
  configWithName,
  currentConfigName,
  emptyDashboard,
  formatDateTime,
  formatPlanSchedule,
  logLevelOptions,
  logLevelClass,
  logSourceLabel,
  logSourceOptions,
  logSourceVariant,
  normalizeDashboard,
  normalizePlanDetail,
  parseSpeed,
  planState,
  projectActivityEvent,
  statusLabel,
  statusState,
  statusVariant,
  threadCounts as makeThreadCounts,
  threadRows as makeThreadRows,
  normalizeFilterSelection,
} from '@/domain/dashboard'
import {
  activeRuntimeNames,
  configNameExists,
  firstConfigError,
  projectConfigForm,
  projectConfigPayload,
  sameConfigName,
  scheduledPlans,
  snapshotConfigForm,
} from '@/domain/configEditor'
import type { ConfigEditorMode, ConfigValidationErrors, ConfigValidationMode } from '@/domain/configEditor'
import type { ConfigForm, ConfigPayload, HistoryRecord, LogEntry, PlanDetailView, PlanSummary } from '@/types'

type AlertVariant = 'default' | 'success' | 'info' | 'warning' | 'destructive'
type ConfirmPendingKey = 'deleteConfig' | 'stopPlan' | 'saveConfig'

type AlertOptions = {
  title: string
  description?: string
  detail?: string
  variant?: AlertVariant
}

type EditorSaveRequest = {
  sourceName: string
  targetName: string
  mode: ConfigEditorMode
  payload: ConfigPayload
}

export function useDashboard() {
  const dashboardCache = useDashboardCache()
  const dashboardQuery = useDashboardQuery()

  const selectedConfigName = ref('')
  const selectedConfigDetail = ref<ConfigPayload | null>(null)
  const logs = ref<LogEntry[]>([])
  const logEnabled = ref(false)
  const selectedLogSources = ref(logSourceOptions.map((option) => option.value))
  const selectedLogLevels = ref(logLevelOptions.map((option) => option.value))
  const lastRunByConfig = reactive<Record<string, HistoryRecord | undefined>>({})
  const logCursor = ref(0)
  const eventCursor = ref(0)
  const editorOpen = ref(false)
  const planDialogOpen = ref(false)
  const selectedPlanDetail = ref<PlanDetailView | null>(null)
  const speedSamples = ref<{ label: string; value: number }[]>([])
  const cronPreview = ref<string[]>([])
  const confirmDialogOpen = ref(false)
  const confirmDialogTitle = ref('')
  const confirmDialogDescription = ref('')
  const confirmDialogActionLabel = ref('确认')
  const confirmDialogVariant = ref<'default' | 'destructive'>('default')
  const confirmDialogPendingKey = ref<null | ConfirmPendingKey>(null)
  const editorMode = ref<ConfigEditorMode>('create')
  const editorOriginalName = ref('')
  const editorInitialSnapshot = ref('')
  const editorValidationErrors = ref<ConfigValidationErrors>({})
  const editorValidationMode = ref<ConfigValidationMode>('save')
  const eventStreamReady = ref(false)
  const logStreamReady = ref(false)
  const refreshingDashboard = ref(false)
  const recentAlertKeys = new Map<string, number>()
  let pendingConfirmAction: null | (() => Promise<void>) = null

  const form = reactive<ConfigForm>({
    name: '',
    urls: '',
    threads: undefined,
    url_strategy: 'random',
    limit_speed: undefined,
    traffic_limit: undefined,
    duration: undefined,
    count: undefined,
    cron_expr: '',
    interval: undefined,
    auto_remove_failed_url: false,
    auto_start: false,
    user_agent: '',
    request_headers: '',
    url_switch_interval: undefined,
    thread_start_delay: undefined,
  })
  editorInitialSnapshot.value = snapshotConfigForm(form)

  const dashboard = computed(() => normalizeDashboard(dashboardQuery.data.value?.dashboard || emptyDashboard))
  const configs = computed(() => configNames(dashboard.value))
  const status = computed(() => dashboard.value.status)
  const scheduler = computed(() => dashboard.value.scheduler)
  const plans = computed(() => dashboard.value.plans || scheduler.value.plans || [])
  const activeConfigCount = computed(() => plans.value.filter((plan) => plan.running || plan.scheduler_running).length)
  const urlStats = computed(() => status.value.url_usage_stats || [])
  const currentConfig = computed(() => currentConfigName(status.value, selectedConfigName.value))
  const statusLabelValue = computed(() => statusLabel(status.value))
  const statusVariantValue = computed(() => statusVariant(status.value))
  const isDashboardLoading = computed(() => dashboardQuery.isPending.value && !dashboardQuery.data.value)
  const threadRows = computed(() => makeThreadRows(status.value))
  const threadCounts = computed(() => makeThreadCounts(threadRows.value))
  const speedValues = computed(() => speedSamples.value.map((sample) => sample.value))
  const speedLabels = computed(() => speedSamples.value.map((sample) => sample.label))
  const urlChartValues = computed(() => urlStats.value.map((item) => item.count))
  const urlChartLabels = computed(() => urlStats.value.map((item, index) => item.url || `链接 ${index + 1}`))

  const activityQuery = useActivityQuery({
    afterLog: logCursor,
    afterEvent: eventCursor,
    includeLogs: logEnabled,
    logLevels: selectedLogLevels,
    logSources: selectedLogSources,
  })
  const configMutations = useConfigCommandMutations()
  const runtimeMutations = useRuntimeCommandMutations()
  const cronPreviewMutation = useMutation({
    mutationFn: (expression: string) => apiClient.previewCron(expression),
  })
  const isSavingConfig = computed(() => configMutations.saveConfig.isPending.value)
  const isDeletingConfig = computed(() => configMutations.deleteConfig.isPending.value)
  const isStartingRuntime = computed(() => runtimeMutations.startRuntime.isPending.value)
  const isStartingTempTest = computed(() => runtimeMutations.startTempTest.isPending.value)
  const isStoppingRuntime = computed(() => runtimeMutations.stopRuntime.isPending.value)
  const isStoppingScheduler = computed(() => runtimeMutations.stopScheduler.isPending.value)
  const isStoppingPlan = computed(() => runtimeMutations.stopPlan.isPending.value)
  const editorLastRun = computed(() => {
    const name = form.name.trim() || selectedConfigName.value
    if (!name) return null
    return lastRunByConfig[name] || plans.value.find((plan) => plan.name === name)?.last_run || null
  })
  const isConfirmingAction = computed(() => {
    if (confirmDialogPendingKey.value === 'deleteConfig') return isDeletingConfig.value
    if (confirmDialogPendingKey.value === 'stopPlan') return isStoppingPlan.value
    if (confirmDialogPendingKey.value === 'saveConfig') return isSavingConfig.value
    return false
  })
  const isEditorDirty = computed(() => editorOpen.value && snapshotConfigForm(form) !== editorInitialSnapshot.value)

  function mergeAlertLines(...lines: Array<string | undefined>) {
    return lines
      .map((line) => line?.trim())
      .filter((line, index, items): line is string => Boolean(line) && items.indexOf(line) === index)
      .join('\n') || undefined
  }

  function pushAlert(options: AlertOptions) {
    const variant = options.variant || 'default'
    const description = mergeAlertLines(options.description, options.detail)
    const now = Date.now()
    const dedupeKey = [variant, options.title, description || ''].join('::')
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
    const toastOptions = { description, duration: variant === 'destructive' ? 8000 : 5000 }

    if (variant === 'destructive') {
      toast.error(options.title, toastOptions)
      return
    }
    if (variant === 'warning') {
      toast.warning(options.title, toastOptions)
      return
    }
    if (variant === 'success') {
      toast.success(options.title, toastOptions)
      return
    }
    if (variant === 'info') {
      toast.info(options.title, toastOptions)
      return
    }
    toast(options.title, toastOptions)
  }

  function pushError(error: unknown, title = '操作失败') {
    if (error instanceof ApiError) {
      pushAlert({
        title,
        description: error.message,
        detail: error.details?.[0]?.message,
        variant: 'destructive',
      })
      return
    }
    pushAlert({
      title,
      description: error instanceof Error ? error.message : '请求失败',
      variant: 'destructive',
    })
  }

  function pushCommandMessage(response: { message?: string }) {
    if (response.message) {
      pushAlert({ title: response.message, variant: 'success' })
    }
  }

  async function runRuntimeMutation<T extends { message?: string }>(mutation: () => Promise<T>) {
    // 运行态操作的用户提示统一使用命令响应，activity 里的 status_update 只保留状态同步职责。
    const response = await mutation()
    pushCommandMessage(response)
    return response
  }

  function clearEditorErrors() {
    editorValidationErrors.value = {}
  }

  function setEditorErrors(errors: ConfigValidationErrors) {
    editorValidationErrors.value = errors
  }

  function validateEditorForm(mode: ConfigValidationMode) {
    editorValidationMode.value = mode
    const projection = projectConfigForm(form, mode)
    setEditorErrors(projection.errors)
    if (!projection.ok) {
      pushAlert({ title: firstConfigError(projection.errors) || '配置内容不完整。', variant: 'warning' })
    }
    return projection
  }

  function markEditorClean() {
    editorInitialSnapshot.value = snapshotConfigForm(form)
    clearEditorErrors()
  }

  function setEditorContext(mode: ConfigEditorMode, originalName = '') {
    editorMode.value = mode
    editorOriginalName.value = originalName
    markEditorClean()
  }

  function closeEditorNow() {
    editorOpen.value = false
    clearEditorErrors()
  }

  function uniqueNames(names: string[]) {
    return names.filter((name, index, items) => Boolean(name) && items.indexOf(name) === index)
  }

  function affectedRuntimeNames(targetName: string) {
    const sourceName = editorMode.value === 'edit' ? editorOriginalName.value : targetName
    return activeRuntimeNames(plans.value, status.value, uniqueNames([sourceName, targetName]))
  }

  function saveNameConflict(targetName: string) {
    const exists = configNameExists(configs.value, targetName)
    if (editorMode.value === 'create') return exists
    return !sameConfigName(targetName, editorOriginalName.value) && exists
  }

  function saveConfirmationDescription(stoppingNames: string[]) {
    return stoppingNames.length ? `保存前停止：${stoppingNames.join('、')}。` : ''
  }

  function openConfirmation(options: {
    title: string
    description: string
    actionLabel: string
    variant?: 'default' | 'destructive'
    pendingKey?: ConfirmPendingKey
    action: () => Promise<void>
  }) {
    confirmDialogTitle.value = options.title
    confirmDialogDescription.value = options.description
    confirmDialogActionLabel.value = options.actionLabel
    confirmDialogVariant.value = options.variant || 'default'
    confirmDialogPendingKey.value = options.pendingKey || null
    pendingConfirmAction = options.action
    confirmDialogOpen.value = true
  }

  function cancelPendingAction() {
    if (isConfirmingAction.value) return
    pendingConfirmAction = null
    confirmDialogPendingKey.value = null
    confirmDialogOpen.value = false
  }

  async function confirmPendingAction() {
    const action = pendingConfirmAction
    if (!action) return
    try {
      await action()
      confirmDialogOpen.value = false
      pendingConfirmAction = null
      confirmDialogPendingKey.value = null
    } catch (error) {
      // 确认后的保存/删除可能因为后端状态变化失败；保留动作让用户可以重试或取消，
      // 否则弹窗还开着但按钮已经没有可执行动作，会表现得像卡死。
      pushError(error)
    }
  }

  function clearLogs() {
    logs.value = []
  }

  function setLogSources(values: string[]) {
    // 空选择会让后端回退到“不过滤”，用户看到的反而是全量日志，所以这里保留上一组有效来源。
    const nextValues = normalizeFilterSelection(logSourceOptions, values, selectedLogSources.value)
    if (nextValues.join(',') !== selectedLogSources.value.join(',')) {
      selectedLogSources.value = nextValues
    }
  }

  function setLogLevels(values: string[]) {
    // 空选择同样会把级别筛选放空成“全部返回”，这里继续沿用当前级别集合。
    const nextValues = normalizeFilterSelection(logLevelOptions, values, selectedLogLevels.value)
    if (nextValues.join(',') !== selectedLogLevels.value.join(',')) {
      selectedLogLevels.value = nextValues
    }
  }

  async function refreshDashboard() {
    if (refreshingDashboard.value) return
    refreshingDashboard.value = true
    try {
      await dashboardQuery.refetch()
    } finally {
      refreshingDashboard.value = false
    }
  }

  async function loadConfigDetail(name: string) {
    if (!name) {
      selectedConfigDetail.value = null
      return
    }
    const data = await apiClient.config(name)
    selectedConfigName.value = data.item.name
    selectedConfigDetail.value = configWithName(data.item.config, data.item.name)
  }

  async function loadConfigPayload(name: string): Promise<ConfigPayload> {
    const data = await apiClient.config(name)
    const payload = configWithName(data.item.config, data.item.name)
    selectedConfigName.value = data.item.name
    selectedConfigDetail.value = payload
    return payload
  }

  async function runAction(action: () => Promise<unknown>) {
    try {
      await action()
    } catch (error) {
      pushError(error)
    }
  }

  function ensureSelectedConfig(): ConfigPayload | null {
    if (!selectedConfigName.value || !selectedConfigDetail.value) {
      pushAlert({ title: '请选择运行配置。', variant: 'warning' })
      return null
    }
    return configWithName(selectedConfigDetail.value, selectedConfigName.value)
  }

  function validateSavedConfigForStart(name: string, payload: ConfigPayload): ConfigPayload | null {
    const projection = projectConfigPayload(name, payload, 'start')
    if (!projection.ok) {
      pushAlert({ title: firstConfigError(projection.errors) || '配置内容不完整。', variant: 'warning' })
      return null
    }
    return projection.payload
  }

  async function startSelected() {
    const payload = ensureSelectedConfig()
    if (!payload) return
    const startPayload = validateSavedConfigForStart(selectedConfigName.value, payload)
    if (!startPayload) return
    await runRuntimeMutation(() => runtimeMutations.startRuntime.mutateAsync(startPayload))
  }

  async function startPlan(plan: PlanSummary) {
    if (plan.running || plan.scheduler_running) {
      pushAlert({ title: `配置 "${plan.name}" 已在运行。`, variant: 'warning' })
      return
    }
    const payload = await loadConfigPayload(plan.name)
    const startPayload = validateSavedConfigForStart(plan.name, payload)
    if (!startPayload) return
    await runRuntimeMutation(() => runtimeMutations.startRuntime.mutateAsync(startPayload))
  }

  async function startEditorTempTest() {
    const projection = validateEditorForm('temp-test')
    if (!projection.ok) return

    if (status.value.state === 'running') {
      pushAlert({ title: '已有下载任务正在运行。', variant: 'warning' })
      return
    }
    const activePlan = plans.value.find((item) => item.running)
    if (activePlan) {
      pushAlert({ title: `配置 "${activePlan.name}" 正在运行。`, variant: 'warning' })
      return
    }
    const scheduled = scheduledPlans(plans.value)
    if (scheduled.length > 1) {
      pushAlert({ title: '存在多个等待调度的计划。', variant: 'warning' })
      return
    }
    if (scheduled.length === 1 && scheduled[0].name !== projection.targetName) {
      pushAlert({ title: `等待调度的计划为 "${scheduled[0].name}"。`, variant: 'warning' })
      return
    }
    await runRuntimeMutation(() => runtimeMutations.startTempTest.mutateAsync(projection.payload))
  }

  async function stopSelected() {
    await runRuntimeMutation(() => runtimeMutations.stopRuntime.mutateAsync(undefined))
  }

  async function stopSelectedScheduler() {
    await runRuntimeMutation(() => runtimeMutations.stopScheduler.mutateAsync(undefined))
  }

  function openCreateConfig() {
    // 新建配置始终从空白表单开始，避免把当前选中配置误保存成副本。
    resetEditor()
    setEditorContext('create')
    editorOpen.value = true
  }

  function openEditor() {
    // 编辑配置沿用当前选中配置；没有配置时仍提供空白表单作为兜底入口。
    if (selectedConfigName.value && selectedConfigDetail.value) {
      applyConfigToForm(form, selectedConfigName.value, selectedConfigDetail.value)
      setEditorContext('edit', selectedConfigName.value)
    } else {
      applyConfigToForm(form)
      setEditorContext('create')
    }
    editorOpen.value = true
  }

  async function openPlanEditor(plan: PlanSummary) {
    const payload = await loadConfigPayload(plan.name)
    applyConfigToForm(form, plan.name, payload)
    setEditorContext('edit', plan.name)
    editorOpen.value = true
  }

  function buildEditorSaveRequest(payload: ConfigPayload, targetName: string): EditorSaveRequest {
    const mode = editorMode.value
    const sourceName = mode === 'edit' && editorOriginalName.value ? editorOriginalName.value : targetName
    return { sourceName, targetName, mode, payload }
  }

  async function submitEditorSave(request: EditorSaveRequest) {
    const response = await configMutations.saveConfig.mutateAsync({
      name: request.sourceName,
      config: request.payload,
      options: {
        mode: request.mode,
      },
    })
    const savedName = response.item?.name || request.targetName
    selectedConfigName.value = savedName
    selectedConfigDetail.value = response.item
      ? configWithName(response.item.config, savedName)
      : configWithName(request.payload, savedName)
    applyConfigToForm(form, savedName, selectedConfigDetail.value)
    setEditorContext('edit', savedName)
    pushCommandMessage(response)
    closeEditorNow()
  }

  async function saveEditorConfig() {
    const projection = validateEditorForm('save')
    if (!projection.ok) return

    if (saveNameConflict(projection.targetName)) {
      pushAlert({ title: `配置 "${projection.targetName}" 已存在。`, variant: 'warning' })
      return
    }

    const stoppingNames = affectedRuntimeNames(projection.targetName)
    const request = buildEditorSaveRequest(projection.payload, projection.targetName)
    if (stoppingNames.length) {
      openConfirmation({
        title: '确认保存配置',
        description: saveConfirmationDescription(stoppingNames),
        actionLabel: '保存',
        pendingKey: 'saveConfig',
        action: () => submitEditorSave(request),
      })
      return
    }

    await submitEditorSave(request)
  }

  async function deleteSelectedConfig(target: string) {
    const response = await configMutations.deleteConfig.mutateAsync(target)
    selectedConfigName.value = response.dashboard.configs?.[0]?.name || ''
    selectedConfigDetail.value = null
    pushCommandMessage(response)
  }

  function requestDeleteSelectedConfig(targetName?: string) {
    const target = targetName || selectedConfigName.value
    if (!target) {
      pushAlert({ title: '请选择配置。', variant: 'warning' })
      return
    }
    const stoppingNames = activeRuntimeNames(plans.value, status.value, [target])
    openConfirmation({
      title: '删除配置',
      description: stoppingNames.length
        ? `删除前停止：${stoppingNames.join('、')}。`
        : `配置 "${target}" 会从当前列表中移除。`,
      actionLabel: '删除',
      variant: 'destructive',
      pendingKey: 'deleteConfig',
      action: () => deleteSelectedConfig(target),
    })
  }

  async function openPlanDetail(plan: PlanSummary) {
    const response = await apiClient.planDetail(plan.name)
    selectedPlanDetail.value = normalizePlanDetail(response.item)
    planDialogOpen.value = true
  }

  async function stopNamedPlan(plan: PlanSummary) {
    await runRuntimeMutation(() => runtimeMutations.stopPlan.mutateAsync(plan.name))
  }

  function requestStopNamedPlan(plan: PlanSummary) {
    openConfirmation({
      title: '停止计划',
      description: `计划 "${plan.name}" 会立即停止当前执行和后续调度。`,
      actionLabel: '停止',
      variant: 'destructive',
      pendingKey: 'stopPlan',
      action: () => stopNamedPlan(plan),
    })
  }

  function resetEditor() {
    applyConfigToForm(form)
    cronPreview.value = []
    setEditorContext('create')
  }

  function requestResetEditor() {
    if (!isEditorDirty.value) {
      resetEditor()
      return
    }
    openConfirmation({
      title: '清空配置',
      description: '未保存的配置修改会被清空。',
      actionLabel: '清空',
      variant: 'destructive',
      action: async () => resetEditor(),
    })
  }

  function requestCloseEditor() {
    if (!isEditorDirty.value) {
      closeEditorNow()
      return
    }
    openConfirmation({
      title: '放弃更改',
      description: '未保存的配置修改会被丢弃。',
      actionLabel: '关闭',
      variant: 'destructive',
      action: async () => closeEditorNow(),
    })
  }

  let cronTimer = 0

  watch(configs, (names) => {
    if (!names.length) {
      selectedConfigName.value = ''
      selectedConfigDetail.value = null
      if (!editorOpen.value) {
        applyConfigToForm(form)
      }
      return
    }
    if (!selectedConfigName.value || !names.includes(selectedConfigName.value)) {
      selectedConfigName.value = names[0]
    }
  }, { immediate: true })

  watch(plans, (items) => {
    for (const plan of items) {
      if (plan.last_run) {
        lastRunByConfig[plan.name] = plan.last_run
      }
    }
  }, { immediate: true })

  watch(selectedConfigName, async (name) => {
    if (name) {
      await runAction(() => loadConfigDetail(name))
    }
  })

  watch(() => dashboardQuery.error.value, (error) => {
    if (error) {
      pushError(error, '刷新失败')
    }
  })

  watch(() => dashboardQuery.data.value, (response) => {
    const nextStatus = response?.dashboard.status
    if (!nextStatus) return
    speedSamples.value.push({ label: new Date().toLocaleTimeString(), value: parseSpeed(nextStatus.speed) })
    speedSamples.value = speedSamples.value.slice(-30)
  }, { immediate: true })

  watch(() => activityQuery.data.value, async (activity) => {
    if (!activity) return

    const shouldConsumeLogs = logEnabled.value && logStreamReady.value
    const shouldConsumeEvents = eventStreamReady.value

    if (shouldConsumeLogs) {
      logs.value = [...logs.value, ...(activity.logs || [])].slice(-200)
    }
    if (typeof activity.log_cursor === 'number') {
      logCursor.value = activity.log_cursor
    }
    if (typeof activity.event_cursor === 'number') {
      eventCursor.value = activity.event_cursor
    }

    // 首次接入活动流只同步最新游标，避免把页面打开前积压的事件和日志全部重放成“新提示”。
    if (logEnabled.value && !logStreamReady.value) {
      logStreamReady.value = true
    }
    if (!eventStreamReady.value) {
      eventStreamReady.value = true
    }
    if (!shouldConsumeEvents) {
      return
    }

    let shouldRefreshDashboard = false
    for (const event of activity.events || []) {
      const projection = projectActivityEvent(event)
      if (projection.refreshDashboard) {
        shouldRefreshDashboard = true
      }
      if (event.type === 'history_update' && event.config && event.history_record) {
        lastRunByConfig[event.config] = event.history_record
      }
      if (projection.toast) {
        pushAlert(projection.toast)
      }
    }

    if (shouldRefreshDashboard) {
      await dashboardCache.invalidateDashboard()
    }
  })

  watch(logEnabled, async (enabled) => {
    if (!enabled) {
      logStreamReady.value = false
      clearLogs()
      return
    }

    // 实时日志重新开启后从当前时刻开始接收，不补拉关闭期间的旧日志。
    clearLogs()
    logCursor.value = 0
    logStreamReady.value = false
    await activityQuery.refetch()
  })

  watch(
    () => [selectedLogSources.value.join(','), selectedLogLevels.value.join(',')],
    async () => {
      clearLogs()
      logCursor.value = 0
      logStreamReady.value = false
      if (logEnabled.value) {
        await activityQuery.refetch()
      }
    },
  )

  watch(
    () => snapshotConfigForm(form),
    () => {
      if (!Object.keys(editorValidationErrors.value).length) return
      const projection = projectConfigForm(form, editorValidationMode.value)
      setEditorErrors(projection.errors)
    },
  )

  watch(
    () => form.cron_expr,
    (value) => {
      window.clearTimeout(cronTimer)
      cronTimer = window.setTimeout(async () => {
        const expression = value.trim()
        if (!expression) {
          cronPreview.value = []
          return
        }
        try {
          const response = await cronPreviewMutation.mutateAsync(expression)
          if (form.cron_expr.trim() === expression) {
            cronPreview.value = response.items || []
            if (editorValidationErrors.value.cron_expr?.length) {
              const projection = projectConfigForm(form, editorValidationMode.value)
              setEditorErrors(projection.errors)
            }
          }
        } catch (error) {
          if (form.cron_expr.trim() === expression) {
            cronPreview.value = []
            setEditorErrors({
              ...editorValidationErrors.value,
              cron_expr: ['无效的Cron表达式'],
            })
          }
          pushError(error, 'Cron 预览失败')
        }
      }, 500)
    },
  )

  onBeforeUnmount(() => {
    window.clearTimeout(cronTimer)
  })

  return {
    activeConfigCount,
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
    editorLastRun,
    editorValidationErrors,
    isConfirmingAction,
    isDashboardLoading,
    isDeletingConfig,
    isRefreshingDashboard: refreshingDashboard,
    isSavingConfig,
    isStartingRuntime,
    isStartingTempTest,
    isStoppingPlan,
    isStoppingRuntime,
    isStoppingScheduler,
    isEditorDirty,
    editorOpen,
    form,
    formatDateTime,
    formatPlanSchedule,
    logEnabled,
    logLevelClass,
    logLevelOptions,
    logSourceLabel,
    logSourceOptions,
    logSourceVariant,
    logs,
    openCreateConfig,
    openEditor,
    openPlanEditor,
    openPlanDetail,
    planDialogOpen,
    planState,
    plans,
    refreshDashboard,
    requestDeleteSelectedConfig,
    requestCloseEditor,
    requestResetEditor,
    requestStopNamedPlan,
    resetEditor,
    saveEditorConfig,
    scheduler,
    selectedLogLevels,
    selectedLogSources,
    selectedConfigName,
    selectedPlanDetail,
    speedLabels,
    speedValues,
    startEditorTempTest,
    startPlan,
    startSelected,
    status,
    statusLabel: statusLabelValue,
    statusState,
    statusVariant: statusVariantValue,
    stopNamedPlan,
    stopSelected,
    stopSelectedScheduler,
    setLogLevels,
    setLogSources,
    threadCounts,
    threadRows,
    urlChartLabels,
    urlChartValues,
    urlStats,
    runAction,
  }
}
