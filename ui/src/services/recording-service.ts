import { apiClient } from '@/lib/api-client'

// TypeScript interfaces for terminal recording management
export interface RecordingMetadata {
  id: string
  recording_type: 'docker' | 'webssh' | 'k8s_node' | 'k8s_pod'
  type_metadata: Record<string, any>
  storage_type: 'local' | 'minio'
  storage_path: string
  file_size: number
  format: string
  started_at: string
  ended_at?: string
  duration: number
  rows: number
  cols: number
  shell: string
  client_ip: string
  description?: string
  tags?: string
  username: string
  user_id: string
  session_id: string
  created_at: string
  updated_at: string
}

export interface RecordingFilters {
  page?: number
  limit?: number
  user_id?: string
  recording_type?: 'docker' | 'webssh' | 'k8s_node' | 'k8s_pod'
  storage_type?: 'local' | 'minio'
  start_time?: string
  end_time?: string
  sort_by?: 'started_at' | 'ended_at' | 'file_size' | 'duration'
  sort_order?: 'asc' | 'desc'
}

export interface ListRecordingsResponse {
  items: RecordingMetadata[]
  total: number
  page: number
  limit: number
  total_pages: number
}

export interface RecordingStatistics {
  total_count: number
  total_size_bytes: number
  by_type: Array<{
    recording_type: string
    count: number
    total_size_bytes: number
  }>
  top_users: Array<{
    username: string
    count: number
  }>
}

export interface CleanupStats {
  invalid_deleted: number
  expired_deleted: number
  orphan_deleted: number
  total_deleted: number
}

export interface CleanupStatusResponse {
  is_running: boolean
  last_run_at?: string
  last_run_duration?: string
  last_run_stats?: CleanupStats
  current_task_id?: string
}

export interface TriggerCleanupResponse {
  task_id: string
  message: string
}

class RecordingService {
  /**
   * List terminal recordings with optional filtering, pagination, and sorting
   */
  async listRecordings(filters?: RecordingFilters): Promise<ListRecordingsResponse> {
    const response = await apiClient.get('/recordings', filters) as any
    const data = response.data || response
    return {
      items: data.items || [],
      total: data.total || 0,
      page: data.page || 1,
      limit: data.limit || 20,
      total_pages: data.total_pages || 0,
    }
  }

  /**
   * Get detailed information about a specific recording
   */
  async getRecording(id: string): Promise<RecordingMetadata> {
    const response = await apiClient.get(`/recordings/${id}`) as any
    return response.data || response
  }

  /**
   * Delete a recording and its associated file
   */
  async deleteRecording(id: string): Promise<{ message: string }> {
    const response = await apiClient.delete(`/recordings/${id}`) as any
    return response.data || response
  }

  /**
   * Search recordings by username, description, or tags
   */
  async searchRecordings(query: string, page = 1, limit = 20): Promise<ListRecordingsResponse> {
    const response = await apiClient.get('/recordings/search', {
      q: query,
      page,
      limit,
    }) as any
    const data = response.data || response
    return {
      items: data.items || [],
      total: data.total || 0,
      page: data.page || 1,
      limit: data.limit || 20,
      total_pages: data.total_pages || 0,
    }
  }

  /**
   * Get aggregated statistics about recordings
   */
  async getStatistics(): Promise<RecordingStatistics> {
    const response = await apiClient.get('/recordings/statistics') as any
    return response.data || response
  }

  /**
   * Get playback content in Asciinema v2 format
   */
  async getPlaybackContent(id: string): Promise<string> {
    const response = await fetch(`/api/v1/recordings/${id}/playback`, {
      credentials: 'include', // Use cookie authentication
    })
    if (!response.ok) {
      throw new Error(`Failed to fetch playback content: ${response.statusText}`)
    }
    return await response.text()
  }

  /**
   * Download recording file as .cast file
   */
  async downloadRecording(id: string, filename?: string): Promise<void> {
    const response = await fetch(`/api/v1/recordings/${id}/download`, {
      credentials: 'include', // Use cookie authentication
    })
    if (!response.ok) {
      throw new Error(`Failed to download recording: ${response.statusText}`)
    }

    // Extract filename from Content-Disposition header if not provided
    const contentDisposition = response.headers.get('Content-Disposition')
    const defaultFilename = filename ||
      (contentDisposition ?
        contentDisposition.split('filename=')[1]?.replace(/"/g, '') :
        `recording-${id}.cast`)

    // Create blob and trigger download
    const blob = await response.blob()
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = defaultFilename
    document.body.appendChild(a)
    a.click()
    window.URL.revokeObjectURL(url)
    document.body.removeChild(a)
  }

  /**
   * Manually trigger the cleanup task (admin only)
   */
  async triggerCleanup(): Promise<TriggerCleanupResponse> {
    const response = await apiClient.post('/recordings/cleanup/trigger') as any
    return response.data || response
  }

  /**
   * Get the current status of the cleanup task
   */
  async getCleanupStatus(): Promise<CleanupStatusResponse> {
    const response = await apiClient.get('/recordings/cleanup/status') as any
    return response.data || response
  }
}

export const recordingService = new RecordingService()
