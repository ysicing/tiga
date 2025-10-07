// API client with authentication support
class ApiClient {
  private baseUrl: string = ''
  private isRefreshing = false
  private refreshPromise: Promise<void> | null = null
  private getCurrentCluster: (() => string | null) | null = null

  constructor(baseUrl: string = '') {
    this.baseUrl = baseUrl
  }

  // Set cluster provider function
  setClusterProvider(provider: () => string | null) {
    this.getCurrentCluster = provider
  }

  private async refreshToken(): Promise<void> {
    if (this.isRefreshing) {
      return this.refreshPromise!
    }

    this.isRefreshing = true
    this.refreshPromise = fetch('/api/auth/refresh', {
      method: 'POST',
      credentials: 'include',
    })
      .then((response) => {
        if (!response.ok) {
          throw new Error('Token refresh failed')
        }
      })
      .finally(() => {
        this.isRefreshing = false
        this.refreshPromise = null
      })

    return this.refreshPromise
  }

  private async makeRequest<T>(
    url: string,
    options: RequestInit = {}
  ): Promise<T> {
    const fullUrl = this.baseUrl + url

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string>),
    }

    // Add cluster header if available
    const currentCluster = this.getCurrentCluster?.()
    if (currentCluster) {
      headers['x-cluster-name'] = currentCluster
    }

    const defaultOptions: RequestInit = {
      credentials: 'include',
      headers,
      ...options,
    }

    try {
      let response = await fetch(fullUrl, defaultOptions)

      // Handle authentication errors with automatic retry
      if (response.status === 401) {
        try {
          // Try to refresh the token
          await this.refreshToken()
          // Retry the original request
          response = await fetch(fullUrl, defaultOptions)
        } catch (refreshError) {
          // If refresh fails, redirect to login page
          console.error('Token refresh failed:', refreshError)
          window.location.href = '/login'
          throw new Error('Authentication failed')
        }
      }

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(
          errorData.error || `HTTP error! status: ${response.status}`
        )
      }

      const contentType = response.headers.get('content-type')
      if (contentType && contentType.includes('application/json')) {
        return await response.json()
      } else {
        return (await response.text()) as T
      }
    } catch (error) {
      console.error('API request failed:', error)
      throw error
    }
  }

  async get<T>(url: string, options?: RequestInit): Promise<T> {
    return this.makeRequest<T>(url, { ...options, method: 'GET' })
  }

  async post<T>(
    url: string,
    data?: unknown,
    options?: RequestInit
  ): Promise<T> {
    return this.makeRequest<T>(url, {
      ...options,
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    })
  }

  async put<T>(url: string, data?: unknown, options?: RequestInit): Promise<T> {
    return this.makeRequest<T>(url, {
      ...options,
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    })
  }

  async delete<T>(url: string, options?: RequestInit): Promise<T> {
    return this.makeRequest<T>(url, { ...options, method: 'DELETE' })
  }

  async patch<T>(
    url: string,
    data?: unknown,
    options?: RequestInit
  ): Promise<T> {
    return this.makeRequest<T>(url, {
      ...options,
      method: 'PATCH',
      body: data ? JSON.stringify(data) : undefined,
    })
  }
}

export const API_BASE_URL = '/api/v1'

// Create a singleton instance
export const apiClient = new ApiClient(API_BASE_URL)

