import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { ServerIcon, GitCommitIcon, ClockIcon } from 'lucide-react'

import { versionAPI } from '@/services/version'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'

/**
 * Version information component
 * Displays server version, build time, and commit ID
 */
export function VersionInfo() {
  const { t } = useTranslation()

  const {
    data: version,
    isLoading,
    isError,
    error,
  } = useQuery({
    queryKey: ['version'],
    queryFn: versionAPI.getVersion,
    staleTime: 5 * 60 * 1000, // 5 minutes
  })

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>{t('system.version.title', '版本信息')}</CardTitle>
          <CardDescription>
            {t('system.version.loading', '加载中...')}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Skeleton className="h-6 w-full" />
          <Skeleton className="h-6 w-full" />
          <Skeleton className="h-6 w-full" />
        </CardContent>
      </Card>
    )
  }

  if (isError) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>{t('system.version.title', '版本信息')}</CardTitle>
          <CardDescription className="text-destructive">
            {t('system.version.error', '加载失败：')} {error?.message || 'Unknown error'}
          </CardDescription>
        </CardHeader>
      </Card>
    )
  }

  // Format build time to local timezone
  const formatBuildTime = (buildTime: string) => {
    if (buildTime === 'unknown') {
      return t('system.version.unknown', '未知')
    }
    try {
      const date = new Date(buildTime)
      return date.toLocaleString()
    } catch {
      return buildTime
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('system.version.title', '版本信息')}</CardTitle>
        <CardDescription>
          {t('system.version.description', 'Tiga 服务端版本信息')}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex items-center gap-3 p-3 rounded-lg bg-muted/50">
          <ServerIcon className="h-5 w-5 text-muted-foreground" />
          <div className="flex-1">
            <p className="text-sm font-medium text-muted-foreground">
              {t('system.version.version', '版本号')}
            </p>
            <p className="text-base font-mono">{version?.version || 'dev'}</p>
          </div>
        </div>

        <div className="flex items-center gap-3 p-3 rounded-lg bg-muted/50">
          <ClockIcon className="h-5 w-5 text-muted-foreground" />
          <div className="flex-1">
            <p className="text-sm font-medium text-muted-foreground">
              {t('system.version.buildTime', '构建时间')}
            </p>
            <p className="text-base">{formatBuildTime(version?.build_time || 'unknown')}</p>
          </div>
        </div>

        <div className="flex items-center gap-3 p-3 rounded-lg bg-muted/50">
          <GitCommitIcon className="h-5 w-5 text-muted-foreground" />
          <div className="flex-1">
            <p className="text-sm font-medium text-muted-foreground">
              {t('system.version.commitId', 'Commit ID')}
            </p>
            <p className="text-base font-mono">{version?.commit_id || '0000000'}</p>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
