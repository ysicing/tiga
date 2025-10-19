// Kubernetes Cluster Management API service

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { apiClient } from '@/lib/api-client'

// ==================== Type Definitions ====================

export interface Cluster {
  id: string // UUID
  name: string
  description?: string
  config?: string // Kubeconfig YAML
  in_cluster: boolean
  is_default: boolean
  prometheus_url?: string
  enable: boolean
  health_status: 'unknown' | 'healthy' | 'unhealthy'
  last_connected_at?: string
  node_count: number
  pod_count: number
  created_at: string
  updated_at: string
}

export interface ResourceHistory {
  id: string // UUID
  cluster_id: string
  resource_type: string
  resource_name: string
  namespace: string
  api_group?: string
  api_version?: string
  operation_type: 'create' | 'update' | 'delete' | 'apply'
  resource_yaml: string
  previous_yaml?: string
  success: boolean
  error_message?: string
  operator_id: string
  operator_name: string
  created_at: string
}

export interface CRDInfo {
  group: string
  version: string
  kind: string
  plural: string
  singular: string
  namespaced: boolean
}

export interface CRDResource {
  apiVersion: string
  kind: string
  metadata: {
    name: string
    namespace?: string
    uid?: string
    resourceVersion?: string
    creationTimestamp?: string
    labels?: Record<string, string>
    annotations?: Record<string, string>
  }
  spec?: Record<string, unknown>
  status?: Record<string, unknown>
}

export interface CloneSet {
  name: string
  namespace: string
  replicas: number
  ready_replicas: number
  updated_replicas: number
  available_replicas: number
  created_at: string
}

export interface DaemonSet {
  name: string
  namespace: string
  desired_number_scheduled: number
  current_number_scheduled: number
  number_ready: number
  created_at: string
}

export interface StatefulSet {
  name: string
  namespace: string
  replicas: number
  ready_replicas: number
  current_replicas: number
  updated_replicas: number
  created_at: string
}

// ==================== Query Filters ====================

export interface ResourceHistoryFilters {
  resource_type?: string
  resource_name?: string
  namespace?: string
  api_group?: string
  api_version?: string
  operation_type?: string
  operator_id?: string
  success?: boolean
  start_time?: string
  end_time?: string
  page?: number
  page_size?: number
}

export interface CRDResourceQuery {
  group: string
  version: string
  resource: string
  namespace?: string
  name?: string
}

// ==================== Cluster Management ====================

export const useClusters = () => {
  return useQuery({
    queryKey: ['k8s', 'clusters'],
    queryFn: async () => {
      const response = await apiClient.get<{ data: { clusters: Cluster[]; total: number } }>(
        '/k8s/clusters'
      )
      return response
    },
  })
}

export const useCluster = (id: string) => {
  return useQuery({
    queryKey: ['k8s', 'cluster', id],
    queryFn: () => apiClient.get<{ data: Cluster }>(`/k8s/clusters/${id}`),
    enabled: !!id,
  })
}

export const useCreateCluster = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: Partial<Cluster>) =>
      apiClient.post<{ data: Cluster }>('/k8s/clusters', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['k8s', 'clusters'] })
    },
  })
}

export const useUpdateCluster = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Cluster> }) =>
      apiClient.put<{ data: Cluster }>(`/k8s/clusters/${id}`, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['k8s', 'clusters'] })
      queryClient.invalidateQueries({
        queryKey: ['k8s', 'cluster', variables.id],
      })
    },
  })
}

