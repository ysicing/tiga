// API types and interfaces

import type { Pod, Event } from 'kubernetes-types/core/v1'
import type { Deployment, StatefulSet, DaemonSet, ReplicaSet } from 'kubernetes-types/apps/v1'
import type { Job, CronJob } from 'kubernetes-types/batch/v1'
import type { Ingress, IngressClass, NetworkPolicy } from 'kubernetes-types/networking/v1'
import type { Service, ConfigMap, Secret, PersistentVolumeClaim, PersistentVolume, ServiceAccount, Namespace, Node, Endpoints } from 'kubernetes-types/core/v1'
import type { CloneSet, AdvancedDaemonSet, CustomResource as K8sCustomResource } from './k8s'

// Re-export CustomResource from k8s.ts for backward compatibility
export type { CustomResource, Pod } from './k8s'
export type { Event } from 'kubernetes-types/core/v1'

// API response wrapper
export interface APIResponse<T> {
  data: T
  message?: string
  success: boolean
}

export interface User {
  id: number
  username: string
  email: string
  display_name?: string
  avatar?: string
  role: 'admin' | 'user' | 'viewer'
  created_at: string
  updated_at: string
  last_login?: string
}

export interface UserItem {
  id: number
  username: string
  email: string
  display_name?: string
  name?: string // Alias for display_name
  role: string
  roles?: string[] // For multi-role support
  created_at: string
  createdAt?: string // Deprecated: use created_at
  updated_at: string
  last_login?: string
  lastLoginAt?: string // Deprecated: use last_login
  enabled?: boolean
  provider?: string
  avatar?: string
  avatar_url?: string // Alias for avatar
}

export interface FetchUserListResponse {
  data: UserItem[]
  users?: UserItem[] // Alias for data
  pagination: {
    page: number
    pageSize: number
    total: number
    totalPages: number
    hasNextPage: boolean
    hasPrevPage: boolean
  }
}

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  access_token: string
  refresh_token: string
  user: User
}

export interface RefreshTokenRequest {
  refresh_token: string
}

export interface RefreshTokenResponse {
  access_token: string
  refresh_token: string
}

export interface RegisterRequest {
  username: string
  email: string
  password: string
  display_name?: string
}

// Role assignment for RBAC
export interface RoleAssignment {
  subjectType: 'user' | 'group' | 'serviceaccount'
  subject: string
}

export interface Role {
  id: number
  name: string
  description?: string
  permissions: string[]
  created_at: string
  updated_at: string
  // RBAC fields
  isSystem?: boolean
  clusters?: string[]
  namespaces?: string[]
  resources?: string[]
  verbs?: string[]
  assignments?: RoleAssignment[]
}

export interface OAuthProvider {
  id: number
  name?: string
  provider: 'google' | 'github'
  client_id: string
  clientId?: string // Deprecated: use client_id
  client_secret: string
  redirect_url: string
  enabled: boolean
  created_at: string
  updated_at: string
  // Additional OAuth fields
  authUrl?: string
  tokenUrl?: string
  userInfoUrl?: string
  scopes?: string
  issuer?: string
}

export interface Alert {
  id: string
  instance_id: string
  instance_name: string
  instance_type: string
  alert_type: string
  severity: 'critical' | 'warning' | 'info'
  message: string
  triggered_at: string
  resolved_at?: string
  status: 'active' | 'resolved' | 'acknowledged'
  metadata?: Record<string, unknown>
}

export interface Metric {
  instance_id: string
  instance_name: string
  metric_type: 'cpu' | 'memory' | 'disk' | 'network' | 'connections'
  value: number
  unit: string
  timestamp: string
  labels?: Record<string, string>
}

// Usage data point for charts
export interface UsageDataPoint {
  timestamp: string
  value: number
}

// Metrics data for resources
export interface MetricsData {
  cpuUsage?: number // Optional since metrics might not always be available
  cpuLimit?: number
  cpuRequest?: number
  memoryUsage?: number // Optional since metrics might not always be available
  memoryLimit?: number
  memoryRequest?: number
}

// Related resources
export interface RelatedResources {
  type: string
  apiVersion?: string
  name: string
  namespace?: string
}

// Resource usage history
export interface ResourceUsageHistory {
  cpu: UsageDataPoint[]
  memory: UsageDataPoint[]
  networkIn?: UsageDataPoint[]
  networkOut?: UsageDataPoint[]
  diskRead?: UsageDataPoint[]
  diskWrite?: UsageDataPoint[]
}

