import { apiClient } from '@/lib/api-client';

// Service Monitor API Service

export interface ServiceMonitor {
  id: string;
  name: string;
  type: 'HTTP' | 'TCP' | 'ICMP';
  target: string;
  interval: number;
  timeout: number;
  enabled: boolean;

  // Probe strategy
  probe_strategy?: 'server' | 'include' | 'exclude' | 'group';
  probe_node_ids?: string;  // JSON string array
  probe_group_name?: string; // Node group name for group strategy

  // HTTP-specific
  http_method?: string;
  http_headers?: string;
  http_body?: string;
  expect_status?: number;
  expect_body?: string;

  // TCP-specific
  tcp_send?: string;
  tcp_expect?: string;

  // Alert configuration
  notify_on_failure: boolean;
  failure_threshold: number;
  recovery_threshold: number;

  // Runtime status
  status?: 'up' | 'down' | 'degraded' | 'unknown';
  last_check_time?: string;
  uptime_24h?: number;
  created_at: string;
  updated_at: string;
}

export interface ServiceProbeResult {
  id: string;
  service_monitor_id: string;
  host_node_id?: string;  // Node that executed the probe (null for server-side probes)
  timestamp: string;
  success: boolean;
  latency: number;
  error_message?: string;
  http_status_code?: number;
  http_response_body?: string;
  tcp_response?: string;
}

export interface ServiceMonitorListParams {
  search?: string;
  type?: string;
  status?: string;
  enabled?: boolean;
  page?: number;
  page_size?: number;
}

export interface ServiceMonitorFormData {
  name: string;
  type: 'HTTP' | 'TCP' | 'ICMP';
  target: string;
  interval: number;
  timeout: number;
  enabled: boolean;

  // Probe strategy
  probe_strategy?: 'server' | 'include' | 'exclude' | 'group';
  probe_node_ids?: string;  // JSON string array
  probe_group_name?: string; // Node group name for group strategy

  // HTTP-specific
  http_method?: string;
  http_headers?: string;
  http_body?: string;
  expect_status?: number;
  expect_body?: string;

  // TCP-specific
  tcp_send?: string;
  tcp_expect?: string;

  // Alert configuration
  notify_on_failure: boolean;
  failure_threshold: number;
  recovery_threshold: number;
}

export interface ServiceMonitorListResponse {
  items: ServiceMonitor[];
  total: number;
}

