import { apiClient } from '@/lib/api-client'

export interface AppConfig {
  app_name: string
  app_subtitle: string
}

export const configApi = {
  /**
   * Get application configuration
   * Used by login page to customize branding
   */
  getAppConfig: async (): Promise<AppConfig> => {
    return apiClient.get<AppConfig>('/config')
  },
}
