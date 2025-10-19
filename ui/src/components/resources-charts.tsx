import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { OverviewData } from '@/types/api'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

export interface ResourceChartsProps {
  data?: OverviewData['resource']
  isLoading: boolean
  error: Error | null
  isError: boolean
}

export function ResourceCharts(props: ResourceChartsProps) {
  const { t } = useTranslation()
  const { isLoading, error, isError } = props
  const chartData = useMemo(() => {
    const { cpu, memory } = props.data || {
      cpu: { requested: 0, allocatable: 0, limited: 0 },
      memory: { requested: 0, allocatable: 0, limited: 0 },
    }
    return [
      {
        name: t('monitoring.cpuUsage'),
        request: cpu.requested / 1000,
        limit: cpu.limited / 1000,
        total: cpu.allocatable / 1000,
        requestPercentage: (cpu.requested / cpu.allocatable) * 100,
        limitPercentage: (cpu.limited / cpu.allocatable) * 100,
        unit: 'cores',
      },
      {
        name: t('monitoring.memoryUsage'),
        request: memory.requested / 1024 / 1024 / 1024 / 1024,
        limit: memory.limited / 1024 / 1024 / 1024 / 1024,
        total: memory.allocatable / 1024 / 1024 / 1024 / 1024,
        requestPercentage: (memory.requested / memory.allocatable) * 100,
        limitPercentage: (memory.limited / memory.allocatable) * 100,
        unit: 'GiB',
      },
    ]
  }, [props, t])

  if (isLoading) {
    return (
      <Card className="animate-pulse">
        <CardHeader>
          <div className="h-6 bg-muted rounded w-1/3 mb-2"></div>
          <div className="h-4 bg-muted rounded w-1/2"></div>
        </CardHeader>
        <CardContent className="space-y-6">
          {Array.from({ length: 2 }).map((_, i) => (
            <div key={i} className="space-y-3">
              <div className="h-4 bg-muted rounded w-1/4"></div>
              <div className="grid grid-cols-2 gap-6">
                <div className="space-y-2">
                  <div className="h-3 bg-muted rounded"></div>
                  <div className="h-2 bg-muted rounded-full"></div>
                </div>
                <div className="space-y-2">
                  <div className="h-3 bg-muted rounded"></div>
                  <div className="h-2 bg-muted rounded-full"></div>
                </div>
              </div>
            </div>
          ))}
        </CardContent>
      </Card>
    )
  }

  if (isError) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span>Resources Request</span>
          </CardTitle>
        </CardHeader>
        <div className="flex flex-col items-center justify-center h-64 gap-2">
          <p className="text-sm text-muted-foreground">
            {error instanceof Error ? error.message : 'Unknown error occurred'}
          </p>
        </div>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('overview.resourceCharts', 'Resource Charts')}</CardTitle>
        <CardDescription>
          {t(
            'overview.resourceChartsDesc',
            'Cluster resource requests and limits overview'
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        {chartData.map((resource) => {
          const requestIsHigh = resource.requestPercentage > 90
          const requestIsMedium = resource.requestPercentage > 60
          const limitIsHigh = resource.limitPercentage > 90
          const limitIsMedium = resource.limitPercentage > 60

          return (
            <div key={resource.name} className="space-y-3">
              <div>
                <div className="flex items-center justify-between mb-2">
                  <span className="font-medium">{resource.name}</span>
                  <span className="text-sm text-muted-foreground">
                    Requests: {resource.request.toFixed(1)} / Limits:{' '}
                    {resource.limit.toFixed(1)} / Total:{' '}
                    {resource.total.toFixed(2)} {resource.unit}
                  </span>
                </div>

                <div className="grid grid-cols-2 gap-6">
                  <div>
                    <div className="flex justify-between text-xs text-muted-foreground mb-1">
                      <span className="font-medium text-blue-600">
                        Requests
                      </span>
                      <span>
                        {resource.request.toFixed(1)} {resource.unit}
                      </span>
                    </div>
                    <div className="w-full bg-muted rounded-full h-2">
                      <div
                        className={`h-2 rounded-full transition-all duration-300 ${
                          requestIsHigh
                            ? 'bg-red-500'
                            : requestIsMedium
                              ? 'bg-yellow-500'
                              : 'bg-blue-500'
                        }`}
                        style={{
                          width: `${Math.min(resource.requestPercentage, 100)}%`,
                        }}
                      />
                    </div>
                    <div className="text-xs text-muted-foreground mt-1">
                      {resource.requestPercentage.toFixed(1)}% of capacity
                    </div>
                  </div>

                  <div>
                    <div className="flex justify-between text-xs text-muted-foreground mb-1">
                      <span className="font-medium text-orange-600">Limits</span>
                      <span>
                        {resource.limit.toFixed(1)} {resource.unit}
                      </span>
                    </div>
                    <div className="w-full bg-muted rounded-full h-2">
                      <div
                        className={`h-2 rounded-full transition-all duration-300 ${
                          limitIsHigh
                            ? 'bg-red-500'
                            : limitIsMedium
                              ? 'bg-yellow-500'
                              : 'bg-orange-500'
                        }`}
                        style={{
                          width: `${Math.min(resource.limitPercentage, 100)}%`,
                        }}
                      />
                    </div>
                    <div className="text-xs text-muted-foreground mt-1">
                      {resource.limitPercentage.toFixed(1)}% of capacity
                    </div>
                  </div>
                </div>

                <div className="text-xs text-muted-foreground mt-2">
                  Available: {(resource.total - resource.request).toFixed(1)}{' '}
                  {resource.unit}
                </div>
              </div>
            </div>
          )
        })}
      </CardContent>
    </Card>
  )
}
