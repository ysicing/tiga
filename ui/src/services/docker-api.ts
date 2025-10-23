// Docker Management API service

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { apiClient } from '@/lib/api-client'

// Type definitions

export interface DockerInstance {
  id: string // UUID
  agent_id: string // UUID
  agent_name: string
  name: string
  host: string
  port: number
  version?: string
  api_version?: string
  os?: string
  architecture?: string
  total_containers?: number
  running_containers?: number
  total_images?: number
  kernel_version?: string
  operating_system?: string
  status: 'online' | 'offline' | 'archived'
  last_seen_at?: string
  created_at: string
  updated_at: string
  archived_at?: string
  labels?: Record<string, string>
  description?: string
}

export interface Container {
  id: string // Container ID
  names: string[]
  image: string
  image_id: string
  command: string
  created: number // Unix timestamp
  state: string // running, exited, paused, restarting, etc.
  status: string // Status text (e.g., "Up 2 hours")
  ports: ContainerPort[]
  labels?: Record<string, string>
  size_rw?: number
  size_root_fs?: number
  mounts?: ContainerMount[]
  network_settings?: ContainerNetworkSettings
}

export interface ContainerPort {
  ip?: string
  private_port: number
  public_port?: number
  type: string // tcp, udp
}

export interface ContainerMount {
  type: string // bind, volume, tmpfs
  name?: string
  source: string
  destination: string
  driver?: string
  mode: string
  rw: boolean
  propagation: string
}

export interface ContainerNetworkSettings {
  networks?: Record<string, ContainerNetwork>
}

export interface ContainerNetwork {
  network_id: string
  endpoint_id: string
  gateway: string
  ip_address: string
  ip_prefix_len: number
  ipv6_gateway?: string
  global_ipv6_address?: string
  global_ipv6_prefix_len?: number
  mac_address: string
  driver_opts?: Record<string, string>
}

export interface ContainerDetails extends Container {
  path: string
  args: string[]
  config?: ContainerConfig
  host_config?: HostConfig
  graph_driver?: GraphDriver
}

export interface ContainerConfig {
  hostname: string
  domainname: string
  user: string
  attach_stdin: boolean
  attach_stdout: boolean
  attach_stderr: boolean
  tty: boolean
  open_stdin: boolean
  stdin_once: boolean
  env: string[]
  cmd: string[]
  image: string
  volumes?: Record<string, any>
  working_dir: string
  entrypoint?: string[]
  labels?: Record<string, string>
}

export interface HostConfig {
  cpu_shares: number
  memory: number
  memory_swap: number
  memory_reservation: number
  kernel_memory: number
  cpu_period: number
  cpu_quota: number
  cpuset_cpus: string
  cpuset_mems: string
  devices?: Device[]
  restart_policy?: RestartPolicy
  network_mode: string
  port_bindings?: Record<string, PortBinding[]>
  privileged: boolean
  readonly_rootfs: boolean
  binds?: string[]
  tmpfs?: Record<string, string>
  ulimits?: Ulimit[]
  log_config?: LogConfig
}

export interface Device {
  path_on_host: string
  path_in_container: string
  cgroup_permissions: string
}

export interface RestartPolicy {
  name: string // no, always, on-failure, unless-stopped
  maximum_retry_count?: number
}

export interface PortBinding {
  host_ip: string
  host_port: string
}

export interface Ulimit {
  name: string
  soft: number
  hard: number
}

export interface LogConfig {
  type: string
  config?: Record<string, string>
}

export interface GraphDriver {
  name: string
  data: Record<string, string>
}

export interface ContainerStats {
  read: string // Timestamp
  preread: string
  cpu_stats: CPUStats
  precpu_stats: CPUStats
  memory_stats: MemoryStats
  blkio_stats: BlkioStats
  pids_stats?: PidsStats
  networks?: Record<string, NetworkStats>
}

export interface CPUStats {
  cpu_usage: CPUUsage
  system_cpu_usage?: number
  online_cpus?: number
  throttling_data?: ThrottlingData
}

export interface CPUUsage {
  total_usage: number
  usage_in_kernelmode: number
  usage_in_usermode: number
  percpu_usage?: number[]
}

export interface ThrottlingData {
  periods: number
  throttled_periods: number
  throttled_time: number
}

