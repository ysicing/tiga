import { apiClient } from '@/lib/api-client'
import { z } from 'zod'

// Zod schemas for runtime validation
const PrincipalSchema = z.object({
  uid: z.string(),
  username: z.string(),
  type: z.enum(['user', 'service', 'anonymous', 'system']),
})

const ResourceSchema = z.object({
  type: z.string(),
  identifier: z.string(),
  data: z.record(z.string(), z.string()).optional(),
})

const DiffObjectSchema = z.object({
  old_object: z.string().optional(),
  new_object: z.string().optional(),
  old_object_truncated: z.boolean(),
  new_object_truncated: z.boolean(),
  truncated_fields: z.array(z.string()).optional(),
})

const AuditEventSchema = z.object({
  id: z.string(),
  timestamp: z.number(),
  action: z.string(),
  resource_type: z.string(),
  resource: ResourceSchema,
  subsystem: z.string(),
  user: PrincipalSchema,
  space_path: z.string().optional(),
  diff_object: DiffObjectSchema.optional(),
  client_ip: z.string(),
  user_agent: z.string().optional(),
  request_method: z.string().optional(),
  request_id: z.string().optional(),
  data: z.record(z.string(), z.string()).optional(),
  created_at: z.string(),
  // Computed fields
  resource_name: z.string().optional(),
  user_name: z.string().optional(),
  success: z.boolean().optional(),
  error_message: z.string().optional(),
  metadata: z.record(z.string(), z.any()).optional(),
})

const PaginationSchema = z.object({
  page: z.number(),
  page_size: z.number(),
  total: z.number(),
  total_pages: z.number(),
})

const AuditEventsResponseSchema = z.object({
  data: z.array(AuditEventSchema),
  pagination: PaginationSchema,
})

const AuditConfigSchema = z.object({
  retention_days: z.number(),
  max_object_bytes: z.number().optional(),
})

// TypeScript interfaces (exported for use in components)
export type Principal = z.infer<typeof PrincipalSchema>
export type Resource = z.infer<typeof ResourceSchema>
export type DiffObject = z.infer<typeof DiffObjectSchema>
export type AuditEvent = z.infer<typeof AuditEventSchema>
export interface AuditEventsResponse {
  data: AuditEvent[]
  pagination: {
    page: number
    page_size: number
    total: number
    total_pages: number
  }
}
export type AuditConfig = z.infer<typeof AuditConfigSchema>

export const SUBSYSTEMS = [
  'cluster',
  'pod',
  'deployment',
  'service',
  'configmap',
  'secret',
  'database',
  'databaseInstance',
  'databaseUser',
  'minio',
  'redis',
  'mysql',
  'postgresql',
  'user',
  'role',
  'instance',
  'scheduledTask',
]

export const ACTIONS = [
  'created',
  'updated',
  'deleted',
  'read',
  'enabled',
  'disabled',
  'bypassed',
  'forcePush',
  'login',
  'logout',
  'granted',
  'revoked',
]

class AuditService {
  async getEvents(params?: {
    page?: number
    page_size?: number
    user_uid?: string
    action?: string
    resource_type?: string
    start_time?: number
    end_time?: number
    client_ip?: string
    request_id?: string
  }): Promise<AuditEventsResponse> {
    const response = await apiClient.get('/audit/events', params)
    // Runtime validation with Zod
    const validated = AuditEventsResponseSchema.parse(response)
    return validated
  }

  async getEvent(id: string): Promise<AuditEvent> {
    const response = await apiClient.get('/audit/events/' + id) as { data: unknown }
    // Runtime validation with Zod
    const validated = AuditEventSchema.parse(response.data)
    return validated
  }

  async getConfig(): Promise<AuditConfig> {
    const response = await apiClient.get('/audit/config')
    const data = 'data' in (response as any) ? (response as any).data : response
    // Runtime validation with Zod
    const validated = AuditConfigSchema.parse(data)
    return validated
  }

  async search(query: string, limit?: number): Promise<AuditEvent[]> {
    const params = new URLSearchParams({ query })
    if (limit) params.append('limit', limit.toString())
    const response = await apiClient.get('/audit/search?' + params.toString()) as { events: unknown[] }
    // Runtime validation with Zod
    const validated = z.array(AuditEventSchema).parse(response.events || [])
    return validated
  }

  async getStatistics(): Promise<Record<string, any>> {
    const response = await apiClient.get('/audit/statistics') as Record<string, any>
    // Statistics schema varies, returning as-is for flexibility
    return response
  }
}

export const auditService = new AuditService()