export const ServiceMonitorService = {
  // List service monitors
  async list(params?: ServiceMonitorListParams): Promise<ServiceMonitorListResponse> {
    const queryParams = new URLSearchParams();
    if (params?.search) queryParams.append('search', params.search);
    if (params?.type) queryParams.append('type', params.type);
    if (params?.status) queryParams.append('status', params.status);
    if (params?.enabled !== undefined) queryParams.append('enabled', String(params.enabled));
    if (params?.page) queryParams.append('page', String(params.page));
    if (params?.page_size) queryParams.append('page_size', String(params.page_size));

    const queryString = queryParams.toString();
    const response = await apiClient.get<{
      data?: {
        items?: ServiceMonitor[];
        total?: number;
      };
    }>(
      `/vms/service-monitors${queryString ? `?${queryString}` : ''}`
    );
    return {
      items: response.data?.items || [],
      total: response.data?.total || 0,
    };
  },

  // Get single service monitor
  async get(id: string): Promise<ServiceMonitor> {
    const response = await apiClient.get<{ data: ServiceMonitor }>(`/vms/service-monitors/${id}`);
    return response.data;
  },

  // Create service monitor
  async create(data: ServiceMonitorFormData): Promise<ServiceMonitor> {
    const response = await apiClient.post<{ data: ServiceMonitor }>('/vms/service-monitors', data);
    return response.data;
  },

  // Update service monitor
  async update(id: string, data: Partial<ServiceMonitorFormData>): Promise<ServiceMonitor> {
    const response = await apiClient.put<{ data: ServiceMonitor }>(`/vms/service-monitors/${id}`, data);
    return response.data;
  },

  // Delete service monitor
  async delete(id: string): Promise<void> {
    await apiClient.delete(`/vms/service-monitors/${id}`);
  },

  // Trigger manual probe
  async triggerProbe(id: string): Promise<ServiceProbeResult> {
    const response = await apiClient.post<{ data: ServiceProbeResult }>(
      `/vms/service-monitors/${id}/trigger`
    );
    return response.data;
  },

  // Get probe history
  async getProbeHistory(
    id: string,
    params?: {
      start_time?: string;
      end_time?: string;
      limit?: number
    }
  ): Promise<ServiceProbeResult[]> {
    const queryParams = new URLSearchParams();
    if (params?.start_time) queryParams.append('start_time', params.start_time);
    if (params?.end_time) queryParams.append('end_time', params.end_time);
    if (params?.limit) queryParams.append('limit', String(params.limit));

    const queryString = queryParams.toString();
    const response = await apiClient.get<{ data?: ServiceProbeResult[] }>(
      `/vms/service-monitors/${id}/history${queryString ? `?${queryString}` : ''}`
    );
    return response.data || [];
  },

  // Get availability stats
  async getAvailabilityStats(
    id: string,
    params?: {
      period?: '1h' | '12h' | '24h' | '7d' | '30d';
    }
  ): Promise<{
    uptime_percentage: number;
    total_checks: number;
    successful_checks: number;
    failed_checks: number;
    avg_latency: number;
    min_latency: number;
    max_latency: number;
  }> {
    const queryParams = new URLSearchParams();
    if (params?.period) queryParams.append('period', params.period);

    const queryString = queryParams.toString();
    const response = await apiClient.get<{
      data: {
        uptime_percentage: number;
        total_checks: number;
        successful_checks: number;
        failed_checks: number;
        avg_latency: number;
        min_latency: number;
        max_latency: number;
      };
    }>(`/vms/service-monitors/${id}/availability${queryString ? `?${queryString}` : ''}`);
    return response.data;
  },

  // Batch enable/disable monitors
  async batchUpdate(
    ids: string[],
    action: 'enable' | 'disable'
  ): Promise<void> {
    await apiClient.post('/vms/service-monitors/batch', {
      ids,
      action,
      enabled: action === 'enable',
    });
  },

  // Export monitors configuration
  async export(format: 'json' | 'yaml' = 'json'): Promise<Blob> {
    return apiClient.getBlob('/vms/service-monitors/export', { format });
  },

  // Import monitors configuration
  async import(file: File): Promise<{ imported: number; failed: number }> {
    const formData = new FormData();
    formData.append('file', file);
    const response = await apiClient.postForm<{ data: { imported: number; failed: number } }>(
      '/vms/service-monitors/import',
      formData
    );
    return response.data;
  },

  // Get overview with 30-day statistics for all service monitors
  async getOverview(): Promise<ServiceOverviewResponse> {
    const response = await apiClient.get<{ data: ServiceOverviewResponse }>(
      '/vms/service-monitors/overview'
    );
    return response.data;
  },

  // Get probe history for a specific host (multi-line chart data)
  async getHostProbeHistory(
    hostId: string,
    hours: number = 24
  ): Promise<ServiceHistoryInfo[]> {
    const response = await apiClient.get<{ data?: ServiceHistoryInfo[] }>(
      `/vms/hosts/${hostId}/probe-history`,
      { hours }
    );
    return response.data || [];
  },

  // Get network topology matrix data
  async getNetworkTopology(hours: number = 1): Promise<NetworkTopologyResponse> {
    const response = await apiClient.get<{ data: NetworkTopologyResponse }>(
      '/vms/service-monitors/topology',
      { hours }
    );
    return response.data;
  },
};

// 30-day service statistics response
export interface ServiceResponseItem {
  service_monitor_id: string;
  service_name: string;

  // 30-day arrays (index 0 = today, 29 = 30 days ago)
  delay: number[];       // Average delay per day (ms)
  up: number[];          // Successful checks per day
  down: number[];        // Failed checks per day

  // Aggregated statistics
  total_up: number;
  total_down: number;
  uptime_percentage: number;

  // Current status (from today)
  current_up: number;
  current_down: number;

  // Status code: Good(>95%) / LowAvailability(80-95%) / Down(<80%)
  status_code: 'Good' | 'LowAvailability' | 'Down' | 'Unknown';
}

export interface ServiceOverviewResponse {
  services: Record<string, ServiceResponseItem>; // Key: service_monitor_id
}

// Host probe history for multi-line chart
export interface ServiceHistoryInfo {
  service_monitor_id: string;
  service_monitor_name: string; // Target name
  service_monitor_type: string; // HTTP, TCP, or ICMP
  host_node_id: string;
  host_node_name?: string;      // Executor name
  timestamps: number[];         // Unix timestamps in milliseconds
  avg_delays: number[];         // Average delays in milliseconds
  uptimes: number[];            // Uptime percentages
}

// Network topology types
export interface NetworkTopologyNode {
  id: string;
  name: string;
  type: 'host' | 'service';
  is_online: boolean;
}

export interface NetworkTopologyEdge {
  source_id: string;
  target_id: string;
  avg_latency: number;
  min_latency: number;
  max_latency: number;
  packet_loss: number;
  success_rate: number;
  probe_count: number;
  last_probe_time: string;
}

export interface NetworkTopologyResponse {
  nodes: NetworkTopologyNode[];
  edges: NetworkTopologyEdge[];
  matrix: Record<string, Record<string, NetworkTopologyEdge>>;
}