// Helper function to build K8s resource URLs with cluster ID
export function buildK8sResourceUrl(path: string): string {
  const currentCluster = localStorage.getItem('current-cluster')
  if (!currentCluster) {
    console.warn('No current cluster selected, K8s API call may fail')
    return path
  }

  // If path already starts with /cluster/, don't add it again
  if (path.startsWith('/cluster/')) {
    return path
  }

  // K8s resource paths that need cluster prefix
  const k8sPaths = [
    '/overview',
    '/prometheus/',
    '/logs/',
    '/terminal/',
    '/node-terminal/',
    '/search',
    '/resources/apply',
    '/image/tags',
    '/pods', '/namespaces', '/nodes', '/services', '/deployments',
    '/configmaps', '/secrets', '/persistentvolumes', '/persistentvolumeclaims',
    '/storageclasses', '/statefulsets', '/daemonsets', '/replicasets',
    '/jobs', '/cronjobs', '/ingresses', '/ingressclasses', '/networkpolicies',
    '/serviceaccounts', '/roles', '/rolebindings', '/clusterroles',
    '/clusterrolebindings', '/horizontalpodautoscalers', '/poddisruptionbudgets',
    '/endpoints', '/events', '/limitranges', '/resourcequotas',
    '/customresourcedefinitions', '/crds'
  ]

  // Check if path needs cluster prefix
  const needsClusterPrefix = k8sPaths.some(p => path.startsWith(p))

  if (needsClusterPrefix) {
    return `/cluster/${currentCluster}${path}`
  }

  return path
}

