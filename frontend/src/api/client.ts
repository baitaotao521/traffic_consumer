import {
  deleteConfiguration,
  getActivity,
  getConfiguration,
  getDashboard,
  getPlanDetail,
  previewCron,
  saveConfiguration,
  startRuntimeCommand,
  startTempTestCommand,
  stopPlanCommand,
  stopRuntimeCommand,
  stopSchedulerCommand,
} from './generated'
import type {
  ActivityResponse,
  ConfigCommandResponse,
  ConfigResourceResponse,
  ConfigWritable,
  CronPreviewResponse,
  DashboardResponse,
  ErrorModel,
  PlanDetailResponse,
  RuntimeCommandResponse,
} from './generated'

type GeneratedResult<T> = {
  data?: T
  error?: unknown
  response?: Response
}

export class ApiError extends Error {
  readonly status?: number
  readonly title?: string
  readonly details?: ErrorModel['errors']

  constructor(message: string, options: { status?: number; title?: string; details?: ErrorModel['errors'] } = {}) {
    super(message)
    this.name = 'ApiError'
    this.status = options.status
    this.title = options.title
    this.details = options.details
  }
}

function isErrorModel(value: unknown): value is ErrorModel {
  if (typeof value !== 'object' || value === null || value instanceof Error) return false
  return ['detail', 'title', 'status', 'errors', 'type'].some((key) => key in value)
}

function toApiError(error: unknown, response?: Response): ApiError {
  if (isErrorModel(error)) {
    const message = error.detail || error.title || '请求失败'
    return new ApiError(message, {
      status: error.status || response?.status,
      title: error.title,
      details: error.errors,
    })
  }
  if (error instanceof Error) {
    return new ApiError(error.message || '请求失败', { status: response?.status })
  }
  if (typeof error === 'string' && error.trim()) {
    return new ApiError(error, { status: response?.status })
  }
  return new ApiError('请求失败', { status: response?.status })
}

async function unwrap<T>(request: Promise<GeneratedResult<T>>): Promise<T> {
  const result = await request
  if (result.error !== undefined) {
    throw toApiError(result.error, result.response)
  }
  if (result.data === undefined) {
    throw new ApiError('响应为空', { status: result.response?.status })
  }
  return result.data
}

export const apiClient = {
  activity(
    afterLog: number,
    afterEvent: number,
    options: { includeLogs: boolean; logLevels: string[]; logSources: string[] },
  ): Promise<ActivityResponse> {
    return unwrap(getActivity({
      query: {
        after_log: afterLog,
        after_event: afterEvent,
        include_logs: options.includeLogs ? 'true' : 'false',
        log_levels: options.logLevels.join(','),
        log_sources: options.logSources.join(','),
      },
    }))
  },

  dashboard(): Promise<DashboardResponse> {
    return unwrap(getDashboard())
  },

  config(name: string): Promise<ConfigResourceResponse> {
    return unwrap(getConfiguration({ path: { name } }))
  },

  saveConfig(name: string, config: ConfigWritable): Promise<ConfigCommandResponse> {
    return unwrap(saveConfiguration({ path: { name }, body: config }))
  },

  deleteConfig(name: string): Promise<ConfigCommandResponse> {
    return unwrap(deleteConfiguration({ path: { name } }))
  },

  startRuntime(config: ConfigWritable): Promise<RuntimeCommandResponse> {
    return unwrap(startRuntimeCommand({ body: config }))
  },

  startTempTest(config: ConfigWritable): Promise<RuntimeCommandResponse> {
    return unwrap(startTempTestCommand({ body: config }))
  },

  stopRuntime(): Promise<RuntimeCommandResponse> {
    return unwrap(stopRuntimeCommand())
  },

  stopScheduler(): Promise<RuntimeCommandResponse> {
    return unwrap(stopSchedulerCommand())
  },

  stopPlan(name: string): Promise<RuntimeCommandResponse> {
    return unwrap(stopPlanCommand({ path: { name } }))
  },

  planDetail(name: string): Promise<PlanDetailResponse> {
    return unwrap(getPlanDetail({ path: { name } }))
  },

  previewCron(expression: string): Promise<CronPreviewResponse> {
    return unwrap(previewCron({ body: { expression } }))
  },
}
