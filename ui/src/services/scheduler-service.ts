import { apiClient } from '@/lib/api-client'

// TypeScript interfaces for type safety
export interface SchedulerTask {
  uid: string
  name: string
  type: string
  description?: string
  is_recurring: boolean
  cron_expr?: string
  interval?: number
  next_run?: string
  enabled: boolean
  max_duration_seconds: number
  max_retries?: number
  timeout_grace_period?: number
  max_concurrent?: number
  priority?: number
  labels?: Record<string, string>
  data?: string
  total_executions: number
  success_executions: number
  failure_executions: number
  consecutive_failures: number
  last_executed_at?: string
  last_failure_error?: string
  created_at: string
  updated_at: string
}

export interface SchedulerExecution {
  id: number
  task_uid: string
  task_name: string
  task_type: string
  execution_uid: string
  run_by: string
  scheduled_at: string
  started_at: string
  finished_at?: string
  state: string
  result?: string
  error?: string
  error_message?: string
  error_stack?: string
  duration_ms: number
  progress: number
  retry_count: number
  trigger_type: string
  trigger_by?: string
  triggered_by?: string
  created_at: string
  updated_at: string
}

export interface ExecutionsResponse {
  data: SchedulerExecution[]
  pagination: {
    page: number
    page_size: number
    total: number
    total_pages: number
  }
}

export interface TriggerResponse {
  uid: string
  execution_uid?: string
  message: string
}

export interface SchedulerStats {
  total_tasks: number
  enabled_tasks: number
  total_executions: number
  avg_success_rate: number
}

class SchedulerService {
  async getTasks(): Promise<SchedulerTask[]> {
    const response = await apiClient.get('/scheduler/tasks') as { data?: SchedulerTask[]; tasks?: SchedulerTask[] } | SchedulerTask[]
    if (Array.isArray(response)) {
      return response
    }
    return response.data || response.tasks || []
  }

  async getExecutions(params?: {
    page?: number
    page_size?: number
    task_uid?: string
    task_name?: string
    state?: string
    start_time?: number
    end_time?: number
  }): Promise<ExecutionsResponse> {
    const response = await apiClient.get('/scheduler/executions', params) as ExecutionsResponse
    // Defensive check with fallback
    return {
      data: response.data || [],
      pagination: response.pagination || {
        page: 1,
        page_size: 20,
        total: 0,
        total_pages: 0,
      },
    }
  }

  async triggerTask(taskUid: string): Promise<TriggerResponse> {
    const response = await apiClient.post(`/scheduler/tasks/${taskUid}/trigger`) as TriggerResponse
    return response
  }

  async getStats(): Promise<SchedulerStats> {
    const response = await apiClient.get('/scheduler/stats') as SchedulerStats
    return response
  }
}

export const schedulerService = new SchedulerService()
