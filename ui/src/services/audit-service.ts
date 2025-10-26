import { apiClient } from '@/lib/api-client'

// TypeScript interfaces for type safety
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
  resource_name?: string
  user_name?: string
  success?: boolean
  error_message?: string
  metadata?: Record<string, any>
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

// Subsystem types - match backend SubsystemType enum
export const SUBSYSTEMS = [
  'http',
  'docker',
  'database',
  'kubernetes',
  'host',
  'webssh',
  'scheduler',
  'alert',
  'auth',
  'minio',
  'middleware',
  'storage',
  'webserver',
]

// Action types - match backend Action enum
export const ACTIONS = [
  // Basic CRUD operations
  'created',
  'updated',
  'deleted',
  'read',

  // State change operations
  'enabled',
  'disabled',

  // Special operations (Gitness reference)
  'bypassed',
  'forcePush',

  // Authentication operations
  'login',
  'logout',

  // Permission operations
  'granted',
  'revoked',

  // Host management operations (T038)
  'agent_connected',
  'agent_disconnected',
  'agent_reconnected',
  'terminal_created',
  'terminal_closed',
  'terminal_replay',
  'node_created',
  'node_updated',
  'node_deleted',
  'system_alert',
  'system_error',
]

class AuditService {
  async getEvents(params?: {
    page?: number
    page_size?: number
    subsystem?: string
    user_uid?: string
    action?: string
    resource_type?: string
    start_time?: number
    end_time?: number
    client_ip?: string
    request_id?: string
  }): Promise<AuditEventsResponse> {
    const response = await apiClient.get('/audit/events', params) as AuditEventsResponse
    // Defensive check for response structure
    if (!response?.data || !Array.isArray(response.data)) {
      console.warn('Invalid audit events response format, returning empty')
      return {
        data: [],
        pagination: { page: 1, page_size: 20, total: 0, total_pages: 0 },
      }
    }
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