export interface MemoryStats {
  usage?: number
  max_usage?: number
  stats?: Record<string, number>
  limit?: number
}

export interface BlkioStats {
  io_service_bytes_recursive?: BlkioStatEntry[]
  io_serviced_recursive?: BlkioStatEntry[]
}

export interface BlkioStatEntry {
  major: number
  minor: number
  op: string
  value: number
}

export interface PidsStats {
  current?: number
  limit?: number
}

export interface NetworkStats {
  rx_bytes: number
  rx_packets: number
  rx_errors: number
  rx_dropped: number
  tx_bytes: number
  tx_packets: number
  tx_errors: number
  tx_dropped: number
}

export interface DockerImage {
  id: string // Image ID
  repo_tags?: string[]
  repo_digests?: string[]
  parent_id: string
  comment: string
  created: number // Unix timestamp
  container: string
  container_config?: ContainerConfig
  docker_version: string
  author: string
  config?: ContainerConfig
  architecture: string
  os: string
  size: number
  virtual_size: number
  graph_driver?: GraphDriver
  root_fs?: RootFS
}

export interface RootFS {
  type: string
  layers?: string[]
}

export interface ImageHistory {
  id: string
  created: number
  created_by: string
  tags?: string[]
  size: number
  comment: string
}

export interface Volume {
  name: string
  driver: string
  mountpoint: string
  created_at?: string
  status?: Record<string, string>
  labels?: Record<string, string>
  scope: string // local, global
  options?: Record<string, string>
  usage_data?: VolumeUsageData
}

export interface VolumeUsageData {
  size: number
  ref_count: number
}

export interface Network {
  id: string
  name: string
  created: string
  scope: string
  driver: string
  enable_ipv6: boolean
  ipam?: IPAMConfig
  internal: boolean
  attachable: boolean
  ingress: boolean
  containers?: Record<string, NetworkContainer>
  options?: Record<string, string>
  labels?: Record<string, string>
}

export interface IPAMConfig {
  driver: string
  config?: IPAMPool[]
  options?: Record<string, string>
}

export interface IPAMPool {
  subnet: string
  ip_range?: string
  gateway?: string
  aux_addresses?: Record<string, string>
}

export interface NetworkContainer {
  name: string
  endpoint_id: string
  mac_address: string
  ipv4_address: string
  ipv6_address: string
}

export interface SystemInfo {
  id: string
  containers: number
  containers_running: number
  containers_paused: number
  containers_stopped: number
  images: number
  driver: string
  driver_status?: DriverStatus[]
  system_status?: DriverStatus[]
  plugins?: PluginsInfo
  memory_limit: boolean
  swap_limit: boolean
  kernel_memory: boolean
  cpu_cfs_period: boolean
  cpu_cfs_quota: boolean
  cpu_shares: boolean
  cpu_set: boolean
  pids_limit: boolean
  ipv4_forwarding: boolean
  bridge_nf_iptables: boolean
  bridge_nf_ip6tables: boolean
  debug: boolean
  nfd: number
  oom_kill_disable: boolean
  ngprocs: number
  system_time: string
  logging_driver: string
  cgroup_driver: string
  cgroup_version: string
  n_events_listener: number
  kernel_version: string
  operating_system: string
  os_version: string
  os_type: string
  architecture: string
  ncpu: number
  mem_total: number
  docker_root_dir: string
  http_proxy?: string
  https_proxy?: string
  no_proxy?: string
  name: string
  labels?: string[]
  experimental_build: boolean
  server_version: string
  runtimes?: Record<string, Runtime>
  default_runtime: string
  security_options?: string[]
  warnings?: string[]
}

export interface DriverStatus {
  name: string
  value: string
}

export interface PluginsInfo {
  volume?: string[]
  network?: string[]
  authorization?: string[]
  log?: string[]
}

export interface Runtime {
  path: string
  runtime_args?: string[]
}

export interface VersionInfo {
  version: string
  api_version: string
  min_api_version: string
  git_commit: string
  go_version: string
  os: string
  arch: string
  kernel_version: string
  build_time: string
  components?: ComponentVersion[]
}

export interface ComponentVersion {
  name: string
  version: string
  details?: Record<string, string>
}