export const useDeleteCluster = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => apiClient.delete(`/k8s/clusters/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['k8s', 'clusters'] })
    },
  })
}

export const useTestClusterConnection = () => {
  return useMutation({
    mutationFn: (id: string) =>
      apiClient.post<{
        data: { status: string; version?: string; message: string }
      }>(`/k8s/clusters/${id}/test`),
  })
}

export const useSetDefaultCluster = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) =>
      apiClient.post<{ message: string }>(`/k8s/clusters/${id}/set-default`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['k8s', 'clusters'] })
    },
  })
}

// ==================== Prometheus Discovery ====================

export const useRediscoverPrometheus = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) =>
      apiClient.post<{ message: string; url?: string }>(
        `/k8s/clusters/${id}/prometheus/rediscover`
      ),
    onSuccess: (_, clusterId) => {
      queryClient.invalidateQueries({
        queryKey: ['k8s', 'cluster', clusterId],
      })
    },
  })
}

// ==================== CRD Detection ====================

export const useDetectCRDs = (clusterId: string) => {
  return useQuery({
    queryKey: ['k8s', 'cluster', clusterId, 'crds'],
    queryFn: () =>
      apiClient.get<{ data: { detected_crds: CRDInfo[] } }>(
        `/k8s/clusters/${clusterId}/crds`
      ),
    enabled: !!clusterId,
  })
}

// ==================== CloneSet Operations ====================

export const useCloneSets = (clusterId: string, namespace?: string) => {
  const params = namespace ? `?namespace=${namespace}` : ''

  return useQuery({
    queryKey: ['k8s', 'cluster', clusterId, 'clonesets', namespace],
    queryFn: () =>
      apiClient.get<{ data: CloneSet[] }>(
        `/k8s/clusters/${clusterId}/clonesets${params}`
      ),
    enabled: !!clusterId,
  })
}

export const useScaleCloneSet = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      clusterId,
      namespace,
      name,
      replicas,
    }: {
      clusterId: string
      namespace: string
      name: string
      replicas: number
    }) =>
      apiClient.post<{ message: string }>(
        `/k8s/clusters/${clusterId}/clonesets/${namespace}/${name}/scale`,
        { replicas }
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['k8s', 'cluster', variables.clusterId, 'clonesets'],
      })
    },
  })
}

export const useRestartCloneSet = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      clusterId,
      namespace,
      name,
    }: {
      clusterId: string
      namespace: string
      name: string
    }) =>
      apiClient.post<{ message: string }>(
        `/k8s/clusters/${clusterId}/clonesets/${namespace}/${name}/restart`
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['k8s', 'cluster', variables.clusterId, 'clonesets'],
      })
    },
  })
}

// ==================== DaemonSet Operations ====================

export const useDaemonSets = (clusterId: string, namespace?: string) => {
  const params = namespace ? `?namespace=${namespace}` : ''

  return useQuery({
    queryKey: ['k8s', 'cluster', clusterId, 'daemonsets', namespace],
    queryFn: () =>
      apiClient.get<{ data: DaemonSet[] }>(
        `/k8s/clusters/${clusterId}/daemonsets${params}`
      ),
    enabled: !!clusterId,
  })
}

export const useRestartDaemonSet = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      clusterId,
      namespace,
      name,
    }: {
      clusterId: string
      namespace: string
      name: string
    }) =>
      apiClient.post<{ message: string }>(
        `/k8s/clusters/${clusterId}/daemonsets/${namespace}/${name}/restart`
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['k8s', 'cluster', variables.clusterId, 'daemonsets'],
      })
    },
  })
}

// ==================== StatefulSet Operations ====================

export const useStatefulSets = (clusterId: string, namespace?: string) => {
  const params = namespace ? `?namespace=${namespace}` : ''

  return useQuery({
    queryKey: ['k8s', 'cluster', clusterId, 'statefulsets', namespace],
    queryFn: () =>
      apiClient.get<{ data: StatefulSet[] }>(
        `/k8s/clusters/${clusterId}/statefulsets${params}`
      ),
    enabled: !!clusterId,
  })
}

export const useScaleStatefulSet = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      clusterId,
      namespace,
      name,
      replicas,
    }: {
      clusterId: string
      namespace: string
      name: string
      replicas: number
    }) =>
      apiClient.post<{ message: string }>(
        `/k8s/clusters/${clusterId}/statefulsets/${namespace}/${name}/scale`,
        { replicas }
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['k8s', 'cluster', variables.clusterId, 'statefulsets'],
      })
    },
  })
}

export const useRestartStatefulSet = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      clusterId,
      namespace,
      name,
    }: {
      clusterId: string
      namespace: string
      name: string
    }) =>
      apiClient.post<{ message: string }>(
        `/k8s/clusters/${clusterId}/statefulsets/${namespace}/${name}/restart`
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['k8s', 'cluster', variables.clusterId, 'statefulsets'],
      })
    },
  })
}

// ==================== Resource History ====================

export const useResourceHistory = (
  clusterId: string,
  filters?: ResourceHistoryFilters
) => {
  const queryParams = new URLSearchParams()
  if (filters) {
    Object.entries(filters).forEach(([key, value]) => {
      if (value !== undefined && value !== '') {
        queryParams.append(key, String(value))
      }
    })
  }

  return useQuery({
    queryKey: ['k8s', 'cluster', clusterId, 'resource-history', filters],
    queryFn: () =>
      apiClient.get<{
        data: ResourceHistory[]
        total: number
        page: number
        page_size: number
      }>(`/k8s/clusters/${clusterId}/resource-history?${queryParams.toString()}`),
    enabled: !!clusterId,
  })
}

export const useResourceHistoryDetail = (
  clusterId: string,
  historyId: string
) => {
  return useQuery({
    queryKey: ['k8s', 'cluster', clusterId, 'resource-history', historyId],
    queryFn: () =>
      apiClient.get<{ data: ResourceHistory }>(
        `/k8s/clusters/${clusterId}/resource-history/${historyId}`
      ),
    enabled: !!clusterId && !!historyId,
  })
}

export const useDeleteResourceHistory = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      clusterId,
      historyId,
    }: {
      clusterId: string
      historyId: string
    }) =>
      apiClient.delete(
        `/k8s/clusters/${clusterId}/resource-history/${historyId}`
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['k8s', 'cluster', variables.clusterId, 'resource-history'],
      })
    },
  })
}

// ==================== Generic CRD CRUD ====================

export const useCRDResources = (
  clusterId: string,
  query: CRDResourceQuery
) => {
  const queryParams = new URLSearchParams()
  Object.entries(query).forEach(([key, value]) => {
    if (value) {
      queryParams.append(key, value)
    }
  })

  return useQuery({
    queryKey: ['k8s', 'cluster', clusterId, 'crd-resources', query],
    queryFn: () =>
      apiClient.get<{ data: CRDResource[] }>(
        `/k8s/clusters/${clusterId}/crd-resources?${queryParams.toString()}`
      ),
    enabled: !!clusterId && !!query.group && !!query.version && !!query.resource,
  })
}

export const useCRDResource = (clusterId: string, resourceId: string) => {
  return useQuery({
    queryKey: ['k8s', 'cluster', clusterId, 'crd-resource', resourceId],
    queryFn: () =>
      apiClient.get<{ data: CRDResource }>(
        `/k8s/clusters/${clusterId}/crd-resources/${resourceId}`
      ),
    enabled: !!clusterId && !!resourceId,
  })
}

export const useCreateCRDResource = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      clusterId,
      data,
    }: {
      clusterId: string
      data: CRDResource
    }) =>
      apiClient.post<{ data: CRDResource }>(
        `/k8s/clusters/${clusterId}/crd-resources`,
        data
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['k8s', 'cluster', variables.clusterId, 'crd-resources'],
      })
    },
  })
}

export const useUpdateCRDResource = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      clusterId,
      resourceId,
      data,
    }: {
      clusterId: string
      resourceId: string
      data: CRDResource
    }) =>
      apiClient.put<{ data: CRDResource }>(
        `/k8s/clusters/${clusterId}/crd-resources/${resourceId}`,
        data
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['k8s', 'cluster', variables.clusterId, 'crd-resources'],
      })
      queryClient.invalidateQueries({
        queryKey: [
          'k8s',
          'cluster',
          variables.clusterId,
          'crd-resource',
          variables.resourceId,
        ],
      })
    },
  })
}

export const useDeleteCRDResource = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      clusterId,
      resourceId,
    }: {
      clusterId: string
      resourceId: string
    }) =>
      apiClient.delete(
        `/k8s/clusters/${clusterId}/crd-resources/${resourceId}`
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['k8s', 'cluster', variables.clusterId, 'crd-resources'],
      })
    },
  })
}

export const useApplyCRDResourceYAML = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ clusterId, yaml }: { clusterId: string; yaml: string }) =>
      apiClient.post<{
        data: { message: string; kind: string; name: string; namespace?: string }
      }>(`/k8s/clusters/${clusterId}/crd-resources/apply`, { yaml }),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['k8s', 'cluster', variables.clusterId, 'crd-resources'],
      })
      queryClient.invalidateQueries({
        queryKey: ['k8s', 'cluster', variables.clusterId, 'resource-history'],
      })
    },
  })
}

// ==================== WebSocket Terminal ====================

/**
 * Connect to pod terminal via WebSocket
 * @param clusterId Cluster UUID
 * @param namespace Pod namespace
 * @param podName Pod name
 * @param container Container name (optional)
 * @returns WebSocket URL for terminal connection
 */
export const getTerminalWebSocketURL = (
  clusterId: string,
  namespace: string,
  podName: string,
  container?: string
): string => {
  const baseURL = import.meta.env.VITE_API_BASE_URL || ''
  const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = baseURL
    ? new URL(baseURL).host
    : window.location.host

  const params = new URLSearchParams({
    namespace,
    pod: podName,
  })
  if (container) {
    params.append('container', container)
  }

  return `${wsProtocol}//${host}/api/v1/k8s/clusters/${clusterId}/pods/terminal?${params.toString()}`
}

// ==================== OpenKruise Advanced Features ====================

export interface OpenKruiseWorkload {
  name: string
  kind: string
  apiVersion: string
  available: boolean
  count: number
  description: string
}

export interface OpenKruiseStatus {
  installed: boolean
  version?: string
  workloads: OpenKruiseWorkload[]
}

export const useOpenKruiseStatus = () => {
  return useQuery({
    queryKey: ['k8s', 'openkruise-status'],
    queryFn: async () => {
      const response = await apiClient.get<{ data: OpenKruiseStatus }>('/k8s/openkruise/status')
      return response.data
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    refetchInterval: 5 * 60 * 1000, // Refetch every 5 minutes
  })
}

