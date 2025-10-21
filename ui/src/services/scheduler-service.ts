import { apiClient } from '@/lib/api-client'

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
  id: string
  task_uid: string
  state: string
  started_at: string
  duration: number
  result: string
}

export interface ExecutionsResponse {
  executions: SchedulerExecution[]
  total: number
}

export interface TriggerResponse {
  uid: string
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
    const response = await apiClient.get('/scheduler/tasks')
    return response.data || response.tasks || []
  }

  async getExecutions(params?: {
    page?: number
    page_size?: number
    task_name?: string
    state?: string
    start_time?: number
    end_time?: number
  }): Promise<ExecutionsResponse> {
    const response = await apiClient.get('/scheduler/executions', params)
    return {
      executions: response.data || [],
      total: response.pagination?.total || 0,
    }
  }

  async triggerTask(taskUid: string): Promise<TriggerResponse> {
    const response = await apiClient.post(`/scheduler/tasks/${taskUid}/trigger`)
    return response
  }

  async getStats(): Promise<SchedulerStats> {
    const response = await apiClient.get('/scheduler/stats')
    return response
  }
}

export const schedulerService = new SchedulerService()
