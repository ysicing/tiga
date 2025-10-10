import axios from 'axios';

export type AlertType = 'host' | 'service';
export type AlertSeverity = 'info' | 'warning' | 'critical';
export type AlertStatus = 'firing' | 'acknowledged' | 'resolved';

export interface AlertRule {
  id: string;
  created_at: string;
  updated_at: string;
  name: string;
  type: AlertType;
  target_id: string;
  severity: AlertSeverity;
  condition: string;
  duration: number;
  enabled: boolean;
  notify_channels?: string;
  notify_config?: string;
}

export interface AlertEvent {
  id: string;
  created_at: string;
  updated_at: string;
  rule_id: string;
  rule_name?: string;
  target_id: string;
  target_name?: string;
  severity: AlertSeverity;
  status: AlertStatus;
  message: string;
  details?: string;
  started_at: string;
  acknowledged_at?: string;
  acknowledged_by?: string;
  ack_note?: string;
  resolved_at?: string;
  resolved_by?: string;
  res_note?: string;
}

export interface AlertRuleFilter {
  type?: string;
  enabled?: boolean;
  severity?: string;
  page?: number;
  page_size?: number;
}

export interface AlertEventFilter {
  rule_id?: number;
  status?: string;
  severity?: string;
  page?: number;
  page_size?: number;
}

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
}

const API_BASE = '/api/v1/vms';

export const AlertRuleService = {
  // Alert Rules
  async createRule(rule: Partial<AlertRule>): Promise<AlertRule> {
    const response = await axios.post(`${API_BASE}/alert-rules`, rule);
    return response.data.data;
  },

  async listRules(filter?: AlertRuleFilter): Promise<PaginatedResponse<AlertRule>> {
    const response = await axios.get(`${API_BASE}/alert-rules`, { params: filter });
    return response.data.data;
  },

  async getRule(id: string): Promise<AlertRule> {
    const response = await axios.get(`${API_BASE}/alert-rules/${id}`);
    return response.data.data;
  },

  async updateRule(id: string, rule: Partial<AlertRule>): Promise<AlertRule> {
    const response = await axios.put(`${API_BASE}/alert-rules/${id}`, rule);
    return response.data.data;
  },

  async deleteRule(id: string): Promise<void> {
    await axios.delete(`${API_BASE}/alert-rules/${id}`);
  },

  // Alert Events
  async listEvents(filter?: AlertEventFilter): Promise<PaginatedResponse<AlertEvent>> {
    const response = await axios.get(`${API_BASE}/alert-events`, { params: filter });
    return response.data.data;
  },

  async acknowledgeEvent(id: string, note: string): Promise<void> {
    await axios.post(`${API_BASE}/alert-events/${id}/acknowledge`, { note });
  },

  async resolveEvent(id: string, note: string): Promise<void> {
    await axios.post(`${API_BASE}/alert-events/${id}/resolve`, { note });
  },
};