export interface DiskUsage {
  layers_size: number
  images: ImageDiskUsage[]
  containers: ContainerDiskUsage[]
  volumes: VolumeDiskUsage[]
  build_cache?: BuildCacheDiskUsage[]
}

export interface ImageDiskUsage {
  id: string
  parent_id: string
  repo_tags?: string[]
  repo_digests?: string[]
  created: number
  size: number
  shared_size: number
  virtual_size: number
  containers: number
}

export interface ContainerDiskUsage {
  id: string
  names: string[]
  image: string
  image_id: string
  command: string
  created: number
  state: string
  status: string
  size_rw?: number
  size_root_fs?: number
}

export interface VolumeDiskUsage {
  name: string
  driver: string
  mountpoint: string
  labels?: Record<string, string>
  scope: string
  usage_data?: VolumeUsageData
}

export interface BuildCacheDiskUsage {
  id: string
  parent: string
  type: string
  description: string
  in_use: boolean
  shared: boolean
  size: number
  created_at: string
  last_used_at?: string
  usage_count: number
}

export interface DockerEvent {
  type: string // container, image, volume, network, daemon
  action: string // create, start, stop, destroy, pull, push, etc.
  actor: EventActor
  time: number // Unix timestamp
  time_nano: number
  scope: string // local, swarm
}

export interface EventActor {
  id: string
  attributes?: Record<string, string>
}

export interface DockerAuditLog {
  id: number
  resource_type: string // docker_container, docker_image, docker_instance
  resource_id: string // Resource UUID/ID
  action: string // container_start, image_pull, etc.
  user_id?: number
  username?: string
  client_ip: string
  status: 'success' | 'failure'
  error_message?: string
  changes?: string // JSON string of DockerOperationDetails
  created_at: string
}

// API Query Hooks

// Instance Management
export const useDockerInstances = (filters?: {
  page?: number
  page_size?: number
  name?: string
  status?: string
  agent_id?: string
}) => {
  const queryParams = new URLSearchParams()
  if (filters) {
    Object.entries(filters).forEach(([key, value]) => {
      if (value !== undefined && value !== '') {
        queryParams.append(key, String(value))
      }
    })
  }

  return useQuery({
    queryKey: ['docker', 'instances', filters],
    queryFn: async () => {
      const response = await apiClient.get<{
        data: { instances: DockerInstance[]; total: number; page: number; page_size: number }
      }>(`/docker/instances?${queryParams.toString()}`)
      return response
    },
  })
}

export const useDockerInstance = (id: string) => {
  return useQuery({
    queryKey: ['docker', 'instance', id],
    queryFn: () =>
      apiClient.get<{ data: DockerInstance }>(`/docker/instances/${id}`),
    enabled: !!id,
  })
}

export const useCreateDockerInstance = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: Partial<DockerInstance>) =>
      apiClient.post<{ data: DockerInstance }>('/docker/instances', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['docker', 'instances'] })
    },
  })
}

export const useUpdateDockerInstance = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string
      data: Partial<DockerInstance>
    }) =>
      apiClient.put<{ data: DockerInstance }>(
        `/docker/instances/${id}`,
        data
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['docker', 'instances'] })
      queryClient.invalidateQueries({
        queryKey: ['docker', 'instance', variables.id],
      })
    },
  })
}

