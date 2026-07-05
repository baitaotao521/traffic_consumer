import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import type { Ref } from 'vue'

import { apiClient } from './client'
import type { SaveConfigOptions } from './client'
import type {
  Config,
  ConfigCommandResponse,
  DashboardResponse,
  DashboardSnapshot,
  RuntimeCommandResponse,
} from './generated'

export const dashboardQueryKey = ['dashboard'] as const
export const activityQueryKey = ['activity'] as const
export const configurationQueryKey = (name: string) => ['configuration', name] as const
export const planDetailQueryKey = (name: string) => ['plan-detail', name] as const

type DashboardCommandResponse = ConfigCommandResponse | RuntimeCommandResponse

const activeDashboardRefetchInterval = 1000
const idleDashboardRefetchInterval = 5000

function dashboardHasActiveWork(snapshot?: DashboardSnapshot) {
  if (!snapshot) return true
  if (snapshot.status.running || snapshot.status.state === 'running' || snapshot.status.state === 'scheduled') {
    return true
  }
  const plans = snapshot.plans || []
  return plans.some((plan) => plan.running || plan.scheduler_running)
}

function dashboardRefetchInterval(response?: DashboardResponse) {
  return dashboardHasActiveWork(response?.dashboard) ? activeDashboardRefetchInterval : idleDashboardRefetchInterval
}

export function useDashboardQuery() {
  return useQuery({
    queryKey: dashboardQueryKey,
    queryFn: apiClient.dashboard,
    // Dashboard 是全量快照；空闲时靠命令响应和 activity 事件刷新缓存，保留低频兜底即可。
    refetchInterval: (query) => dashboardRefetchInterval(query.state.data),
  })
}

export function useActivityQuery(cursors: {
  afterLog: Ref<number>
  afterEvent: Ref<number>
  includeLogs: Ref<boolean>
  logLevels: Ref<string[]>
  logSources: Ref<string[]>
}) {
  return useQuery({
    queryKey: activityQueryKey,
    queryFn: () => apiClient.activity(cursors.afterLog.value, cursors.afterEvent.value, {
      includeLogs: cursors.includeLogs.value,
      logLevels: cursors.logLevels.value,
      logSources: cursors.logSources.value,
    }),
    refetchInterval: 1000,
  })
}

export function useDashboardCache() {
  const queryClient = useQueryClient()

  return {
    setDashboard(snapshot: DashboardSnapshot) {
      queryClient.setQueryData<DashboardResponse>(dashboardQueryKey, { dashboard: snapshot })
    },
    invalidateDashboard() {
      return queryClient.invalidateQueries({ queryKey: dashboardQueryKey })
    },
  }
}

export function useDashboardCommandMutation<TVariables, TResponse extends DashboardCommandResponse>(
  mutationFn: (variables: TVariables) => Promise<TResponse>,
) {
  const cache = useDashboardCache()
  return useMutation({
    mutationFn,
    onSuccess(response) {
      cache.setDashboard(response.dashboard)
    },
  })
}

export function useConfigCommandMutations() {
  return {
    saveConfig: useDashboardCommandMutation<
      { name: string; config: Config; options?: SaveConfigOptions },
      ConfigCommandResponse
    >((variables) =>
      apiClient.saveConfig(variables.name, variables.config, variables.options),
    ),
    deleteConfig: useDashboardCommandMutation<string, ConfigCommandResponse>((name) => apiClient.deleteConfig(name)),
  }
}

export function useRuntimeCommandMutations() {
  return {
    startRuntime: useDashboardCommandMutation<Config, RuntimeCommandResponse>((config) => apiClient.startRuntime(config)),
    startTempTest: useDashboardCommandMutation<Config, RuntimeCommandResponse>((config) =>
      apiClient.startTempTest(config),
    ),
    stopPlan: useDashboardCommandMutation<string, RuntimeCommandResponse>((name) => apiClient.stopPlan(name)),
    stopRuntime: useDashboardCommandMutation<void, RuntimeCommandResponse>(() => apiClient.stopRuntime()),
    stopScheduler: useDashboardCommandMutation<void, RuntimeCommandResponse>(() => apiClient.stopScheduler()),
  }
}
