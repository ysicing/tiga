import { useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useInstances } from '@/lib/api'
import { SubsystemCard } from '@/components/subsystem-card'
import type { Instance, SubsystemStats } from '@/types/api'
import { Skeleton } from '@/components/ui/skeleton'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { AlertCircle } from 'lucide-react'
import { ClusterProvider } from '@/contexts/cluster-context'
import { GlobalSearchProvider, useGlobalSearch } from '@/components/global-search-provider'
import { GlobalSearch } from '@/components/global-search'
import { Toaster } from '@/components/ui/sonner'

// Aggregate instances by type
function aggregateInstancesByType(instances: Instance[]): SubsystemStats[] {
  const grouped = instances.reduce((acc, instance) => {
    const type = instance.type.toLowerCase()
    if (!acc[type]) {
      acc[type] = { type, count: 0, running: 0, stopped: 0, error: 0 }
    }
    acc[type].count++

    if (instance.status === 'running') {
      acc[type].running++
    } else if (instance.status === 'stopped') {
      acc[type].stopped++
    } else if (instance.status === 'error') {
      acc[type].error++
    }

    return acc
  }, {} as Record<string, SubsystemStats>)

  return Object.values(grouped).sort((a, b) => a.type.localeCompare(b.type))
}

// Get all possible subsystem types (for showing empty cards)
const ALL_SUBSYSTEM_TYPES = [
  'mysql',
  'postgresql',
  'redis',
  'minio',
  'docker',
  'kubernetes',
  'caddy',
]

export function OverviewDashboard() {
  return (
    <ClusterProvider>
      <GlobalSearchProvider>
        <OverviewDashboardContent />
      </GlobalSearchProvider>
    </ClusterProvider>
  )
}

function OverviewDashboardContent() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { data, isLoading, error, isError } = useInstances()
  const { isOpen, closeSearch } = useGlobalSearch()

  // Aggregate instances by type
  const subsystemStats = useMemo(() => {
    if (!data?.data) return []

    const aggregated = aggregateInstancesByType(data.data)
    const statsMap = new Map(aggregated.map(stat => [stat.type, stat]))

    // Include all subsystem types, showing 0 for empty ones
    return ALL_SUBSYSTEM_TYPES.map(type =>
      statsMap.get(type) || { type, count: 0, running: 0, stopped: 0, error: 0 }
    )
  }, [data])

  const handleCardClick = (type: string) => {
    const lowerType = type.toLowerCase()

    // Route to different subsystems based on type
    switch (lowerType) {
      case 'kubernetes':
      case 'k8s':
        // Navigate to K8s subsystem
        navigate('/k8s')
        break
      case 'minio':
      case 'docker':
      case 'caddy':
        navigate(`/devops/instances?type=${lowerType}`)
        break
      case 'mysql':
      case 'postgresql':
      case 'postgres':
      case 'redis':
        navigate(`/dbs/${lowerType === 'postgres' ? 'postgresql' : lowerType}`)
        break
      default:
        // Default: navigate to DevOps instances with type filter
        navigate(`/devops/instances?type=${lowerType}`)
    }
  }

  // Loading state
  if (isLoading) {
    return (
      <div className="flex flex-col gap-6 p-8">
        <div>
          <Skeleton className="h-8 w-48" />
        </div>
        <div className="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {[...Array(8)].map((_, i) => (
            <Skeleton key={i} className="h-40" />
          ))}
        </div>
      </div>
    )
  }

  // Error state
  if (isError || error) {
    return (
      <div className="flex flex-col gap-6 p-8">
        <div>
          <h1 className="text-2xl font-bold">{t('overview.title')}</h1>
        </div>
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>{t('overview.failedToLoad')}</AlertTitle>
          <AlertDescription>
            {error instanceof Error ? error.message : t('overview.loadError')}
          </AlertDescription>
        </Alert>
      </div>
    )
  }

  // Empty state
  if (!data?.data || data.data.length === 0) {
    return (
      <div className="flex flex-col gap-6 p-8">
        <div>
          <h1 className="text-2xl font-bold">{t('overview.title')}</h1>
        </div>
        <div className="flex items-center justify-center h-64">
          <div className="text-center">
            <p className="text-lg text-muted-foreground">{t('overview.noInstances')}</p>
            <p className="text-sm text-muted-foreground mt-2">
              {t('overview.createFirst')}
            </p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <>
      <div className="flex flex-col gap-6 min-h-screen bg-gradient-to-br from-background via-background to-muted/20 p-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold bg-gradient-to-r from-primary to-primary/60 bg-clip-text text-transparent">
              {t('overview.title')}
            </h1>
            <p className="text-muted-foreground mt-2 text-lg">
              {t('overview.manageInfrastructure')}
            </p>
          </div>
        </div>

        <div className="grid gap-6 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {subsystemStats.map((stats) => (
            <SubsystemCard
              key={stats.type}
              type={stats.type}
              count={stats.count}
              running={stats.running}
              stopped={stats.stopped}
              error={stats.error}
              onClick={() => handleCardClick(stats.type)}
              hasPermission={true} // TODO: Implement actual permission check
            />
          ))}
        </div>
      </div>
      <GlobalSearch open={isOpen} onOpenChange={closeSearch} />
      <Toaster />
    </>
  )
}
