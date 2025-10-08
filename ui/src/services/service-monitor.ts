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

export const ServiceMonitorService = {
  // List service monitors
  async list(params?: ServiceMonitorListParams): Promise<ServiceMonitor[]> {
    const queryParams = new URLSearchParams();
    if (params?.search) queryParams.append('search', params.search);
    if (params?.type) queryParams.append('type', params.type);
    if (params?.status) queryParams.append('status', params.status);
    if (params?.enabled !== undefined) queryParams.append('enabled', String(params.enabled));
    if (params?.page) queryParams.append('page', String(params.page));
    if (params?.page_size) queryParams.append('page_size', String(params.page_size));

    const response = await fetch(`/api/v1/service-monitors?${queryParams.toString()}`);
    if (!response.ok) {
      throw new Error('Failed to fetch service monitors');
    }
    const data = await response.json();
    return data.data || [];
  },

  // Get single service monitor
  async get(id: string): Promise<ServiceMonitor> {
    const response = await fetch(`/api/v1/service-monitors/${id}`);
    if (!response.ok) {
      throw new Error('Failed to fetch service monitor');
    }
    const data = await response.json();
    return data.data;
  },

  // Create service monitor
  async create(data: ServiceMonitorFormData): Promise<ServiceMonitor> {
    const response = await fetch('/api/v1/service-monitors', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
    });
    if (!response.ok) {
      throw new Error('Failed to create service monitor');
    }
    const result = await response.json();
    return result.data;
  },

  // Update service monitor
  async update(id: string, data: Partial<ServiceMonitorFormData>): Promise<ServiceMonitor> {
    const response = await fetch(`/api/v1/service-monitors/${id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
    });
    if (!response.ok) {
      throw new Error('Failed to update service monitor');
    }
    const result = await response.json();
    return result.data;
  },

  // Delete service monitor
  async delete(id: string): Promise<void> {
    const response = await fetch(`/api/v1/service-monitors/${id}`, {
      method: 'DELETE',
    });
    if (!response.ok) {
      throw new Error('Failed to delete service monitor');
    }
  },

  // Trigger manual probe
  async triggerProbe(id: string): Promise<ServiceProbeResult> {
    const response = await fetch(`/api/v1/service-monitors/${id}/trigger`, {
      method: 'POST',
    });
    if (!response.ok) {
      throw new Error('Failed to trigger probe');
    }
    const data = await response.json();
    return data.data;
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

    const response = await fetch(`/api/v1/service-monitors/${id}/history?${queryParams.toString()}`);
    if (!response.ok) {
      throw new Error('Failed to fetch probe history');
    }
    const data = await response.json();
    return data.data || [];
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
    average_latency: number;
    p95_latency: number;
    p99_latency: number;
  }> {
    const queryParams = new URLSearchParams();
    if (params?.period) queryParams.append('period', params.period);

    const response = await fetch(`/api/v1/service-monitors/${id}/stats?${queryParams.toString()}`);
    if (!response.ok) {
      throw new Error('Failed to fetch availability stats');
    }
    const data = await response.json();
    return data.data;
  },

  // Batch enable/disable monitors
  async batchUpdate(
    ids: string[],
    action: 'enable' | 'disable'
  ): Promise<void> {
    const response = await fetch('/api/v1/service-monitors/batch', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        ids,
        action,
        enabled: action === 'enable',
      }),
    });
    if (!response.ok) {
      throw new Error('Failed to batch update monitors');
    }
  },

  // Export monitors configuration
  async export(format: 'json' | 'yaml' = 'json'): Promise<Blob> {
    const response = await fetch(`/api/v1/service-monitors/export?format=${format}`);
    if (!response.ok) {
      throw new Error('Failed to export monitors');
    }
    return response.blob();
  },

  // Import monitors configuration
  async import(file: File): Promise<{ imported: number; failed: number }> {
    const formData = new FormData();
    formData.append('file', file);
    const response = await fetch('/api/v1/service-monitors/import', {
      method: 'POST',
      body: formData,
    });
    if (!response.ok) {
      throw new Error('Failed to import monitors');
    }
    const data = await response.json();
    return data.data;
  },

  // Get overview with 30-day statistics for all service monitors
  async getOverview(): Promise<ServiceOverviewResponse> {
    const response = await fetch('/api/v1/vms/service-monitors/overview');
    if (!response.ok) {
      throw new Error('Failed to fetch service overview');
    }
    const data = await response.json();
    return data.data;
  },

  // Get probe history for a specific host (multi-line chart data)
  async getHostProbeHistory(
    hostId: string,
    hours: number = 24
  ): Promise<ServiceHistoryInfo[]> {
    const response = await fetch(`/api/v1/vms/hosts/${hostId}/probe-history?hours=${hours}`);
    if (!response.ok) {
      throw new Error('Failed to fetch host probe history');
    }
    const data = await response.json();
    return data.data || [];
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
  host_node_id: string;
  host_node_name?: string;      // Executor name
  timestamps: number[];         // Unix timestamps in milliseconds
  avg_delays: number[];         // Average delays in milliseconds
  uptimes: number[];            // Uptime percentages
}