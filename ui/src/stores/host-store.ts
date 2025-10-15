import { create } from 'zustand'
import { devtools } from 'zustand/middleware'

// Types
export interface HostInfo {
  platform: string
  platform_version: string
  kernel: string
  arch: string
  virtualization?: string
  cpu_model: string
  cpu_cores: number
  cpu_threads: number
  mem_total: number
  disk_total: number
  agent_version: string
  boot_time: number
  // SSH configuration (reported by Agent)
  ssh_enabled: boolean
  ssh_port: number
  ssh_user: string
}

export interface HostState {
  timestamp: string
  cpu_usage: number
  load_1: number
  load_5: number
  load_15: number
  mem_used: number
  mem_usage: number
  swap_used: number
  disk_used: number
  disk_usage: number
  net_in_transfer: number
  net_out_transfer: number
  net_in_speed: number
  net_out_speed: number
  tcp_conn_count: number
  udp_conn_count: number
  process_count: number
  uptime: number
}

export interface Host {
  id: string
  name: string
  note?: string
  public_note?: string
  display_index: number
  hide_for_guest: boolean

  // Billing and expiry information
  cost: number
  renewal_type: 'monthly' | 'yearly'
  purchase_date?: string // ISO date string
  expiry_date?: string // ISO date string
  auto_renew: boolean
  traffic_limit: number // GB, 0 means unlimited
  traffic_used: number // GB

  // Group name (simple string grouping)
  group_name: string

  online: boolean
  last_active?: string
  created_at: string
  updated_at: string
  host_info?: HostInfo
  current_state?: HostState
}

// Host groups are now just string names (no separate table)
export type HostGroupName = string

export interface ServiceMonitor {
  id: string
  name: string
  type: 'HTTP' | 'TCP' | 'ICMP'
  target: string
  interval: number
  timeout: number
  enabled: boolean

  // Probe node selection strategy
  probe_strategy?: 'server' | 'include' | 'exclude' | 'group'
  probe_node_ids?: string // JSON string array of node UUIDs
  probe_group_name?: string // Node group name for group strategy

  // HTTP-specific
  http_method?: string
  http_headers?: string
  http_body?: string
  expect_status?: number
  expect_body?: string

  // TCP-specific
  tcp_send?: string
  tcp_expect?: string

  created_at: string
  updated_at?: string
}

// WebSocket subscription state
export interface WSSubscription {
  hostIds: string[]
  connected: boolean
  reconnecting: boolean
  error?: string
}

// Store state
interface HostStoreState {
  // Host data
  hosts: Host[]
  selectedHost: Host | null
  hostStates: Map<string, HostState> // hostId -> latest state
  hostGroupNames: string[] // List of unique group names
  serviceMonitors: ServiceMonitor[]

  // Loading states
  loading: boolean
  error: string | null

  // WebSocket state
  wsSubscription: WSSubscription

  // Actions
  setHosts: (hosts: Host[]) => void
  addHost: (host: Host) => void
  updateHost: (id: string, host: Partial<Host>) => void
  removeHost: (id: string) => void
  selectHost: (host: Host | null) => void

  // State updates
  updateHostState: (hostId: string, state: HostState) => void
  updateHostStates: (states: Map<string, HostState>) => void

  // Group names
  setHostGroupNames: (names: string[]) => void

  // Monitors
  setServiceMonitors: (monitors: ServiceMonitor[]) => void
  addServiceMonitor: (monitor: ServiceMonitor) => void
  updateServiceMonitor: (id: string, monitor: Partial<ServiceMonitor>) => void
  removeServiceMonitor: (id: string) => void

  // WebSocket
  setWSSubscription: (subscription: Partial<WSSubscription>) => void
  subscribeToHosts: (hostIds: string[]) => void
  unsubscribeFromHosts: () => void

  // Loading/Error
  setLoading: (loading: boolean) => void
  setError: (error: string | null) => void

  // Reset
  reset: () => void
}

// Initial state
const initialState = {
  hosts: [],
  selectedHost: null,
  hostStates: new Map<string, HostState>(),
  hostGroupNames: [],
  serviceMonitors: [],
  loading: false,
  error: null,
  wsSubscription: {
    hostIds: [],
    connected: false,
    reconnecting: false,
  },
}

// Create store
export const useHostStore = create<HostStoreState>()(
  devtools(
    (set) => ({
      ...initialState,

      // Host actions
      setHosts: (hosts) => set({ hosts }),

      addHost: (host) => set((state) => ({ hosts: [...state.hosts, host] })),

      updateHost: (id, hostUpdate) =>
        set((state) => ({
          hosts: state.hosts.map((h) =>
            h.id === id ? { ...h, ...hostUpdate } : h
          ),
          selectedHost:
            state.selectedHost?.id === id
              ? { ...state.selectedHost, ...hostUpdate }
              : state.selectedHost,
        })),

      removeHost: (id) =>
        set((state) => ({
          hosts: state.hosts.filter((h) => h.id !== id),
          selectedHost:
            state.selectedHost?.id === id ? null : state.selectedHost,
        })),

      selectHost: (host) => set({ selectedHost: host }),

      // State updates
      updateHostState: (hostId, state) =>
        set((store) => {
          const newStates = new Map(store.hostStates)
          newStates.set(hostId, state)

          // Also update current_state in hosts array
          const hosts = store.hosts.map((h) =>
            h.id === hostId ? { ...h, current_state: state, online: true } : h
          )

          return { hostStates: newStates, hosts }
        }),

      updateHostStates: (states) => set({ hostStates: states }),

      // Group names
      setHostGroupNames: (names) => set({ hostGroupNames: names }),

      // Monitors
      setServiceMonitors: (monitors) => set({ serviceMonitors: monitors }),

      addServiceMonitor: (monitor) =>
        set((state) => ({
          serviceMonitors: [...state.serviceMonitors, monitor],
        })),

      updateServiceMonitor: (id, monitorUpdate) =>
        set((state) => ({
          serviceMonitors: state.serviceMonitors.map((m) =>
            m.id === id ? { ...m, ...monitorUpdate } : m
          ),
        })),

      removeServiceMonitor: (id) =>
        set((state) => ({
          serviceMonitors: state.serviceMonitors.filter((m) => m.id !== id),
        })),

      // WebSocket
      setWSSubscription: (subscription) =>
        set((state) => ({
          wsSubscription: { ...state.wsSubscription, ...subscription },
        })),

      subscribeToHosts: (hostIds) =>
        set((state) => ({
          wsSubscription: {
            ...state.wsSubscription,
            hostIds,
          },
        })),

      unsubscribeFromHosts: () =>
        set((state) => ({
          wsSubscription: {
            ...state.wsSubscription,
            hostIds: [],
            connected: false,
          },
        })),

      // Loading/Error
      setLoading: (loading) => set({ loading }),
      setError: (error) => set({ error }),

      // Reset
      reset: () => set(initialState),
    }),
    { name: 'host-store' }
  )
)

// Selectors
export const selectOnlineHosts = (state: HostStoreState) =>
  state.hosts.filter((h) => h.online)

export const selectOfflineHosts = (state: HostStoreState) =>
  state.hosts.filter((h) => !h.online)

export const selectHostsByGroupName =
  (groupName: string) => (state: HostStoreState) =>
    state.hosts.filter((h) => h.group_name === groupName)

export const selectEnabledMonitors = (state: HostStoreState) =>
  state.serviceMonitors.filter((m) => m.enabled)
