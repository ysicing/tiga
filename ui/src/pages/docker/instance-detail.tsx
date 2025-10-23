import { useState } from 'react'
import { useDockerInstance, useSystemInfo, useDockerVersion } from '@/services/docker-api'
import { useParams, useNavigate } from 'react-router-dom'
import { IconArrowLeft, IconRefresh } from '@tabler/icons-react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'

// Import tab components (to be created separately)
import { ContainerList } from '@/components/docker/container-list'
import { ImageList } from '@/components/docker/image-list'
import { VolumeList } from '@/components/docker/volume-list'
import { NetworkList } from '@/components/docker/network-list'

export function DockerInstanceDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState('containers')

  const { data: instanceData, isLoading, error, refetch } = useDockerInstance(id!)
  const { data: systemInfoData } = useSystemInfo(id!)
  const { data: versionData } = useDockerVersion(id!)

  const instance = instanceData?.data
  const systemInfo = systemInfoData?.data
  const version = versionData?.data

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-32" />
        <Skeleton className="h-96" />
      </div>
    )
  }

  if (error || !instance) {
    return (
      <div className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle className="text-destructive">加载失败</CardTitle>
            <CardDescription>
              {(error as any)?.message || '无法加载实例详情'}
            </CardDescription>
          </CardHeader>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate('/docker/instances')}>
            <IconArrowLeft className="w-5 h-5" />
          </Button>
          <div>
            <h1 className="text-3xl font-bold">{instance.name}</h1>
            <p className="text-muted-foreground mt-1">
              {instance.host}:{instance.port} • Agent: {instance.agent_name}
            </p>
          </div>
        </div>
        <div className="flex gap-2">
          <Badge variant={instance.status === 'online' ? 'default' : 'secondary'}>
            {instance.status}
          </Badge>
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <IconRefresh className="w-4 h-4 mr-2" />
            刷新
          </Button>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-3">
            <CardDescription>总容器数</CardDescription>
            <CardTitle className="text-3xl">{systemInfo?.containers ?? '-'}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-xs text-muted-foreground">
              运行中: {systemInfo?.containers_running ?? '-'} •
              已停止: {systemInfo?.containers_stopped ?? '-'}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardDescription>镜像数</CardDescription>
            <CardTitle className="text-3xl">{systemInfo?.images ?? '-'}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-xs text-muted-foreground">
              {version?.version ? `Docker ${version.version}` : '版本未知'}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardDescription>系统</CardDescription>
            <CardTitle className="text-lg">{systemInfo?.operating_system ?? '-'}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-xs text-muted-foreground">
              {systemInfo?.architecture} • {systemInfo?.ncpu ?? '-'} CPUs
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardDescription>内存</CardDescription>
            <CardTitle className="text-lg">
              {systemInfo?.mem_total ? `${(systemInfo.mem_total / 1024 / 1024 / 1024).toFixed(1)} GB` : '-'}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-xs text-muted-foreground">
              {systemInfo?.driver ?? 'Unknown'} 驱动
            </div>
          </CardContent>
        </Card>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
        <TabsList>
          <TabsTrigger value="containers">容器</TabsTrigger>
          <TabsTrigger value="images">镜像</TabsTrigger>
          <TabsTrigger value="volumes">卷</TabsTrigger>
          <TabsTrigger value="networks">网络</TabsTrigger>
          <TabsTrigger value="system">系统信息</TabsTrigger>
        </TabsList>

        <TabsContent value="containers" className="space-y-4">
          <ContainerList instanceId={id!} />
        </TabsContent>

        <TabsContent value="images" className="space-y-4">
          <ImageList instanceId={id!} />
        </TabsContent>

        <TabsContent value="volumes" className="space-y-4">
          <VolumeList instanceId={id!} />
        </TabsContent>

        <TabsContent value="networks" className="space-y-4">
          <NetworkList instanceId={id!} />
        </TabsContent>

        <TabsContent value="system" className="space-y-4">
          {systemInfo && (
            <Card>
              <CardHeader>
                <CardTitle>系统信息</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid gap-2 text-sm">
                  <div className="grid grid-cols-2 gap-2">
                    <span className="text-muted-foreground">Docker 版本:</span>
                    <span>{version?.version ?? '-'}</span>
                  </div>
                  <div className="grid grid-cols-2 gap-2">
                    <span className="text-muted-foreground">API 版本:</span>
                    <span>{version?.api_version ?? '-'}</span>
                  </div>
                  <div className="grid grid-cols-2 gap-2">
                    <span className="text-muted-foreground">操作系统:</span>
                    <span>{systemInfo.operating_system}</span>
                  </div>
                  <div className="grid grid-cols-2 gap-2">
                    <span className="text-muted-foreground">内核版本:</span>
                    <span>{systemInfo.kernel_version}</span>
                  </div>
                  <div className="grid grid-cols-2 gap-2">
                    <span className="text-muted-foreground">架构:</span>
                    <span>{systemInfo.architecture}</span>
                  </div>
                  <div className="grid grid-cols-2 gap-2">
                    <span className="text-muted-foreground">CPU 数量:</span>
                    <span>{systemInfo.ncpu}</span>
                  </div>
                  <div className="grid grid-cols-2 gap-2">
                    <span className="text-muted-foreground">存储驱动:</span>
                    <span>{systemInfo.driver}</span>
                  </div>
                  <div className="grid grid-cols-2 gap-2">
                    <span className="text-muted-foreground">日志驱动:</span>
                    <span>{systemInfo.logging_driver}</span>
                  </div>
                  <div className="grid grid-cols-2 gap-2">
                    <span className="text-muted-foreground">Cgroup 驱动:</span>
                    <span>{systemInfo.cgroup_driver}</span>
                  </div>
                  <div className="grid grid-cols-2 gap-2">
                    <span className="text-muted-foreground">Docker 根目录:</span>
                    <span className="font-mono text-xs">{systemInfo.docker_root_dir}</span>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}
        </TabsContent>
      </Tabs>
    </div>
  )
}
