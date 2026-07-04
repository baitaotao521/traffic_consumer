import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import type { Ref } from 'vue'

import { apiClient } from './client'
import type {
  ConfigCommandResponse,
  ConfigWritable,
  DashboardResponse,
  DashboardSnapshot,
  RuntimeCommandResponse,
} from './generated'

export const dashboardQueryKey = ['dashboard'] as const
export const activityQueryKey = ['activity'] as const
export const configurationQueryKey = (name: string) => ['configuration', name] as const
export const planDetailQueryKey = (name: string) => ['plan-detail', name] as const

type DashboardCommandResponse = ConfigCommandResponse | RuntimeCommandResponse

export function useDashboardQuery() {
  return useQuery({
    queryKey: dashboardQueryKey,
    queryFn: apiClient.dashboard,
    refetchInterval: 1000,
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
    saveConfig: useDashboardCommandMutation<{ name: string; config: ConfigWritable }, ConfigCommandResponse>((variables) =>
      apiClient.saveConfig(variables.name, variables.config),
    ),
    deleteConfig: useDashboardCommandMutation<string, ConfigCommandResponse>((name) => apiClient.deleteConfig(name)),
  }
}

export function useRuntimeCommandMutations() {
  return {
    startRuntime: useDashboardCommandMutation<ConfigWritable, RuntimeCommandResponse>((config) => apiClient.startRuntime(config)),
    startTempTest: useDashboardCommandMutation<ConfigWritable, RuntimeCommandResponse>((config) =>
      apiClient.startTempTest(config),
    ),
    stopPlan: useDashboardCommandMutation<string, RuntimeCommandResponse>((name) => apiClient.stopPlan(name)),
    stopRuntime: useDashboardCommandMutation<void, RuntimeCommandResponse>(() => apiClient.stopRuntime()),
    stopScheduler: useDashboardCommandMutation<void, RuntimeCommandResponse>(() => apiClient.stopScheduler()),
  }
}
