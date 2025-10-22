import { apiClient } from '@/lib/api-client'

export interface Principal {
  uid: string
  username: string
  type: 'user' | 'service' | 'anonymous' | 'system'
}

export interface Resource {
  type: string
  identifier: string
  data?: Record<string, string>
}

export interface DiffObject {
  old_object?: string
  new_object?: string
  old_object_truncated: boolean
  new_object_truncated: boolean
  truncated_fields?: string[]
}

export interface AuditEvent {
  id: string
  timestamp: number
  action: string
  resource_type: string
  resource: Resource
  subsystem: string
  user: Principal
  space_path?: string
  diff_object?: DiffObject
  client_ip: string
  user_agent?: string
  request_method?: string
  request_id?: string
  data?: Record<string, string>
  created_at: string

  // Computed/flattened fields for easier access in UI
  resource_name?: string  // Extracted from resource.data.name or resource.identifier
  user_name?: string      // Extracted from user.username
  success?: boolean       // Derived from response status or error presence
  error_message?: string  // Error message if operation failed
  metadata?: Record<string, any>  // Additional metadata
}

export interface AuditEventsResponse {
  data: AuditEvent[]
  pagination: {
    page: number
    page_size: number
    total: number
    total_pages: number
  }
}

export interface AuditConfig {
  retention_days: number
  max_object_bytes?: number
}

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
    const response = await apiClient.get('/audit/events', params) as AuditEventsResponse
    return response
  }

  async getEvent(id: string): Promise<AuditEvent> {
    const response = await apiClient.get('/audit/events/' + id) as { data: AuditEvent }
    return response.data
  }

  async getConfig(): Promise<AuditConfig> {
    const response = await apiClient.get('/audit/config') as { data: AuditConfig } | AuditConfig
    return 'data' in response ? response.data : response
  }

  async search(query: string, limit?: number): Promise<AuditEvent[]> {
    const params = new URLSearchParams({ query })
    if (limit) params.append('limit', limit.toString())
    const response = await apiClient.get('/audit/search?' + params.toString()) as { events: AuditEvent[] }
    return response.events || []
  }

  async getStatistics(): Promise<Record<string, any>> {
    const response = await apiClient.get('/audit/statistics') as Record<string, any>
    return response
  }
}

export const auditService = new AuditService()