// Image tag info
export interface ImageTagInfo {
  tag: string
  name?: string
  digest: string
  created: string
  timestamp?: string
  size: number
}

// Resource allocation data for overview dashboard
export interface ResourceAllocation {
  cpu: {
    requested: number
    allocatable: number
    limited: number
  }
  memory: {
    requested: number
    allocatable: number
    limited: number
  }
}

export interface OverviewData {
  totalInstances: number
  runningInstances: number
  stoppedInstances: number
  errorInstances: number
  // K8s cluster statistics
  totalNodes?: number
  readyNodes?: number
  totalPods?: number
  runningPods?: number
  totalNamespaces?: number
  totalServices?: number
  // Additional fields
  resource?: ResourceAllocation // K8s cluster resource allocation
  prometheusEnabled?: boolean
  subsystems: {
    type: string
    count: number
    running: number
    stopped: number
    error: number
  }[]
  recentAlerts: Alert[]
  recentMetrics: {
    cpu: Metric[]
    memory: Metric[]
    disk: Metric[]
  }
}

// K8s Cluster types
export interface Cluster {
  id: number
  name: string
  description?: string
  config?: string
  // Support both snake_case (backend) and camelCase (frontend)
  in_cluster: boolean
  inCluster?: boolean // Deprecated: use in_cluster
  is_default: boolean
  isDefault?: boolean // Deprecated: use is_default
  version?: string
  prometheus_url?: string
  prometheusURL?: string // Deprecated: use prometheus_url
  enable: boolean
  enabled?: boolean // Deprecated: use enable
  health_status: 'unknown' | 'healthy' | 'unhealthy'
  last_connected_at?: string
  node_count: number
  pod_count: number
  created_at: string
  updated_at: string
}

// Pod types
export interface PodCondition {
  type: string
  status: string
  lastProbeTime?: string
  lastTransitionTime?: string
  reason?: string
  message?: string
}

export interface ContainerStatus {
  name: string
  state?: {
    waiting?: {
      reason?: string
      message?: string
    }
    running?: {
      startedAt?: string
    }
    terminated?: {
      exitCode?: number
      signal?: number
      reason?: string
      message?: string
      startedAt?: string
      finishedAt?: string
      containerID?: string
    }
  }
  lastState?: {
    waiting?: {
      reason?: string
      message?: string
    }
    running?: {
      startedAt?: string
    }
    terminated?: {
      exitCode?: number
      signal?: number
      reason?: string
      message?: string
      startedAt?: string
      finishedAt?: string
      containerID?: string
    }
  }
  ready: boolean
  restartCount: number
  image: string
  imageID?: string
  containerID?: string
  started?: boolean
}

// Prometheus time-series metrics for pod monitoring
export interface PodMetrics {
  // Time-series data arrays for charts
  cpu: UsageDataPoint[]
  memory: UsageDataPoint[]
  networkIn?: UsageDataPoint[]
  networkOut?: UsageDataPoint[]
  diskRead?: UsageDataPoint[]
  diskWrite?: UsageDataPoint[]
  fallback?: boolean // Indicates if data is from metrics-server (limited historical data)
}

// Metrics-server format (single time point) - used for pod list display
export interface PodMetricsSnapshot {
  metadata: {
    name: string
    namespace: string
    creationTimestamp: string
  }
  timestamp: string
  window: string
  containers: Array<{
    name: string
    usage: {
      cpu: string
      memory: string
    }
  }>
  // Flattened metrics for convenience (computed from containers)
  cpuUsage?: number    // Total CPU usage across all containers (in millicores)
  memoryUsage?: number // Total memory usage across all containers (in bytes)
  cpuLimit?: number
  cpuRequest?: number
  memoryLimit?: number
  memoryRequest?: number
  networkIn?: number
  networkOut?: number
  diskRead?: number
  diskWrite?: number
}

export interface PodWithMetrics {
  metadata: {
    name: string
    namespace: string
    creationTimestamp: string
    uid?: string
    resourceVersion?: string
    labels?: Record<string, string>
    annotations?: Record<string, string>
    generateName?: string // For pod name generation
  }
  spec?: {
    nodeName?: string
    containers?: Array<{
      name: string
      image: string
      ports?: Array<{
        containerPort: number
        protocol?: string
      }>
      env?: Array<{
        name: string
        value?: string
      }>
      resources?: {
        requests?: {
          cpu?: string
          memory?: string
        }
        limits?: {
          cpu?: string
          memory?: string
        }
      }
      volumeMounts?: Array<{
        name: string
        mountPath: string
      }>
    }>
    volumes?: Array<{
      name: string
      configMap?: {
        name: string
      }
      secret?: {
        secretName: string
      }
      persistentVolumeClaim?: {
        claimName: string
      }
    }>
    restartPolicy?: string
    serviceAccountName?: string
  }
  status?: {
    phase?: string
    conditions?: PodCondition[]
    hostIP?: string
    podIP?: string
    podIPs?: Array<{ ip: string }>
    startTime?: string
    containerStatuses?: ContainerStatus[]
    initContainerStatuses?: ContainerStatus[]
    qosClass?: string
    reason?: string
    message?: string
  }
  metrics?: PodMetricsSnapshot // Single time point metrics for list display
}

