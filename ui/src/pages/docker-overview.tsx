import {
  IconServer,
  IconPlus,
  IconRefresh,
  IconLayoutGrid,
  IconList,
} from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { useState } from 'react'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { useDockerInstances } from '@/services/docker-api'
import { Skeleton } from '@/components/ui/skeleton'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { IconAlertCircle } from '@tabler/icons-react'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'

type ViewMode = 'grid' | 'list'

export function DockerOverview() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { data, isLoading, error, refetch } = useDockerInstances()
  const [viewMode, setViewMode] = useState<ViewMode>('grid')

  const instances = data?.data || []

  if (error) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold">
            {t('docker.instances', 'Docker 实例')}
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
            {t('docker.instances', 'Docker 实例')}
          </h1>
          <p className="text-muted-foreground mt-1">
            {t('docker.instances.description', '管理 Docker 实例，查看容器、镜像和网络')}
          </p>
        </div>
        <div className="flex gap-2">
          <ToggleGroup type="single" value={viewMode} onValueChange={(value) => value && setViewMode(value as ViewMode)}>
            <ToggleGroupItem value="grid" aria-label="网格视图">
              <IconLayoutGrid className="h-4 w-4" />
            </ToggleGroupItem>
            <ToggleGroupItem value="list" aria-label="列表视图">
              <IconList className="h-4 w-4" />
            </ToggleGroupItem>
          </ToggleGroup>
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
        <div className={viewMode === 'grid' ? 'grid gap-6 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3' : 'space-y-4'}>
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className={viewMode === 'grid' ? 'h-40' : 'h-24'} />
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
      ) : viewMode === 'grid' ? (
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
                  <div className="flex items-center justify-between pt-2 mt-2 border-t">
                    <div className="text-center">
                      <div className="text-lg font-bold text-blue-600">
                        {instance.total_containers ?? 0}
                      </div>
                      <div className="text-xs text-muted-foreground">容器</div>
                    </div>
                    <div className="text-center">
                      <div className="text-lg font-bold text-purple-600">
                        {instance.total_images ?? 0}
                      </div>
                      <div className="text-xs text-muted-foreground">镜像</div>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : (
        <div className="space-y-3">
          {instances.map((instance) => (
            <Card
              key={instance.id}
              className="cursor-pointer transition-all hover:shadow-md"
              onClick={() => navigate(`/docker/instances/${instance.id}/containers`)}
            >
              <CardContent className="py-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-4 flex-1">
                    <IconServer className="h-8 w-8 text-muted-foreground flex-shrink-0" />
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <h3 className="font-semibold text-lg truncate">{instance.name}</h3>
                        <Badge
                          variant={instance.status === 'online' ? 'default' : 'secondary'}
                          className="flex-shrink-0"
                        >
                          {instance.status === 'online' ? '在线' : '离线'}
                        </Badge>
                      </div>
                      <p className="text-sm text-muted-foreground truncate">
                        {instance.description || '暂无描述'}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-6 ml-4">
                    <div className="text-sm text-muted-foreground">
                      <div className="font-mono">{instance.host}:{instance.port}</div>
                      <div className="text-xs">Agent: {instance.agent_name}</div>
                    </div>
                    <div className="flex items-center gap-4">
                      <div className="text-center">
                        <div className="text-xl font-bold text-blue-600">
                          {instance.total_containers ?? 0}
                        </div>
                        <div className="text-xs text-muted-foreground">容器</div>
                      </div>
                      <div className="text-center">
                        <div className="text-xl font-bold text-purple-600">
                          {instance.total_images ?? 0}
                        </div>
                        <div className="text-xs text-muted-foreground">镜像</div>
                      </div>
                    </div>
                    {instance.version && (
                      <div className="text-sm text-muted-foreground">
                        <div className="text-xs">Docker</div>
                        <div className="font-medium">{instance.version}</div>
                      </div>
                    )}
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
