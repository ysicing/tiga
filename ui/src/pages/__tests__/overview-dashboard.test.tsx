import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { Instance } from '@/types/api'
import * as api from '@/lib/api'

import { OverviewDashboard } from '../overview-dashboard'

// Mock the API module
vi.mock('@/lib/api', () => ({
  useInstances: vi.fn(),
}))

// Mock auth context
vi.mock('@/contexts/auth-context', () => ({
  useAuth: () => ({
    user: { roles: [{ name: 'admin' }] },
    hasPermission: () => true,
  }),
}))

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  })
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>{children}</BrowserRouter>
    </QueryClientProvider>
  )
}

describe('OverviewDashboard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders loading state initially', () => {
    vi.mocked(api.useInstances).mockReturnValue({
      data: undefined,
      isLoading: true,
      error: null,
      isError: false,
    } as any)

    render(<OverviewDashboard />, { wrapper: createWrapper() })

    expect(screen.getByText(/loading/i)).toBeInTheDocument()
  })

  it('renders subsystem cards with aggregated data', async () => {
    const mockInstances: Instance[] = [
      {
        id: '1',
        name: 'mysql-1',
        type: 'mysql',
        status: 'running',
        health: 'healthy',
        created_at: '2025-01-01',
        updated_at: '2025-01-01',
      },
      {
        id: '2',
        name: 'mysql-2',
        type: 'mysql',
        status: 'stopped',
        health: 'unknown',
        created_at: '2025-01-01',
        updated_at: '2025-01-01',
      },
      {
        id: '3',
        name: 'redis-1',
        type: 'redis',
        status: 'running',
        health: 'healthy',
        created_at: '2025-01-01',
        updated_at: '2025-01-01',
      },
      {
        id: '4',
        name: 'minio-1',
        type: 'minio',
        status: 'error',
        health: 'unhealthy',
        created_at: '2025-01-01',
        updated_at: '2025-01-01',
      },
    ]

    vi.mocked(api.useInstances).mockReturnValue({
      data: { data: mockInstances },
      isLoading: false,
      error: null,
      isError: false,
    } as any)

    render(<OverviewDashboard />, { wrapper: createWrapper() })

    await waitFor(() => {
      expect(screen.getByText(/MySQL/i)).toBeInTheDocument()
      expect(screen.getByText(/Redis/i)).toBeInTheDocument()
      expect(screen.getByText(/MinIO/i)).toBeInTheDocument()
    })

    // Check aggregated counts
    expect(screen.getByText(/2.*instance/i)).toBeInTheDocument() // MySQL has 2 instances
  })

  it('correctly aggregates instances by type', async () => {
    const mockInstances: Instance[] = [
      {
        id: '1',
        name: 'mysql-1',
        type: 'mysql',
        status: 'running',
        health: 'healthy',
        created_at: '2025-01-01',
        updated_at: '2025-01-01',
      },
      {
        id: '2',
        name: 'mysql-2',
        type: 'mysql',
        status: 'running',
        health: 'healthy',
        created_at: '2025-01-01',
        updated_at: '2025-01-01',
      },
      {
        id: '3',
        name: 'mysql-3',
        type: 'mysql',
        status: 'stopped',
        health: 'unknown',
        created_at: '2025-01-01',
        updated_at: '2025-01-01',
      },
      {
        id: '4',
        name: 'mysql-4',
        type: 'mysql',
        status: 'error',
        health: 'unhealthy',
        created_at: '2025-01-01',
        updated_at: '2025-01-01',
      },
    ]

    vi.mocked(api.useInstances).mockReturnValue({
      data: { data: mockInstances },
      isLoading: false,
      error: null,
      isError: false,
    } as any)

    render(<OverviewDashboard />, { wrapper: createWrapper() })

    await waitFor(() => {
      // Should show 4 instances total
      expect(screen.getByText(/4.*instance/i)).toBeInTheDocument()
      // Should show 2 running
      expect(screen.getByText(/2.*running/i)).toBeInTheDocument()
      // Should show 1 stopped
      expect(screen.getByText(/1.*stopped/i)).toBeInTheDocument()
      // Should show 1 error
      expect(screen.getByText(/1.*error/i)).toBeInTheDocument()
    })
  })

  it('displays error message when data fetch fails', async () => {
    vi.mocked(api.useInstances).mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error('Network error'),
      isError: true,
    } as any)

    render(<OverviewDashboard />, { wrapper: createWrapper() })

    await waitFor(() => {
      expect(screen.getByText(/failed to load/i)).toBeInTheDocument()
    })
  })

  it('renders responsive grid layout', async () => {
    const mockInstances: Instance[] = [
      {
        id: '1',
        name: 'mysql-1',
        type: 'mysql',
        status: 'running',
        health: 'healthy',
        created_at: '2025-01-01',
        updated_at: '2025-01-01',
      },
      {
        id: '2',
        name: 'redis-1',
        type: 'redis',
        status: 'running',
        health: 'healthy',
        created_at: '2025-01-01',
        updated_at: '2025-01-01',
      },
    ]

    vi.mocked(api.useInstances).mockReturnValue({
      data: { data: mockInstances },
      isLoading: false,
      error: null,
      isError: false,
    } as any)

    const { container } = render(<OverviewDashboard />, {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      const grid = container.querySelector('.grid')
      expect(grid).toBeInTheDocument()
      expect(grid).toHaveClass('grid')
    })
  })

  it('filters cards based on user permissions', async () => {
    const mockInstances: Instance[] = [
      {
        id: '1',
        name: 'mysql-1',
        type: 'mysql',
        status: 'running',
        health: 'healthy',
        created_at: '2025-01-01',
        updated_at: '2025-01-01',
      },
      {
        id: '2',
        name: 'redis-1',
        type: 'redis',
        status: 'running',
        health: 'healthy',
        created_at: '2025-01-01',
        updated_at: '2025-01-01',
      },
    ]

    vi.mocked(api.useInstances).mockReturnValue({
      data: { data: mockInstances },
      isLoading: false,
      error: null,
      isError: false,
    } as any)

    render(<OverviewDashboard />, { wrapper: createWrapper() })

    await waitFor(() => {
      // Should render cards for permitted subsystems
      expect(screen.getByText(/MySQL/i)).toBeInTheDocument()
      expect(screen.getByText(/Redis/i)).toBeInTheDocument()
    })
  })

  it('handles empty instance list', async () => {
    vi.mocked(api.useInstances).mockReturnValue({
      data: { data: [] },
      isLoading: false,
      error: null,
      isError: false,
    } as any)

    render(<OverviewDashboard />, { wrapper: createWrapper() })

    await waitFor(() => {
      expect(screen.getByText(/no instances/i)).toBeInTheDocument()
    })
  })
})