export interface NodeCondition {
  type: string
  status: string
  lastHeartbeatTime?: string
  lastTransitionTime?: string
  reason?: string
  message?: string
}

export interface NodeWithMetrics {
  metadata: {
    name: string
    creationTimestamp: string
    uid?: string
    resourceVersion?: string
    labels?: Record<string, string>
    annotations?: Record<string, string>
  }
  spec?: {
    podCIDR?: string
    podCIDRs?: string[]
    providerID?: string
    unschedulable?: boolean
    taints?: Array<{
      key: string
      value?: string
      effect: string
    }>
  }
  status?: {
    conditions?: NodeCondition[]
    addresses?: Array<{
      type: string
      address: string
    }>
    allocatable?: {
      cpu?: string
      memory?: string
      pods?: string
      'ephemeral-storage'?: string
    }
    capacity?: {
      cpu?: string
      memory?: string
      pods?: string
      'ephemeral-storage'?: string
    }
    nodeInfo?: {
      machineID?: string
      systemUUID?: string
      bootID?: string
      kernelVersion?: string
      osImage?: string
      containerRuntimeVersion?: string
      kubeletVersion?: string
      kubeProxyVersion?: string
      operatingSystem?: string
      architecture?: string
    }
  }
  metrics?: {
    metadata: {
      name: string
      creationTimestamp: string
    }
    timestamp: string
    window: string
    usage: {
      cpu: string
      memory: string
    }
    // Flattened computed fields for convenience
    cpuUsage?: number
    memoryUsage?: number
  }
}

export interface ResourceHistory {
  id: number
  clusterName: string
  resourceType: string
  resourceName: string
  namespace: string
  operationType: string
  resourceYaml: string
  previousYaml: string
  success: boolean
  errorMessage: string
  operatorId: number
  operator: {
    id: number
    username: string
    email: string
  }
  createdAt: string
  updatedAt: string
}

export interface ResourceHistoryResponse {
  data: ResourceHistory[]
  pagination: {
    page: number
    pageSize: number
    total: number
    totalPages: number
    hasNextPage: boolean
    hasPrevPage: boolean
  }
}

// DevOps Instance types
export interface Instance {
  id: string
  name: string
  display_name?: string
  description?: string
  type:
    | 'mysql'
    | 'postgresql'
    | 'redis'
    | 'minio'
    | 'docker'
    | 'kubernetes'
    | 'k8s'
    | 'caddy'
  status: 'running' | 'stopped' | 'error' | 'unknown' | 'provisioning'
  health: 'healthy' | 'unhealthy' | 'degraded' | 'unknown'
  health_message?: string
  last_health_check?: string
  version?: string
  created_at: string
  updated_at: string
  connection?: Record<string, unknown>
}

export interface InstanceListResponse {
  data: Instance[]
}

export interface SubsystemStats {
  type: string
  count: number
  running: number
  stopped: number
  error: number
}

// System Upgrade Plan types
export interface UpgradePlan extends K8sCustomResource {
  spec: {
    concurrency?: number
    cordon?: boolean
    nodeSelector?: {
      matchLabels?: Record<string, string>
      matchExpressions?: Array<{
        key: string
        operator: string
        values?: string[]
      }>
    }
    tolerations?: Array<{
      key?: string
      operator?: string
      value?: string
      effect?: string
      tolerationSeconds?: number
    }>
    secrets?: Array<{
      name: string
      path: string
    }>
    serviceAccountName?: string
    prepare?: {
      image: string
      command?: string[]
      args?: string[]
      envs?: Array<{
        name: string
        value?: string
        valueFrom?: unknown
      }>
    }
    upgrade: {
      image: string
      command?: string[]
      args?: string[]
      envs?: Array<{
        name: string
        value?: string
        valueFrom?: unknown
      }>
    }
    drain?: {
      enabled?: boolean
      force?: boolean
      timeout?: string
      skipWaitForDeleteTimeout?: number
      ignoreDaemonSets?: boolean
      deleteLocalData?: boolean
    }
    version?: string
    channel?: string
  }
  status?: {
    conditions?: Array<{
      type: string
      status: string
      lastUpdateTime: string
      lastTransitionTime?: string
      reason?: string
      message?: string
    }>
    applying?: Array<{
      name: string
      image: string
      phase: string
    }>
  }
}

