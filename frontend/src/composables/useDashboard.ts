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
  configFromForm,
  configNames,
  configWithName,
  currentConfigName,
  emptyDashboard,
  formatDateTime,
  formatPlanSchedule,
  logLevelClass,
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
} from '@/domain/dashboard'
import type { ConfigForm, ConfigPayload, PlanDetailView, PlanSummary } from '@/types'

export function useDashboard() {
  const dashboardCache = useDashboardCache()
  const dashboardQuery = useDashboardQuery()

  const selectedConfigName = ref('')
  const selectedConfigDetail = ref<ConfigPayload | null>(null)
  const logs = ref<Array<{ seq: number; timestamp: string; level: string; message: string }>>([])
  const logEnabled = ref(false)
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

  const dashboard = computed(() => normalizeDashboard(dashboardQuery.data.value?.dashboard || emptyDashboard))
  const configs = computed(() => configNames(dashboard.value))
  const status = computed(() => dashboard.value.status)
  const scheduler = computed(() => dashboard.value.scheduler)
  const plans = computed(() => dashboard.value.plans || scheduler.value.plans || [])
  const urlStats = computed(() => status.value.url_usage_stats || [])
  const currentConfig = computed(() => currentConfigName(status.value, selectedConfigName.value))
  const statusLabelValue = computed(() => statusLabel(status.value))
  const statusVariantValue = computed(() => statusVariant(status.value))
  const threadRows = computed(() => makeThreadRows(status.value))
  const threadCounts = computed(() => makeThreadCounts(threadRows.value))
  const speedValues = computed(() => speedSamples.value.map((sample) => sample.value))
  const speedLabels = computed(() => speedSamples.value.map((sample) => sample.label))
  const urlChartValues = computed(() => urlStats.value.map((item) => item.count))
  const urlChartLabels = computed(() => urlStats.value.map((item, index) => item.url || `链接 ${index + 1}`))

  const activityQuery = useActivityQuery({ afterLog: logCursor, afterEvent: eventCursor })
  const configMutations = useConfigCommandMutations()
  const runtimeMutations = useRuntimeCommandMutations()
  const cronPreviewMutation = useMutation({
    mutationFn: (expression: string) => apiClient.previewCron(expression),
  })

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

  function pushError(error: unknown, title = '操作失败') {
    if (error instanceof ApiError) {
      pushAlert(error.message, title, 'destructive')
      return
    }
    pushAlert(error instanceof Error ? error.message : '请求失败', title, 'destructive')
  }

  function pushCommandMessage(response: { message?: string }) {
    if (response.message) {
      pushAlert(response.message)
    }
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

  async function refreshDashboard() {
    await dashboardQuery.refetch()
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

  async function runAction(action: () => Promise<unknown>) {
    try {
      await action()
    } catch (error) {
      pushError(error)
    }
  }

  function ensureSelectedConfig(): ConfigPayload | null {
    if (!selectedConfigName.value || !selectedConfigDetail.value) {
      pushAlert('请选择运行配置。', '提示')
      return null
    }
    return configWithName(selectedConfigDetail.value, selectedConfigName.value)
  }

  async function startSelected() {
    const payload = ensureSelectedConfig()
    if (!payload) return
    const response = await runtimeMutations.startRuntime.mutateAsync(payload)
    pushCommandMessage(response)
  }

  async function startSelectedTempTest() {
    const payload = ensureSelectedConfig()
    if (!payload) return
    const response = await runtimeMutations.startTempTest.mutateAsync(payload)
    pushCommandMessage(response)
  }

  async function stopSelected() {
    const response = await runtimeMutations.stopRuntime.mutateAsync(undefined)
    pushCommandMessage(response)
  }

  async function stopSelectedScheduler() {
    const response = await runtimeMutations.stopScheduler.mutateAsync(undefined)
    pushCommandMessage(response)
  }

  function openCreateConfig() {
    // 新建配置始终从空白表单开始，避免把当前选中配置误保存成副本。
    resetEditor()
    editorOpen.value = true
  }

  function openEditor() {
    // 编辑配置沿用当前选中配置；没有配置时仍提供空白表单作为兜底入口。
    if (selectedConfigName.value && selectedConfigDetail.value) {
      applyConfigToForm(form, selectedConfigName.value, selectedConfigDetail.value)
    } else {
      applyConfigToForm(form)
    }
    editorOpen.value = true
  }

  async function saveEditorConfig() {
    const payload = configFromForm(form)
    const name = payload.name?.trim() || ''
    if (!name) {
      pushAlert('请填写配置名称。', '提示')
      return
    }
    const response = await configMutations.saveConfig.mutateAsync({ name, config: payload })
    const savedName = response.item?.name || name
    selectedConfigName.value = savedName
    selectedConfigDetail.value = response.item ? configWithName(response.item.config, savedName) : configWithName(payload, savedName)
    pushCommandMessage(response)
    editorOpen.value = false
  }

  async function deleteSelectedConfig(target: string) {
    const response = await configMutations.deleteConfig.mutateAsync(target)
    selectedConfigName.value = response.dashboard.configs?.[0]?.name || ''
    selectedConfigDetail.value = null
    pushCommandMessage(response)
  }

  function requestDeleteSelectedConfig() {
    const target = selectedConfigName.value
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
    const response = await apiClient.planDetail(plan.name)
    selectedPlanDetail.value = normalizePlanDetail(response.item)
    planDialogOpen.value = true
  }

  async function stopNamedPlan(plan: PlanSummary) {
    const response = await runtimeMutations.stopPlan.mutateAsync(plan.name)
    pushCommandMessage(response)
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
    applyConfigToForm(form)
    cronPreview.value = []
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

    if (logEnabled.value) {
      logs.value = [...logs.value, ...(activity.logs || [])].slice(-200)
    }
    logCursor.value = Number(activity.log_cursor || logCursor.value)
    eventCursor.value = Number(activity.event_cursor || eventCursor.value)

    let shouldRefreshDashboard = false
    for (const event of activity.events || []) {
      const projection = projectActivityEvent(event)
      if (projection.refreshDashboard) {
        shouldRefreshDashboard = true
      }
      if (projection.toast) {
        pushAlert(projection.toast.message, projection.toast.title, projection.toast.variant, projection.toast.detail)
      }
    }

    if (shouldRefreshDashboard) {
      await dashboardCache.invalidateDashboard()
    }
  })

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
        await runAction(async () => {
          const response = await cronPreviewMutation.mutateAsync(expression)
          if (form.cron_expr.trim() === expression) {
            cronPreview.value = response.items || []
          }
        })
      }, 500)
    },
  )

  onBeforeUnmount(() => {
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
    openCreateConfig,
    openEditor,
    openPlanDetail,
    planDialogOpen,
    planState,
    plans,
    refreshDashboard,
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
    statusLabel: statusLabelValue,
    statusState,
    statusVariant: statusVariantValue,
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
