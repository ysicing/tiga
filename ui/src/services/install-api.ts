import type {
  DatabaseConfig,
  AdminAccount,
  SystemSettings,
  CheckDBResponse,
  ValidateResponse,
  FinalizeResponse,
  StatusResponse,
} from '@/lib/schemas/install-schemas'

// T036: Install API Service

const API_BASE = '/api/install'

export const installApi = {
  /**
   * GET /api/install/status
   * Check installation status
   */
  async checkStatus(): Promise<StatusResponse> {
    const response = await fetch(`${API_BASE}/status`)
    if (!response.ok) {
      throw new Error('Failed to check installation status')
    }
    return response.json()
  },

  /**
   * POST /api/install/check-db
   * Test database connection and check for existing data
   */
  async checkDatabase(config: DatabaseConfig & { confirm_reinstall?: boolean }): Promise<CheckDBResponse> {
    const response = await fetch(`${API_BASE}/check-db`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(config),
    })

    if (!response.ok) {
      const error = await response.json()
      throw new Error(error.error || 'Failed to check database')
    }

    return response.json()
  },

  /**
   * POST /api/install/validate-admin
   * Validate admin account information
   */
  async validateAdmin(admin: Omit<AdminAccount, 'confirm_password'>): Promise<ValidateResponse> {
    const response = await fetch(`${API_BASE}/validate-admin`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(admin),
    })

    if (!response.ok) {
      const error = await response.json()
      throw new Error(error.error || 'Failed to validate admin account')
    }

    return response.json()
  },

  /**
   * POST /api/install/validate-settings
   * Validate system settings
   */
  async validateSettings(settings: SystemSettings): Promise<ValidateResponse> {
    const response = await fetch(`${API_BASE}/validate-settings`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(settings),
    })

    if (!response.ok) {
      const error = await response.json()
      throw new Error(error.error || 'Failed to validate settings')
    }

    return response.json()
  },

  /**
   * POST /api/install/finalize
   * Complete installation
   */
  async finalize(data: {
    database: DatabaseConfig
    admin: Omit<AdminAccount, 'confirm_password'>
    settings: SystemSettings
    confirm_reinstall?: boolean
  }): Promise<FinalizeResponse> {
    const response = await fetch(`${API_BASE}/finalize`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        database: data.database,
        admin: data.admin,
        settings: data.settings,
        confirm_reinstall: data.confirm_reinstall || false,
      }),
    })

    const result = await response.json()

    if (response.status === 403) {
      throw new Error(result.error || 'Installation already completed')
    }

    if (response.status === 409) {
      throw new Error(result.error || 'Existing data found')
    }

    if (!response.ok) {
      throw new Error(result.error || 'Installation failed')
    }

    return result
  },
}
