import { apiClient } from '@/lib/api-client'

// Types
export interface SchedulerTask {
  uid: string
  name: string
  type: string
  description: string
  is_recurring: boolean
  cron_expr?: string
  enabled: boolean
  max_duration_seconds: number
  max_concurrent: number
  max_retries: number
  timeout_grace_period: number
  next_run?: string
  last_run?: string
  total_executions: number
  success_executions: number
  failure_executions: number
  created_at: string
  updated_at: string
}

export interface TaskExecution {
  id: number
  task_uid: string
  task_name: string
  execution_uid: string
  state: 'pending' | 'running' | 'success' | 'failure' | 'timeout' | 'cancelled' | 'interrupted'
  started_at: string
  finished_at?: string
  duration_ms?: number
  result?: string
  error?: string
  retry_count: number
  triggered_by: 'cron' | 'manual'
  operator_uid?: string
  created_at: string
  updated_at: string
}

export interface TaskStats {
  task_uid: string
  task_name: string
  total_executions: number
  success_executions: number
  failure_executions: number
  success_rate: number
  avg_duration_ms: number
  last_success_at?: string
  last_failure_at?: string
}

export interface PaginatedResponse<T> {
  data: T[]
  pagination: {
    page: number
    page_size: number
    total: number
    total_pages: number
  }
}

// Scheduler Service
export const schedulerService = {
  // Tasks
  async getTasks(): Promise<SchedulerTask[]> {
    const response = await apiClient.get<{ data: SchedulerTask[] }>('/scheduler/tasks')
    return response.data
  },

  async getTask(uid: string): Promise<SchedulerTask> {
    const response = await apiClient.get<{ data: SchedulerTask }>(`/scheduler/tasks/${uid}`)
    return response.data
  },

  async triggerTask(uid: string): Promise<{ execution_uid: string }> {
    const response = await apiClient.post<{ data: { execution_uid: string } }>(`/scheduler/tasks/${uid}/trigger`)
    return response.data
  },

  // Executions
  async getExecutions(params?: {
    task_uid?: string
    state?: string
    page?: number
    page_size?: number
  }): Promise<PaginatedResponse<TaskExecution>> {
    const response = await apiClient.get<PaginatedResponse<TaskExecution>>('/scheduler/executions', { params })
    return response
  },

  async getExecution(uid: string): Promise<TaskExecution> {
    const response = await apiClient.get<{ data: TaskExecution }>(`/scheduler/executions/${uid}`)
    return response.data
  },

  // Stats
  async getStats(taskUid?: string): Promise<TaskStats[]> {
    const params = taskUid ? { task_uid: taskUid } : undefined
    const response = await apiClient.get<{ data: TaskStats[] }>('/scheduler/stats', { params })
    return response.data
  },
}
