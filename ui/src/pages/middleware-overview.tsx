import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useInstances } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import {
  IconDatabase,
  IconServer2,
  IconAlertCircle,
  IconPlus
} from '@tabler/icons-react'

interface MiddlewareOverviewProps {
  type: 'mysql' | 'postgresql' | 'redis'
}

const getIcon = (type: string) => {
  switch (type.toLowerCase()) {
    case 'mysql':
      return IconDatabase
    case 'postgresql':
      return IconDatabase
    case 'redis':
      return IconServer2
    default:
      return IconDatabase
  }
}

const getTypeTitle = (type: string) => {
  switch (type.toLowerCase()) {
    case 'mysql':
      return 'MySQL'
    case 'postgresql':
      return 'PostgreSQL'
    case 'redis':
      return 'Redis'
    default:
      return type.toUpperCase()
  }
}

export function MiddlewareOverview({ type }: MiddlewareOverviewProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const { data, isLoading, error } = useInstances()

  // Filter instances by type
  const instances = data?.data?.filter(instance =>
    instance.type.toLowerCase() === type.toLowerCase()
  ) || []

  const Icon = getIcon(type)
  const typeTitle = getTypeTitle(type)

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold">{typeTitle} {t('middleware.instances', '实例')}</h1>
        </div>
        <div className="grid gap-4 grid-cols-1 md:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-32" />
          ))}
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold">{typeTitle} {t('middleware.instances', '实例')}</h1>
        </div>
        <Alert variant="destructive">
          <IconAlertCircle className="h-4 w-4" />
          <AlertTitle>{t('common.error')}</AlertTitle>
          <AlertDescription>{error?.message || 'Failed to load instances'}</AlertDescription>
        </Alert>
      </div>
    )
  }

  const handleInstanceClick = (instance: any) => {
    // Navigate to specific instance management page
    navigate(`/database/${instance.id}`)
  }

  const handleCreateInstance = () => {
    navigate(`/devops/instances/new?type=${type}`)
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{typeTitle} {t('middleware.instances', '实例')}</h1>
        <Button onClick={handleCreateInstance}>
          <IconPlus className="h-4 w-4 mr-2" />
          {t('middleware.createInstance', '创建实例')}
        </Button>
      </div>

      {/* Instance List */}
      {instances.length === 0 ? (
        <Card>
          <CardContent className="text-center py-12">
            <Icon className="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
            <p className="text-muted-foreground mb-2">
              {t('middleware.noInstancesOfType', `暂无 ${typeTitle} 实例`)}
            </p>
            <Button
              className="mt-4"
              onClick={handleCreateInstance}
            >
              <IconPlus className="h-4 w-4 mr-2" />
              {t('middleware.createFirstInstance', '创建第一个实例')}
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 grid-cols-1 md:grid-cols-2 lg:grid-cols-3">
          {instances.map(instance => {
            return (
              <Card
                key={instance.id}
                className="cursor-pointer hover:shadow-md transition-shadow"
                onClick={() => handleInstanceClick(instance)}
              >
                <CardHeader>
                  <CardTitle className="flex items-center justify-between text-base">
                    <span className="flex items-center gap-2">
                      <Icon className="h-5 w-5" />
                      {instance.name}
                    </span>
                    <Badge
                      variant={instance.status === 'running' ? 'default' : 'secondary'}
                    >
                      {instance.status}
                    </Badge>
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-1 text-sm">
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">{t('common.connection')}:</span>
                      <span>
                        {(instance.connection?.host as string) || 'N/A'}
                        {instance.connection?.port ? `:${instance.connection.port}` : ''}
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">{t('common.version')}:</span>
                      <span>{instance.version || 'N/A'}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">{t('common.health')}:</span>
                      <span>{instance.health || 'unknown'}</span>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
      )}
    </div>
  )
}