// Database Management API service

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { apiClient } from '@/lib/api-client'

// Type definitions
export interface DatabaseInstance {
  id: number
  name: string
  type: 'mysql' | 'postgresql' | 'redis'
  host: string
  port: number
  username: string
  ssl_mode?: string
  description?: string
  status: 'pending' | 'online' | 'offline' | 'error'
  version?: string
  uptime?: number
  last_check_at?: string
  created_at: string
  updated_at: string
}

export interface Database {
  id: number
  instance_id: number
  name: string
  charset?: string
  collation?: string
  owner?: string
  size?: number
  table_count?: number
  db_number?: number
  key_count?: number
  created_at: string
}

export interface DatabaseUser {
  id: number
  instance_id: number
  username: string
  host?: string
  description?: string
  is_active: boolean
  last_login_at?: string
  created_at: string
}

export interface PermissionPolicy {
  id: number
  user_id: number
  database_id: number
  role: 'readonly' | 'readwrite'
  granted_by: string
  granted_at: string
  revoked_at?: string
}

export interface QueryResult {
  columns?: string[]
  rows?: Record<string, any>[]
  affected_rows?: number
  row_count: number
  execution_time: number
  truncated?: boolean
  message?: string
}

export interface AuditLog {
  id: number
  instance_id?: number
  operator: string
  action: string
  target_type: string
  target_name: string
  details?: string
  success: boolean
  error_msg?: string
  client_ip: string
  created_at: string
}

// API functions

// Instance Management
export const useInstances = () => {
  return useQuery({
    queryKey: ['database', 'instances'],
    queryFn: async () => {
      const response = await apiClient.get<{
        data: { instances: DatabaseInstance[]; count: number }
      }>('/database/instances')
      return response
    },
  })
}

export const useInstance = (id: number) => {
  return useQuery({
    queryKey: ['database', 'instance', id],
    queryFn: () =>
      apiClient.get<{ data: DatabaseInstance }>(`/database/instances/${id}`),
    enabled: !!id,
  })
}

export const useCreateInstance = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: Partial<DatabaseInstance>) =>
      apiClient.post<{ data: DatabaseInstance }>('/database/instances', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['database', 'instances'] })
    },
  })
}

export const useUpdateInstance = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: number
      data: Partial<DatabaseInstance>
    }) =>
      apiClient.put<{ data: DatabaseInstance }>(
        `/database/instances/${id}`,
        data
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['database', 'instances'] })
      queryClient.invalidateQueries({
        queryKey: ['database', 'instance', variables.id],
      })
    },
  })
}

export const useDeleteInstance = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => apiClient.delete(`/database/instances/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['database', 'instances'] })
    },
  })
}

export const useTestConnection = () => {
  return useMutation({
    mutationFn: (id: number) =>
      apiClient.post<{
        data: { status: string; version?: string; message: string }
      }>(`/database/instances/${id}/test`),
  })
}

// Database Operations
export const useDatabases = (instanceId: number) => {
  return useQuery({
    queryKey: ['database', 'instance', instanceId, 'databases'],
    queryFn: () =>
      apiClient.get<{ data: Database[] }>(
        `/database/instances/${instanceId}/databases`
      ),
    enabled: !!instanceId,
  })
}

export const useCreateDatabase = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      instanceId,
      data,
    }: {
      instanceId: number
      data: Partial<Database>
    }) =>
      apiClient.post<{ data: Database }>(
        `/database/instances/${instanceId}/databases`,
        data
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['database', 'instance', variables.instanceId, 'databases'],
      })
    },
  })
}

export const useDeleteDatabase = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, confirmName }: { id: number; confirmName: string }) =>
      apiClient.delete(`/database/databases/${id}?confirm_name=${confirmName}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['database'] })
    },
  })
}

// User Management
export const useDatabaseUsers = (instanceId: number) => {
  return useQuery({
    queryKey: ['database', 'instance', instanceId, 'users'],
    queryFn: () =>
      apiClient.get<{ data: DatabaseUser[] }>(
        `/database/instances/${instanceId}/users`
      ),
    enabled: !!instanceId,
  })
}

export const useCreateDatabaseUser = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      instanceId,
      data,
    }: {
      instanceId: number
      data: Partial<DatabaseUser> & { password: string }
    }) =>
      apiClient.post<{ data: DatabaseUser }>(
        `/database/instances/${instanceId}/users`,
        data
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['database', 'instance', variables.instanceId, 'users'],
      })
    },
  })
}

export const useUpdateUserPassword = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      id,
      oldPassword,
      newPassword,
    }: {
      id: number
      oldPassword: string
      newPassword: string
    }) =>
      apiClient.patch(`/database/users/${id}`, {
        old_password: oldPassword,
        new_password: newPassword,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['database'] })
    },
  })
}

export const useDeleteDatabaseUser = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => apiClient.delete(`/database/users/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['database'] })
    },
  })
}

// Permission Management
export const useUserPermissions = (userId: number) => {
  return useQuery({
    queryKey: ['database', 'user', userId, 'permissions'],
    queryFn: () =>
      apiClient.get<{ data: PermissionPolicy[] }>(
        `/database/users/${userId}/permissions`
      ),
    enabled: !!userId,
  })
}

export const useGrantPermission = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: {
      user_id: number
      database_id: number
      role: 'readonly' | 'readwrite'
    }) =>
      apiClient.post<{ data: PermissionPolicy }>('/database/permissions', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['database'] })
    },
  })
}

export const useRevokePermission = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => apiClient.delete(`/database/permissions/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['database'] })
    },
  })
}

// Query Execution
export const useExecuteQuery = () => {
  return useMutation({
    mutationFn: ({
      instanceId,
      databaseName,
      query,
    }: {
      instanceId: number
      databaseName: string
      query: string
    }) =>
      apiClient.post<{ data: QueryResult }>(
        `/database/instances/${instanceId}/query`,
        {
          database_name: databaseName,
          query,
        }
      ),
  })
}

// Audit Logs
export interface AuditLogFilters {
  instance_id?: number
  operator?: string
  action?: string
  start_date?: string
  end_date?: string
  page?: number
  page_size?: number
}

export const useAuditLogs = (filters: AuditLogFilters) => {
  const queryParams = new URLSearchParams()
  Object.entries(filters).forEach(([key, value]) => {
    if (value !== undefined && value !== '') {
      queryParams.append(key, String(value))
    }
  })

  return useQuery({
    queryKey: ['database', 'audit-logs', filters],
    queryFn: () =>
      apiClient.get<{
        data: {
          logs: AuditLog[]
          total: number
          page: number
          page_size: number
        }
      }>(`/database/audit-logs?${queryParams.toString()}`),
  })
}
