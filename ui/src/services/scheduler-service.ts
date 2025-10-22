import { apiClient } from '@/lib/api-client'
import { z } from 'zod'

// Zod schemas for runtime validation
const SchedulerTaskSchema = z.object({
  uid: z.string(),
  name: z.string(),
  type: z.string(),
  description: z.string().optional(),
  is_recurring: z.boolean(),
  cron_expr: z.string().optional(),
  interval: z.number().optional(),
  next_run: z.string().optional(),
  enabled: z.boolean(),
  max_duration_seconds: z.number(),
  max_retries: z.number().optional(),
  timeout_grace_period: z.number().optional(),
  max_concurrent: z.number().optional(),
  priority: z.number().optional(),
  labels: z.record(z.string(), z.string()).optional(),
  data: z.string().optional(),
  total_executions: z.number(),
  success_executions: z.number(),
  failure_executions: z.number(),
  consecutive_failures: z.number(),
  last_executed_at: z.string().optional(),
  last_failure_error: z.string().optional(),
  created_at: z.string(),
  updated_at: z.string(),
})

const SchedulerExecutionSchema = z.object({
  id: z.number(),
  task_uid: z.string(),
  task_name: z.string(),
  task_type: z.string(),
  execution_uid: z.string(),
  run_by: z.string(),
  scheduled_at: z.string(),
  started_at: z.string(),
  finished_at: z.string().optional(),
  state: z.string(),
  result: z.string().optional(),
  error: z.string().optional(),
  error_message: z.string().optional(),
  error_stack: z.string().optional(),
  duration_ms: z.number(),
  progress: z.number(),
  retry_count: z.number(),
  trigger_type: z.string(),
  trigger_by: z.string().optional(),
  triggered_by: z.string().optional(),
  created_at: z.string(),
  updated_at: z.string(),
})

const PaginationSchema = z.object({
  page: z.number(),
  page_size: z.number(),
  total: z.number(),
  total_pages: z.number(),
})

const ExecutionsResponseSchema = z.object({
  data: z.array(SchedulerExecutionSchema),
  pagination: PaginationSchema,
})

const TriggerResponseSchema = z.object({
  uid: z.string(),
  execution_uid: z.string().optional(),
  message: z.string(),
})

const SchedulerStatsSchema = z.object({
  total_tasks: z.number(),
  enabled_tasks: z.number(),
  total_executions: z.number(),
  avg_success_rate: z.number(),
})

// TypeScript types derived from Zod schemas
export type SchedulerTask = z.infer<typeof SchedulerTaskSchema>
export type SchedulerExecution = z.infer<typeof SchedulerExecutionSchema>
export interface ExecutionsResponse {
  data: SchedulerExecution[]
  pagination: {
    page: number
    page_size: number
    total: number
    total_pages: number
  }
}
export type TriggerResponse = z.infer<typeof TriggerResponseSchema>
export type SchedulerStats = z.infer<typeof SchedulerStatsSchema>

class SchedulerService {
  async getTasks(): Promise<SchedulerTask[]> {
    const response = await apiClient.get('/scheduler/tasks')
    let data: unknown
    if (Array.isArray(response)) {
      data = response
    } else {
      data = (response as any).data || (response as any).tasks || []
    }
    // Runtime validation with Zod
    const validated = z.array(SchedulerTaskSchema).parse(data)
    return validated
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
    const response = await apiClient.get('/scheduler/executions', params)
    // Runtime validation with Zod - with fallback for missing data
    const result = ExecutionsResponseSchema.safeParse(response)
    if (result.success) {
      return result.data
    }
    // Fallback: construct valid response if backend returns unexpected format
    return {
      data: [],
      pagination: {
        page: 1,
        page_size: 20,
        total: 0,
        total_pages: 0,
      },
    }
  }

  async triggerTask(taskUid: string): Promise<TriggerResponse> {
    const response = await apiClient.post(`/scheduler/tasks/${taskUid}/trigger`)
    // Runtime validation with Zod
    const validated = TriggerResponseSchema.parse(response)
    return validated
  }

  async getStats(): Promise<SchedulerStats> {
    const response = await apiClient.get('/scheduler/stats')
    // Runtime validation with Zod
    const validated = SchedulerStatsSchema.parse(response)
    return validated
  }
}

export const schedulerService = new SchedulerService()
