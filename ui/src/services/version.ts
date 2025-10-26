import { apiClient } from '@/lib/api-client'
import { VersionInfo } from '@/types/version'

/**
 * Version API client for server and agent version information
 */
export const versionAPI = {
  /**
   * Get server version information
   * @returns Promise with version, build time, and commit ID
   */
  getVersion: async (): Promise<VersionInfo> => {
    return apiClient.get<VersionInfo>('/version')
  },
}