// K8s Resource Types
export type ResourceType =
  // Core resources
  | 'pods'
  | 'nodes'
  | 'namespaces'
  | 'services'
  | 'configmaps'
  | 'secrets'
  | 'persistentvolumes'
  | 'persistentvolumeclaims'
  | 'serviceaccounts'
  | 'endpoints'
  | 'events'
  // Workloads
  | 'deployments'
  | 'statefulsets'
  | 'daemonsets'
  | 'replicasets'
  | 'jobs'
  | 'cronjobs'
  // Networking
  | 'ingresses'
  | 'ingressclasses'
  | 'networkpolicies'
  // OpenKruise
  | 'clonesets'
  | 'daemonsets.apps.kruise.io'
  | 'uniteddeployments'
  | 'broadcastjobs'
  | 'advancedcronjobs'
  | 'sidecarsets'
  | 'imagepulljobs'
  | 'containerrecreaterequests'
  | 'resourcedistributions'
  | 'persistentpodstates'
  | 'podprobemarkers'
  | 'podunavailablebudgets'
  // Traefik
  | 'ingressroutes'
  | 'middlewares'
  | 'middlewaretcps'
  | 'ingressroutetcps'
  | 'ingressrouteudps'
  | 'tlsoptions'
  | 'tlsstores'
  | 'traefikservices'
  // Tailscale
  | 'connectors'
  | 'proxyclasses'
  // System Upgrade
  | 'plans'
  // Advanced/Custom
  | 'advanceddaemonsets'
  // Custom resources
  | 'customresourcedefinitions'
  | 'crds'

// Resource Type Map - maps resource types to their corresponding interfaces
export interface ResourceTypeMap {
  // Core resources
  pods: Pod
  nodes: Node
  namespaces: Namespace
  services: Service
  configmaps: ConfigMap
  secrets: Secret
  persistentvolumes: PersistentVolume
  persistentvolumeclaims: PersistentVolumeClaim
  serviceaccounts: ServiceAccount
  endpoints: Endpoints
  events: Event
  // Workloads
  deployments: Deployment
  statefulsets: StatefulSet
  daemonsets: DaemonSet
  replicasets: ReplicaSet
  jobs: Job
  cronjobs: CronJob
  // Networking
  ingresses: Ingress
  ingressclasses: IngressClass
  networkpolicies: NetworkPolicy
  // OpenKruise
  clonesets: CloneSet
  'daemonsets.apps.kruise.io': AdvancedDaemonSet
  uniteddeployments: K8sCustomResource
  broadcastjobs: K8sCustomResource
  advancedcronjobs: K8sCustomResource
  sidecarsets: K8sCustomResource
  imagepulljobs: K8sCustomResource
  containerrecreaterequests: K8sCustomResource
  resourcedistributions: K8sCustomResource
  persistentpodstates: K8sCustomResource
  podprobemarkers: K8sCustomResource
  podunavailablebudgets: K8sCustomResource
  // Traefik
  ingressroutes: K8sCustomResource
  middlewares: K8sCustomResource
  middlewaretcps: K8sCustomResource
  ingressroutetcps: K8sCustomResource
  ingressrouteudps: K8sCustomResource
  tlsoptions: K8sCustomResource
  tlsstores: K8sCustomResource
  traefikservices: K8sCustomResource
  // Tailscale
  connectors: K8sCustomResource
  proxyclasses: K8sCustomResource
  // System Upgrade
  plans: UpgradePlan
  // Advanced/Custom
  advanceddaemonsets: AdvancedDaemonSet
  // Custom resources
  customresourcedefinitions: K8sCustomResource
  crds: K8sCustomResource
}

// Resources Type Map - maps resource types to their list response structure
export type ResourcesTypeMap = {
  [K in keyof ResourceTypeMap]: {
    items: ResourceTypeMap[K][]
  }
}

// Cluster-scoped resources list
export const clusterScopeResources: ResourceType[] = [
  'nodes',
  'namespaces',
  'persistentvolumes',
  'customresourcedefinitions',
  'crds',
  'ingressclasses',
]
