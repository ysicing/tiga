import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Database, Server, Box, Container } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useTranslation } from 'react-i18next'

export interface SubsystemCardProps {
  type: string
  count: number
  running: number
  stopped: number
  error: number
  onClick: () => void
  hasPermission: boolean
}

const getServiceIcon = (type: string) => {
  const iconClass = 'h-8 w-8 text-muted-foreground'
  const lowerType = type.toLowerCase()

  if (['mysql', 'postgresql', 'postgres', 'redis'].includes(lowerType)) {
    return <Database className={iconClass} />
  }
  if (['docker', 'container'].includes(lowerType)) {
    return <Container className={iconClass} />
  }
  if (['minio', 'k8s', 'kubernetes', 'caddy'].includes(lowerType)) {
    return <Server className={iconClass} />
  }
  return <Box className={iconClass} />
}

export function SubsystemCard({
  type,
  count,
  running,
  stopped,
  error,
  onClick,
  hasPermission,
}: SubsystemCardProps) {
  const { t } = useTranslation()

  // Don't render if user has no permission
  if (!hasPermission) {
    return null
  }

  const isEmpty = count === 0
  const displayName = t(`overview.subsystem.${type.toLowerCase()}`, type)

  return (
    <Card
      role="button"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onClick()
        }
      }}
      className={cn(
        'cursor-pointer transition-all hover:shadow-md hover:border-primary/50',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2'
      )}
    >
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">{displayName}</CardTitle>
        {getServiceIcon(type)}
      </CardHeader>
      <CardContent>
        {isEmpty ? (
          <div className="py-4 text-center">
            <p className="text-sm text-muted-foreground">{t('overview.empty')}</p>
          </div>
        ) : (
          <div className="space-y-2">
            <div className="text-2xl font-bold">
              {count} {t('overview.instances')}
            </div>
            <div className="flex gap-2 flex-wrap">
              {running > 0 && (
                <Badge variant="default" className="bg-green-500/10 text-green-700 dark:text-green-400 hover:bg-green-500/20">
                  {running} {t('overview.status.running')}
                </Badge>
              )}
              {stopped > 0 && (
                <Badge variant="secondary">
                  {stopped} {t('overview.status.stopped')}
                </Badge>
              )}
              {error > 0 && (
                <Badge variant="destructive">
                  {error} {t('overview.status.error')}
                </Badge>
              )}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
