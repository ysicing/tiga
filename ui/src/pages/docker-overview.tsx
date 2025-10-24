import {
  IconServer,
  IconPlus,
  IconRefresh,
} from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { useDockerInstances } from '@/services/docker-api'
import { Skeleton } from '@/components/ui/skeleton'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { IconAlertCircle } from '@tabler/icons-react'

export function DockerOverview() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { data, isLoading, error, refetch } = useDockerInstances()

  const instances = data?.data || []

  if (error) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold">
            {t('docker.overview', 'Docker 概览')}
          </h1>
        </div>
        <Alert variant="destructive">
          <IconAlertCircle className="h-4 w-4" />
          <AlertTitle>{t('common.error', '错误')}</AlertTitle>
          <AlertDescription>
            {(error as any)?.message || t('docker.loadFailed', '加载失败')}
          </AlertDescription>
        </Alert>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">
            {t('docker.overview', 'Docker 概览')}
          </h1>
          <p className="text-muted-foreground mt-1">
            {t('docker.overview.description', '选择一个 Docker 实例开始管理容器、镜像和网络')}
          </p>
        </div>
        <div className="flex gap-2">
          <Button onClick={() => refetch()} variant="outline" size="sm">
            <IconRefresh className="h-4 w-4 mr-2" />
            {t('common.refresh', '刷新')}
          </Button>
          <Button onClick={() => navigate('/docker/instances/new')} size="sm">
            <IconPlus className="h-4 w-4 mr-2" />
            {t('docker.addInstance', '添加实例')}
          </Button>
        </div>
      </div>

      {isLoading ? (
        <div className="grid gap-6 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-40" />
          ))}
        </div>
      ) : instances.length === 0 ? (
        <Card>
          <CardHeader>
            <CardTitle>{t('docker.noInstances', '暂无实例')}</CardTitle>
            <CardDescription>
              {t('docker.noInstancesDescription', '点击右上角"添加实例"按钮开始创建 Docker 实例')}
            </CardDescription>
          </CardHeader>
        </Card>
      ) : (
        <div className="grid gap-6 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3">
          {instances.map((instance) => (
            <Card
              key={instance.id}
              className="cursor-pointer transition-all hover:shadow-lg hover:scale-105"
              onClick={() => navigate(`/docker/instances/${instance.id}/containers`)}
            >
              <CardHeader className="pb-4">
                <div className="flex items-center justify-between">
                  <CardTitle className="flex items-center gap-3">
                    <IconServer className="h-6 w-6" />
                    <span>{instance.name}</span>
                  </CardTitle>
                  <Badge
                    variant={instance.status === 'online' ? 'default' : 'secondary'}
                  >
                    {instance.status === 'online' ? t('common.online', '在线') : t('common.offline', '离线')}
                  </Badge>
                </div>
                <CardDescription className="mt-2">
                  {instance.description || t('docker.noDescription', '暂无描述')}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="text-sm space-y-1">
                  <div className="flex items-center justify-between text-muted-foreground">
                    <span>{t('docker.host', '主机')}:</span>
                    <span className="font-mono">{instance.host}:{instance.port}</span>
                  </div>
                  <div className="flex items-center justify-between text-muted-foreground">
                    <span>{t('docker.agent', 'Agent')}:</span>
                    <span>{instance.agent_name}</span>
                  </div>
                  {instance.version && (
                    <div className="flex items-center justify-between text-muted-foreground">
                      <span>{t('docker.version', '版本')}:</span>
                      <span>{instance.version}</span>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
