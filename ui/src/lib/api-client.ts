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

  private buildUrlWithParams(
    url: string,
    params?: Record<string, any>
  ): string {
    if (!params) {
      return url
    }

    const searchParams = new URLSearchParams()
    for (const [key, value] of Object.entries(params)) {
      if (value === undefined || value === null) {
        continue
      }
      searchParams.append(key, String(value))
    }

    if (!searchParams.toString()) {
      return url
    }

    return `${url}${url.includes('?') ? '&' : '?'}${searchParams.toString()}`
  }

  private async fetchWithAuth(
    url: string,
    options: RequestInit = {}
  ): Promise<Response> {
    const fullUrl = this.baseUrl + url
    const headers = new Headers(options.headers as HeadersInit | undefined)

    // Add JSON content type if we have a body and header not already supplied
    const hasBody = options.body !== undefined && options.body !== null
    if (
      hasBody &&
      !headers.has('Content-Type') &&
      !(options.body instanceof FormData)
    ) {
      headers.set('Content-Type', 'application/json')
    }

    // Add cluster header if available
    const currentCluster = this.getCurrentCluster?.()
    if (currentCluster && !headers.has('x-cluster-name')) {
      headers.set('x-cluster-name', currentCluster)
    }

    const defaultOptions: RequestInit = {
      credentials: 'include',
      ...options,
      headers,
    }

    try {
      let response = await fetch(fullUrl, defaultOptions)

      // Handle authentication errors with automatic retry
      if (response.status === 401) {
        // Avoid redirect loop: don't redirect if already on login page
        const currentPath = window.location.pathname
        const isLoginPage =
          currentPath === '/login' || currentPath.startsWith('/login')

        try {
          // Try to refresh the token
          await this.refreshToken()
          // Retry the original request
          response = await fetch(fullUrl, defaultOptions)
        } catch (refreshError) {
          // If refresh fails, redirect to login page (unless already there)
          console.error('Token refresh failed:', refreshError)
          if (!isLoginPage) {
            window.location.href = '/login'
          }
          throw new Error('Authentication failed')
        }
      }

      return response
    } catch (error) {
      console.error('API request failed:', error)
      throw error
    }
  }

  private async handleError(response: Response): Promise<never> {
    const contentType = response.headers.get('content-type') || ''

    if (contentType.includes('application/json')) {
      const errorData = await response.json().catch(() => ({}))
      const message =
        (errorData as { error?: string }).error ||
        `HTTP error! status: ${response.status}`
      console.error('API request failed:', message)
      throw new Error(message)
    }

    const text = await response.text().catch(() => '')
    const message = text || `HTTP error! status: ${response.status}`
    console.error('API request failed:', message)
    throw new Error(message)
  }

  private async ensureSuccess(response: Response): Promise<void> {
    if (!response.ok) {
      await this.handleError(response)
    }
  }

  private async parseResponse<T>(response: Response): Promise<T> {
    await this.ensureSuccess(response)

    const contentType = response.headers.get('content-type')
    if (contentType && contentType.includes('application/json')) {
      return (await response.json()) as T
    }

    return (await response.text()) as T
  }

  private async makeRequest<T>(
    url: string,
    options: RequestInit = {}
  ): Promise<T> {
    const response = await this.fetchWithAuth(url, options)
    return this.parseResponse<T>(response)
  }

  async get<T>(
    url: string,
    params?: Record<string, any>,
    options?: RequestInit
  ): Promise<T> {
    const requestUrl = this.buildUrlWithParams(url, params)
    return this.makeRequest<T>(requestUrl, { ...options, method: 'GET' })
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

  async postForm<T>(
    url: string,
    formData: FormData,
    options?: RequestInit
  ): Promise<T> {
    const response = await this.fetchWithAuth(url, {
      ...options,
      method: 'POST',
      body: formData,
    })
    return this.parseResponse<T>(response)
  }

  async getBlob(
    url: string,
    params?: Record<string, any>,
    options?: RequestInit
  ): Promise<Blob> {
    const requestUrl = this.buildUrlWithParams(url, params)
    const response = await this.fetchWithAuth(requestUrl, {
      ...options,
      method: 'GET',
    })
    await this.ensureSuccess(response)
    return response.blob()
  }

  async put<T>(
    url: string,
    data?: unknown,
    options?: RequestInit
  ): Promise<T> {
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
    list: (params?: Record<string, any>) => apiClient.get('/dbs', params),
    get: (id: string) => apiClient.get(`/dbs/${id}`),
    create: (data: Record<string, any>) => apiClient.post('/dbs', data),
    update: (id: string, data: Record<string, any>) => apiClient.patch(`/dbs/${id}`, data),
    delete: (id: string) => apiClient.delete(`/dbs/${id}`),
    updateStatus: (id: string, status: string) =>
      apiClient.patch(`/dbs/${id}/status`, { status }),
    updateHealth: (id: string, health: string, message?: string) =>
      apiClient.patch(`/dbs/${id}/health`, { health, health_message: message }),
    statistics: () => apiClient.get('/dbs/statistics'),
  },

  // Metrics
  metrics: {
    query: (params: Record<string, any>) => apiClient.get('/metrics', params),
    create: (data: Record<string, any>) => apiClient.post('/metrics', data),
    aggregate: (params: Record<string, any>) => apiClient.get('/metrics/aggregate', params),
    timeseries: (params: Record<string, any>) => apiClient.get('/metrics/timeseries', params),
  },

  // Alerts
  alerts: {
    listRules: (params?: Record<string, any>) => apiClient.get('/alerts', params),
    getRule: (id: string) => apiClient.get(`/alerts/${id}`),
    createRule: (data: Record<string, any>) => apiClient.post('/alerts', data),
    updateRule: (id: string, data: Record<string, any>) => apiClient.patch(`/alerts/${id}`, data),
    deleteRule: (id: string) => apiClient.delete(`/alerts/${id}`),
    toggleRule: (id: string, enabled: boolean) =>
      apiClient.patch(`/alerts/${id}/toggle`, { enabled }),
    listEvents: (params?: Record<string, any>) => apiClient.get('/alerts/events', params),
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
    list: (params?: Record<string, any>) => apiClient.get('/users', params),
    get: (id: string) => apiClient.get(`/users/${id}`),
    create: (data: Record<string, any>) => apiClient.post('/users', data),
    update: (id: string, data: Record<string, any>) => apiClient.patch(`/users/${id}`, data),
    delete: (id: string) => apiClient.delete(`/users/${id}`),
    assignRoles: (userId: string, roleIds: string[]) =>
      apiClient.post(`/users/${userId}/roles`, { role_ids: roleIds }),
  },

  // Roles
  roles: {
    list: (params?: Record<string, any>) => apiClient.get('/roles', params),
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

  // VMs (Host Monitoring)
  vms: {
    // Host management
    hosts: {
      list: () => apiClient.get('/vms/hosts'),
      get: (id: string) => apiClient.get(`/vms/hosts/${id}`),
      create: (data: Record<string, any>) => apiClient.post('/vms/hosts', data),
      update: (id: string, data: Record<string, any>) => apiClient.put(`/vms/hosts/${id}`, data),
      delete: (id: string) => apiClient.delete(`/vms/hosts/${id}`),
      getCurrentState: (id: string) => apiClient.get(`/vms/hosts/${id}/state/current`),
      getHistoryState: (id: string, params?: Record<string, any>) =>
        apiClient.get(`/vms/hosts/${id}/state/history`, params),
      getActivities: (id: string, params?: Record<string, any>) =>
        apiClient.get(`/vms/hosts/${id}/activities`, params),
      regenerateSecretKey: (id: string) => apiClient.post(`/vms/hosts/${id}/regenerate-key`),
      getAgentInstallCommand: (id: string) => apiClient.get(`/vms/hosts/${id}/agent-command`),
    },

    // Service monitors
    serviceMonitors: {
      list: () => apiClient.get('/vms/service-monitors'),
      get: (id: string) => apiClient.get(`/vms/service-monitors/${id}`),
      create: (data: Record<string, any>) => apiClient.post('/vms/service-monitors', data),
      update: (id: string, data: Record<string, any>) =>
        apiClient.put(`/vms/service-monitors/${id}`, data),
      delete: (id: string) => apiClient.delete(`/vms/service-monitors/${id}`),
      trigger: (id: string) => apiClient.post(`/vms/service-monitors/${id}/trigger`, {}),
      getAvailability: (id: string, params?: Record<string, any>) =>
        apiClient.get(`/vms/service-monitors/${id}/availability`, params),
    },

    // Alert rules
    alertRules: {
      list: () => apiClient.get('/alerts/rules'),
      get: (id: string) => apiClient.get(`/alerts/rules/${id}`),
      create: (data: Record<string, any>) => apiClient.post('/alerts/rules', data),
      update: (id: string, data: Record<string, any>) =>
        apiClient.put(`/alerts/rules/${id}`, data),
      delete: (id: string) => apiClient.delete(`/alerts/rules/${id}`),
      toggle: (id: string, enabled: boolean) =>
        apiClient.post(`/alerts/rules/${id}/toggle`, { enabled }),
    },

    // Alert events
    alertEvents: {
      list: (params?: Record<string, any>) => apiClient.get('/alerts/events', params),
      get: (id: string) => apiClient.get(`/alerts/events/${id}`),
      acknowledge: (id: string, note?: string) =>
        apiClient.post(`/alerts/events/${id}/acknowledge`, { note }),
      resolve: (id: string, note?: string) =>
        apiClient.post(`/alerts/events/${id}/resolve`, { note }),
    },

    // WebSSH
    webssh: {
      createSession: (data: { host_id: string; username?: string; password?: string }) =>
        apiClient.post('/vms/webssh/sessions', data),
      listSessions: () => apiClient.get('/vms/webssh/sessions'),
      listAllSessions: (params?: Record<string, any>) =>
        apiClient.get('/vms/webssh/sessions/all', params),
      getSession: (id: string) => apiClient.get(`/vms/webssh/sessions/${id}`),
      getRecording: (id: string) => apiClient.get(`/vms/webssh/sessions/${id}/playback`),
      closeSession: (id: string) => apiClient.delete(`/vms/webssh/sessions/${id}`),
    },
  },
};