export const useDeleteDockerInstance = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => apiClient.delete(`/docker/instances/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['docker', 'instances'] })
    },
  })
}

export const useTestDockerConnection = () => {
  return useMutation({
    mutationFn: (id: string) =>
      apiClient.post<{
        data: { status: string; version?: string; message: string }
      }>(`/docker/instances/${id}/test-connection`),
  })
}

// Container Operations
export const useContainers = (instanceId: string, filters?: {
  all?: boolean
  name?: string
  limit?: number
  page?: number
}) => {
  const queryParams = new URLSearchParams()
  if (filters) {
    Object.entries(filters).forEach(([key, value]) => {
      if (value !== undefined && value !== '') {
        queryParams.append(key, String(value))
      }
    })
  }

  return useQuery({
    queryKey: ['docker', 'instance', instanceId, 'containers', filters],
    queryFn: () =>
      apiClient.get<{ data: { containers: Container[]; total: number } }>(
        `/docker/instances/${instanceId}/containers?${queryParams.toString()}`
      ),
    enabled: !!instanceId,
  })
}

export const useContainer = (instanceId: string, containerId: string) => {
  return useQuery({
    queryKey: ['docker', 'instance', instanceId, 'container', containerId],
    queryFn: () =>
      apiClient.get<{ data: ContainerDetails }>(
        `/docker/instances/${instanceId}/containers/${containerId}`
      ),
    enabled: !!instanceId && !!containerId,
  })
}

export const useStartContainer = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ instanceId, containerId }: { instanceId: string; containerId: string }) =>
      apiClient.post(
        `/docker/instances/${instanceId}/containers/start`,
        { container_id: containerId }
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['docker', 'instance', variables.instanceId, 'containers'],
      })
      queryClient.invalidateQueries({
        queryKey: ['docker', 'instance', variables.instanceId, 'container', variables.containerId],
      })
    },
  })
}

export const useStopContainer = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ instanceId, containerId, timeout }: { instanceId: string; containerId: string; timeout?: number }) =>
      apiClient.post(
        `/docker/instances/${instanceId}/containers/stop`,
        { container_id: containerId, timeout }
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['docker', 'instance', variables.instanceId, 'containers'],
      })
      queryClient.invalidateQueries({
        queryKey: ['docker', 'instance', variables.instanceId, 'container', variables.containerId],
      })
    },
  })
}

export const useRestartContainer = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ instanceId, containerId, timeout }: { instanceId: string; containerId: string; timeout?: number }) =>
      apiClient.post(
        `/docker/instances/${instanceId}/containers/restart`,
        { container_id: containerId, timeout }
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['docker', 'instance', variables.instanceId, 'containers'],
      })
    },
  })
}

export const usePauseContainer = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ instanceId, containerId }: { instanceId: string; containerId: string }) =>
      apiClient.post(
        `/docker/instances/${instanceId}/containers/pause`,
        { container_id: containerId }
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['docker', 'instance', variables.instanceId, 'containers'],
      })
    },
  })
}

export const useUnpauseContainer = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ instanceId, containerId }: { instanceId: string; containerId: string }) =>
      apiClient.post(
        `/docker/instances/${instanceId}/containers/unpause`,
        { container_id: containerId }
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['docker', 'instance', variables.instanceId, 'containers'],
      })
    },
  })
}

export const useDeleteContainer = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ instanceId, containerId, force, removeVolumes }: {
      instanceId: string
      containerId: string
      force?: boolean
      removeVolumes?: boolean
    }) =>
      apiClient.post(
        `/docker/instances/${instanceId}/containers/delete`,
        { container_id: containerId, force, remove_volumes: removeVolumes }
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['docker', 'instance', variables.instanceId, 'containers'],
      })
    },
  })
}

// Container Logs (historical, not streaming)
export const useContainerLogs = (
  instanceId: string,
  containerId: string,
  options?: {
    tail?: number
    since?: string | number
    timestamps?: boolean
    stdout?: boolean
    stderr?: boolean
  }
) => {
  const queryParams = new URLSearchParams()
  if (options) {
    if (options.tail !== undefined) queryParams.append('tail', options.tail.toString())
    if (options.since !== undefined) queryParams.append('since', String(options.since))
    if (options.timestamps !== undefined) queryParams.append('timestamps', options.timestamps.toString())
    if (options.stdout !== undefined) queryParams.append('stdout', options.stdout.toString())
    if (options.stderr !== undefined) queryParams.append('stderr', options.stderr.toString())
  }

  return useQuery({
    queryKey: ['docker', 'instance', instanceId, 'container', containerId, 'logs', options],
    queryFn: () =>
      apiClient.get<{ data: any[] }>(
        `/docker/instances/${instanceId}/containers/${containerId}/logs?${queryParams.toString()}`
      ),
    enabled: !!instanceId && !!containerId,
  })
}

// Container Stats (single query, not streaming)
export const useContainerStats = (instanceId: string, containerId: string) => {
  return useQuery({
    queryKey: ['docker', 'instance', instanceId, 'container', containerId, 'stats'],
    queryFn: () =>
      apiClient.get<{ data: ContainerStats }>(
        `/docker/instances/${instanceId}/containers/${containerId}/stats`
      ),
    enabled: !!instanceId && !!containerId,
    refetchInterval: 5000, // Refresh every 5 seconds
  })
}

// Image Operations
export const useImages = (instanceId: string, filters?: {
  all?: boolean
  filter?: string
}) => {
  const queryParams = new URLSearchParams()
  if (filters) {
    Object.entries(filters).forEach(([key, value]) => {
      if (value !== undefined && value !== '') {
        queryParams.append(key, String(value))
      }
    })
  }

  return useQuery({
    queryKey: ['docker', 'instance', instanceId, 'images', filters],
    queryFn: () =>
      apiClient.get<{ data: { images: DockerImage[]; total: number } }>(
        `/docker/instances/${instanceId}/images?${queryParams.toString()}`
      ),
    enabled: !!instanceId,
  })
}

export const useImage = (instanceId: string, imageId: string) => {
  return useQuery({
    queryKey: ['docker', 'instance', instanceId, 'image', imageId],
    queryFn: () =>
      apiClient.get<{ data: DockerImage }>(
        `/docker/instances/${instanceId}/images/${imageId}`
      ),
    enabled: !!instanceId && !!imageId,
  })
}

export const useDeleteImage = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ instanceId, imageId, force, noPrune }: {
      instanceId: string
      imageId: string
      force?: boolean
      noPrune?: boolean
    }) =>
      apiClient.post(
        `/docker/instances/${instanceId}/images/delete`,
        { image_id: imageId, force, no_prune: noPrune }
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['docker', 'instance', variables.instanceId, 'images'],
      })
    },
  })
}

export const useTagImage = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ instanceId, source, target }: {
      instanceId: string
      source: string
      target: string
    }) =>
      apiClient.post(
        `/docker/instances/${instanceId}/images/tag`,
        { source, target }
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['docker', 'instance', variables.instanceId, 'images'],
      })
    },
  })
}

// Volume Operations
export const useVolumes = (instanceId: string) => {
  return useQuery({
    queryKey: ['docker', 'instance', instanceId, 'volumes'],
    queryFn: () =>
      apiClient.get<{ data: { volumes: Volume[] } }>(
        `/docker/instances/${instanceId}/volumes`
      ),
    enabled: !!instanceId,
  })
}

export const useVolume = (instanceId: string, volumeName: string) => {
  return useQuery({
    queryKey: ['docker', 'instance', instanceId, 'volume', volumeName],
    queryFn: () =>
      apiClient.get<{ data: Volume }>(
        `/docker/instances/${instanceId}/volumes/${volumeName}`
      ),
    enabled: !!instanceId && !!volumeName,
  })
}

export const useCreateVolume = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ instanceId, data }: {
      instanceId: string
      data: {
        name: string
        driver?: string
        driver_opts?: Record<string, string>
        labels?: Record<string, string>
      }
    }) =>
      apiClient.post<{ data: Volume }>(
        `/docker/instances/${instanceId}/volumes`,
        data
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['docker', 'instance', variables.instanceId, 'volumes'],
      })
    },
  })
}

export const useDeleteVolume = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ instanceId, name, force }: {
      instanceId: string
      name: string
      force?: boolean
    }) =>
      apiClient.post(
        `/docker/instances/${instanceId}/volumes/delete`,
        { name, force }
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['docker', 'instance', variables.instanceId, 'volumes'],
      })
    },
  })
}

export const usePruneVolumes = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ instanceId, filters }: {
      instanceId: string
      filters?: Record<string, string>
    }) =>
      apiClient.post(
        `/docker/instances/${instanceId}/volumes/prune`,
        { filters }
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['docker', 'instance', variables.instanceId, 'volumes'],
      })
    },
  })
}

// Network Operations
export const useNetworks = (instanceId: string, filters?: string) => {
  const queryParams = filters ? `?filters=${encodeURIComponent(filters)}` : ''

  return useQuery({
    queryKey: ['docker', 'instance', instanceId, 'networks', filters],
    queryFn: () =>
      apiClient.get<{ data: { networks: Network[] } }>(
        `/docker/instances/${instanceId}/networks${queryParams}`
      ),
    enabled: !!instanceId,
  })
}

export const useNetwork = (instanceId: string, networkId: string) => {
  return useQuery({
    queryKey: ['docker', 'instance', instanceId, 'network', networkId],
    queryFn: () =>
      apiClient.get<{ data: Network }>(
        `/docker/instances/${instanceId}/networks/${networkId}`
      ),
    enabled: !!instanceId && !!networkId,
  })
}

export const useCreateNetwork = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ instanceId, data }: {
      instanceId: string
      data: {
        name: string
        check_duplicate?: boolean
        driver?: string
        internal?: boolean
        attachable?: boolean
        ingress?: boolean
        enable_ipv6?: boolean
        ipam?: IPAMConfig
        options?: Record<string, string>
        labels?: Record<string, string>
      }
    }) =>
      apiClient.post<{ data: Network }>(
        `/docker/instances/${instanceId}/networks`,
        data
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['docker', 'instance', variables.instanceId, 'networks'],
      })
    },
  })
}

export const useDeleteNetwork = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ instanceId, networkId }: {
      instanceId: string
      networkId: string
    }) =>
      apiClient.post(
        `/docker/instances/${instanceId}/networks/delete`,
        { network_id: networkId }
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['docker', 'instance', variables.instanceId, 'networks'],
      })
    },
  })
}

export const useConnectNetwork = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ instanceId, networkId, containerId, endpointConfig }: {
      instanceId: string
      networkId: string
      containerId: string
      endpointConfig?: {
        ipam_config?: Record<string, string>
        links?: string[]
        aliases?: string[]
      }
    }) =>
      apiClient.post(
        `/docker/instances/${instanceId}/networks/connect`,
        { network_id: networkId, container_id: containerId, endpoint_config: endpointConfig }
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['docker', 'instance', variables.instanceId, 'networks'],
      })
      queryClient.invalidateQueries({
        queryKey: ['docker', 'instance', variables.instanceId, 'containers'],
      })
    },
  })
}

export const useDisconnectNetwork = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ instanceId, networkId, containerId, force }: {
      instanceId: string
      networkId: string
      containerId: string
      force?: boolean
    }) =>
      apiClient.post(
        `/docker/instances/${instanceId}/networks/disconnect`,
        { network_id: networkId, container_id: containerId, force }
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['docker', 'instance', variables.instanceId, 'networks'],
      })
      queryClient.invalidateQueries({
        queryKey: ['docker', 'instance', variables.instanceId, 'containers'],
      })
    },
  })
}

// System Operations
export const useSystemInfo = (instanceId: string) => {
  return useQuery({
    queryKey: ['docker', 'instance', instanceId, 'system', 'info'],
    queryFn: () =>
      apiClient.get<{ data: SystemInfo }>(
        `/docker/instances/${instanceId}/system/info`
      ),
    enabled: !!instanceId,
  })
}

export const useDockerVersion = (instanceId: string) => {
  return useQuery({
    queryKey: ['docker', 'instance', instanceId, 'system', 'version'],
    queryFn: () =>
      apiClient.get<{ data: VersionInfo }>(
        `/docker/instances/${instanceId}/system/version`
      ),
    enabled: !!instanceId,
  })
}

export const useDiskUsage = (instanceId: string) => {
  return useQuery({
    queryKey: ['docker', 'instance', instanceId, 'system', 'disk-usage'],
    queryFn: () =>
      apiClient.get<{ data: DiskUsage }>(
        `/docker/instances/${instanceId}/system/disk-usage`
      ),
    enabled: !!instanceId,
  })
}

export const usePingDocker = () => {
  return useMutation({
    mutationFn: (instanceId: string) =>
      apiClient.get<{ data: { message: string } }>(
        `/docker/instances/${instanceId}/system/ping`
      ),
  })
}

// Audit Logs
export interface DockerAuditLogFilters {
  instance_id?: string
  user?: string
  action?: string
  resource_type?: string
  start_time?: string
  end_time?: string
  success?: boolean
  page?: number
  page_size?: number
}

export const useDockerAuditLogs = (filters: DockerAuditLogFilters) => {
  const queryParams = new URLSearchParams()
  Object.entries(filters).forEach(([key, value]) => {
    if (value !== undefined && value !== '') {
      queryParams.append(key, String(value))
    }
  })

  return useQuery({
    queryKey: ['docker', 'audit-logs', filters],
    queryFn: () =>
      apiClient.get<{
        data: {
          logs: DockerAuditLog[]
          total: number
          page: number
          page_size: number
        }
      }>(`/docker/audit-logs?${queryParams.toString()}`),
  })
}
