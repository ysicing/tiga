import { apiClient } from '@/lib/api-client'

// Types
export interface AuditEvent {
  id: string
  timestamp: number
  action: string
  resource_type: string
  resource_id?: string
  resource_name?: string
  subsystem: string
  user_uid?: string
  user_name?: string
  principal_type: 'user' | 'service' | 'system'
  client_ip?: string
  user_agent?: string
  success: boolean
  error_message?: string
  diff_object?: {
    old_object?: any
    new_object?: any
    old_object_truncated?: boolean
    new_object_truncated?: boolean
    truncated_fields?: string[]
  }
  metadata?: Record<string, any>
  created_at: string
}

export interface AuditConfig {
  retention_days: number
  max_object_bytes: number
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

// Subsystem types
export const SUBSYSTEMS = [
  'http',
  'minio',
  'database',
  'middleware',
  'kubernetes',
  'docker',
  'host',
  'webssh',
  'scheduler',
  'alert',
  'auth',
  'storage',
  'webserver',
] as const

export type Subsystem = (typeof SUBSYSTEMS)[number]

// Action types
export const ACTIONS = [
  'created',
  'updated',
  'deleted',
  'accessed',
  'executed',
  'uploaded',
  'downloaded',
  'login',
  'logout',
  'granted',
  'revoked',
] as const

export type Action = (typeof ACTIONS)[number]

// Audit Service
export const auditService = {
  // Events
  async getEvents(params?: {
    subsystem?: string
    action?: string
    resource_type?: string
    user_uid?: string
    start_time?: number
    end_time?: number
    success?: boolean
    page?: number
    page_size?: number
  }): Promise<PaginatedResponse<AuditEvent>> {
    const response = await apiClient.get<PaginatedResponse<AuditEvent>>('/audit/events', { params })
    return response
  },

  async getEvent(id: string): Promise<AuditEvent> {
    const response = await apiClient.get<{ data: AuditEvent }>(`/audit/events/${id}`)
    return response.data
  },

  // Config
  async getConfig(): Promise<AuditConfig> {
    const response = await apiClient.get<{ data: AuditConfig }>('/audit/config')
    return response.data
  },

  async updateConfig(config: Partial<AuditConfig>): Promise<AuditConfig> {
    const response = await apiClient.put<{ data: AuditConfig }>('/audit/config', config)
    return response.data
  },
}