// DevOps Platform API methods
export const devopsAPI = {
  // Auth
  auth: {
    login: (username: string, password: string) =>
      apiClient.post('/auth/login', { username, password }),
    logout: (token: string) => apiClient.post('/auth/logout', { token }),
    refreshToken: (refreshToken: string) =>
      apiClient.post('/auth/refresh', { refresh_token: refreshToken }),
    getProfile: () => apiClient.get('/auth/profile'),
    changePassword: (oldPassword: string, newPassword: string) =>
      apiClient.post('/auth/change-password', {
        old_password: oldPassword,
        new_password: newPassword,
      }),
  },

  // Instances
  instances: {
    list: (params?: Record<string, any>) => apiClient.get('/instances', { ...params }),
    get: (id: string) => apiClient.get(`/instances/${id}`),
    create: (data: Record<string, any>) => apiClient.post('/instances', data),
    update: (id: string, data: Record<string, any>) => apiClient.patch(`/instances/${id}`, data),
    delete: (id: string) => apiClient.delete(`/instances/${id}`),
    updateStatus: (id: string, status: string) =>
      apiClient.patch(`/instances/${id}/status`, { status }),
    updateHealth: (id: string, health: string, message?: string) =>
      apiClient.patch(`/instances/${id}/health`, { health, health_message: message }),
    statistics: () => apiClient.get('/instances/statistics'),
  },

  // Metrics
  metrics: {
    query: (params: Record<string, any>) => apiClient.get('/metrics', { ...params }),
    create: (data: Record<string, any>) => apiClient.post('/metrics', data),
    aggregate: (params: Record<string, any>) => apiClient.get('/metrics/aggregate', { ...params }),
    timeseries: (params: Record<string, any>) => apiClient.get('/metrics/timeseries', { ...params }),
  },

  // Alerts
  alerts: {
    listRules: (params?: Record<string, any>) => apiClient.get('/alerts', { ...params }),
    getRule: (id: string) => apiClient.get(`/alerts/${id}`),
    createRule: (data: Record<string, any>) => apiClient.post('/alerts', data),
    updateRule: (id: string, data: Record<string, any>) => apiClient.patch(`/alerts/${id}`, data),
    deleteRule: (id: string) => apiClient.delete(`/alerts/${id}`),
    toggleRule: (id: string, enabled: boolean) =>
      apiClient.patch(`/alerts/${id}/toggle`, { enabled }),
    listEvents: (params?: Record<string, any>) => apiClient.get('/alerts/events', { ...params }),
    acknowledgeEvent: (eventId: string, note?: string) =>
      apiClient.post(`/alerts/events/${eventId}/acknowledge`, { note }),
    resolveEvent: (eventId: string, note?: string) =>
      apiClient.post(`/alerts/events/${eventId}/resolve`, { note }),
  },

  // Audit Logs
  audit: {
    list: (params?: Record<string, any>) => apiClient.get('/audit', params),
    get: (id: string) => apiClient.get(`/audit/${id}`),
    search: (query: string, limit?: number) => {
      const params = new URLSearchParams({ query });
      if (limit) params.append('limit', limit.toString());
      return apiClient.get(`/audit/search?${params.toString()}`);
    },
    statistics: () => apiClient.get('/audit/statistics'),
  },

  // Users
  users: {
    list: (params?: Record<string, any>) => apiClient.get('/users', { ...params }),
    get: (id: string) => apiClient.get(`/users/${id}`),
    create: (data: Record<string, any>) => apiClient.post('/users', data),
    update: (id: string, data: Record<string, any>) => apiClient.patch(`/users/${id}`, data),
    delete: (id: string) => apiClient.delete(`/users/${id}`),
    assignRoles: (userId: string, roleIds: string[]) =>
      apiClient.post(`/users/${userId}/roles`, { role_ids: roleIds }),
  },

  // Roles
  roles: {
    list: (params?: Record<string, any>) => apiClient.get('/roles', { ...params }),
    get: (id: string) => apiClient.get(`/roles/${id}`),
    create: (data: Record<string, any>) => apiClient.post('/roles', data),
    update: (id: string, data: Record<string, any>) => apiClient.patch(`/roles/${id}`, data),
    delete: (id: string) => apiClient.delete(`/roles/${id}`),
  },

  // MinIO
  minio: {
    listBuckets: (instanceId: string) => apiClient.get(`/minio/instances/${instanceId}/buckets`),
    createBucket: (instanceId: string, bucketName: string, region?: string) =>
      apiClient.post(`/minio/instances/${instanceId}/buckets`, { bucket_name: bucketName, region }),
    deleteBucket: (instanceId: string, bucketName: string) =>
      apiClient.delete(`/minio/instances/${instanceId}/buckets/${bucketName}`),
    listObjects: (instanceId: string, bucketName: string, prefix?: string) => {
      const params = new URLSearchParams();
      if (prefix) params.append('prefix', prefix);
      return apiClient.get(`/minio/instances/${instanceId}/buckets/${bucketName}/objects?${params.toString()}`);
    },
    uploadObject: (instanceId: string, bucketName: string, objectName: string, file: File) => {
      const formData = new FormData();
      formData.append('file', file);
      formData.append('object_name', objectName);
      return fetch(`${API_BASE_URL}/minio/instances/${instanceId}/buckets/${bucketName}/objects`, {
        method: 'POST',
        body: formData,
        credentials: 'include',
      });
    },
    deleteObject: (instanceId: string, bucketName: string, objectName: string) =>
      apiClient.delete(`/minio/instances/${instanceId}/buckets/${bucketName}/objects/${objectName}`),
    getObjectUrl: (instanceId: string, bucketName: string, objectName: string) =>
      apiClient.get(`/minio/instances/${instanceId}/buckets/${bucketName}/objects/${objectName}/url`),
  },

  // Database
  database: {
    listDatabases: (instanceId: string) => apiClient.get(`/database/instances/${instanceId}/databases`),
    createDatabase: (instanceId: string, dbName: string, charset?: string, collation?: string) =>
      apiClient.post(`/database/instances/${instanceId}/databases`, {
        database_name: dbName,
        charset,
        collation,
      }),
    deleteDatabase: (instanceId: string, dbName: string) =>
      apiClient.delete(`/database/instances/${instanceId}/databases/${dbName}`),
    listUsers: (instanceId: string) => apiClient.get(`/database/instances/${instanceId}/users`),
    createUser: (instanceId: string, username: string, password: string, host?: string) =>
      apiClient.post(`/database/instances/${instanceId}/users`, {
        username,
        password,
        host: host || '%',
      }),
    deleteUser: (instanceId: string, username: string) =>
      apiClient.delete(`/database/instances/${instanceId}/users/${username}`),
    grantPermissions: (instanceId: string, username: string, database: string, privileges: string[]) =>
      apiClient.post(`/database/instances/${instanceId}/users/${username}/grant`, {
        database,
        privileges,
      }),
    executeQuery: (instanceId: string, database: string, query: string) =>
      apiClient.post(`/database/instances/${instanceId}/query`, {
        database,
        query,
      }),
  },
};
