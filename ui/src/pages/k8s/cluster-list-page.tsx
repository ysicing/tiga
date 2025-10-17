import { useState } from 'react'
import {
  Cluster,
  useClusters,
  useDeleteCluster,
  useSetDefaultCluster,
  useTestClusterConnection,
} from '@/services/k8s-api'
import {
  IconCheck,
  IconCircleCheck,
  IconCircleX,
  IconCloud,
  IconEdit,
  IconHistory,
  IconPlugConnected,
  IconPlus,
  IconQuestionMark,
  IconServer,
  IconTrash,
} from '@tabler/icons-react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'

const getHealthStatusBadge = (status: string) => {
  switch (status) {
    case 'healthy':
      return (
        <Badge className="bg-green-500">
          <IconCircleCheck className="w-3 h-3 mr-1" /> 健康
        </Badge>
      )
    case 'unhealthy':
      return (
        <Badge variant="destructive">
          <IconCircleX className="w-3 h-3 mr-1" /> 异常
        </Badge>
      )
    case 'unknown':
    default:
      return (
        <Badge variant="outline">
          <IconQuestionMark className="w-3 h-3 mr-1" /> 未知
        </Badge>
      )
  }
}

export function ClusterListPage() {
  const navigate = useNavigate()
  const [deleteId, setDeleteId] = useState<string | null>(null)

  const { data, isLoading, error } = useClusters()
  const deleteMutation = useDeleteCluster()
  const testMutation = useTestClusterConnection()
  const setDefaultMutation = useSetDefaultCluster()

  const clusters = Array.isArray(data?.data?.clusters) ? data.data.clusters : []

  // Calculate statistics
  const stats = {
    total: clusters.length,
    healthy: clusters.filter((c) => c.health_status === 'healthy').length,
    totalNodes: clusters.reduce((sum, c) => sum + c.node_count, 0),
    totalPods: clusters.reduce((sum, c) => sum + c.pod_count, 0),
  }

  const handleDelete = async () => {
    if (!deleteId) return

    try {
      await deleteMutation.mutateAsync(deleteId)
      toast.success('集群已删除')
      setDeleteId(null)
    } catch (error: any) {
      toast.error(error?.message || '无法删除集群')
    }
  }

  const handleTest = async (cluster: Cluster) => {
    try {
      const result = await testMutation.mutateAsync(cluster.id)
      const msg = result.data.message || '连接成功'
      const version = result.data.version
        ? ` (版本: ${result.data.version})`
        : ''
      toast.success(`${cluster.name}: ${msg}${version}`)
    } catch (error: any) {
      toast.error(error?.response?.data?.error || '无法连接到集群')
    }
  }

  const handleSetDefault = async (cluster: Cluster) => {
    if (cluster.is_default) {
      toast.info('该集群已是默认集群')
      return
    }

    try {
      await setDefaultMutation.mutateAsync(cluster.id)
      toast.success(`已将 ${cluster.name} 设置为默认集群`)
    } catch (error: any) {
      toast.error(error?.message || '无法设置默认集群')
    }
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-3xl font-bold">Kubernetes 集群</h1>
        </div>

        {/* Stats skeleton */}
        <div className="grid gap-4 md:grid-cols-4">
          {[1, 2, 3, 4].map((i) => (
            <Skeleton key={i} className="h-24" />
          ))}
        </div>

        {/* Cards skeleton */}
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-56" />
          ))}
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-3xl font-bold">Kubernetes 集群</h1>
        </div>
        <Card>
          <CardHeader>
            <CardTitle className="text-destructive">加载失败</CardTitle>
            <CardDescription>
              {(error as any)?.message || '无法加载集群列表'}
            </CardDescription>
          </CardHeader>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Kubernetes 集群</h1>
          <p className="text-muted-foreground mt-2">
            管理和监控 Kubernetes 集群
          </p>
        </div>
        <Button onClick={() => navigate('/k8s/clusters/new')}>
          <IconPlus className="w-4 h-4 mr-2" />
          新建集群
        </Button>
      </div>

      {/* Statistics Cards */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">总集群数</CardTitle>
            <IconCloud className="w-4 h-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.total}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">健康集群</CardTitle>
            <IconCircleCheck className="w-4 h-4 text-green-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.healthy}</div>
            <p className="text-xs text-muted-foreground mt-1">
              {stats.total > 0
                ? `${((stats.healthy / stats.total) * 100).toFixed(1)}%`
                : '0%'}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">节点总数</CardTitle>
            <IconServer className="w-4 h-4 text-blue-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.totalNodes}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Pod 总数</CardTitle>
            <IconServer className="w-4 h-4 text-purple-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.totalPods}</div>
          </CardContent>
        </Card>
      </div>

      {/* Cluster Cards */}
      {clusters.length === 0 ? (
        <Card>
          <CardHeader>
            <CardTitle>还没有集群</CardTitle>
            <CardDescription>
              点击"新建集群"按钮开始添加 Kubernetes 集群
            </CardDescription>
          </CardHeader>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {clusters.map((cluster) => (
            <Card
              key={cluster.id}
              className="hover:shadow-lg transition-shadow"
            >
              <CardHeader>
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-2">
                    <IconCloud className="w-5 h-5" />
                    <CardTitle className="text-xl">{cluster.name}</CardTitle>
                  </div>
                  <div className="flex flex-col gap-1 items-end">
                    {getHealthStatusBadge(cluster.health_status)}
                    {cluster.is_default && (
                      <Badge className="bg-blue-500">
                        <IconCheck className="w-3 h-3 mr-1" /> 默认
                      </Badge>
                    )}
                  </div>
                </div>
                {cluster.description && (
                  <CardDescription className="mt-2">
                    {cluster.description}
                  </CardDescription>
                )}
              </CardHeader>
              <CardContent>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">类型:</span>
                    <span className="font-medium">
                      {cluster.in_cluster ? '集群内' : 'Kubeconfig'}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">节点数:</span>
                    <span>{cluster.node_count}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Pod 数:</span>
                    <span>{cluster.pod_count}</span>
                  </div>
                  {cluster.prometheus_url && (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Prometheus:</span>
                      <span className="text-green-500">
                        <IconCheck className="w-4 h-4" />
                      </span>
                    </div>
                  )}
                  {cluster.last_connected_at && (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">最后连接:</span>
                      <span className="text-xs">
                        {new Date(cluster.last_connected_at).toLocaleString(
                          'zh-CN'
                        )}
                      </span>
                    </div>
                  )}
                </div>

                <div className="grid grid-cols-3 gap-2 mt-4">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => navigate(`/k8s/clusters/${cluster.id}`)}
                  >
                    <IconEdit className="w-4 h-4 mr-1" />
                    管理
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleTest(cluster)}
                    disabled={testMutation.isPending}
                  >
                    <IconPlugConnected className="w-4 h-4 mr-1" />
                    测试
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() =>
                      navigate(`/k8s/clusters/${cluster.id}/resource-history`)
                    }
                  >
                    <IconHistory className="w-4 h-4 mr-1" />
                    历史
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleSetDefault(cluster)}
                    disabled={
                      cluster.is_default || setDefaultMutation.isPending
                    }
                  >
                    <IconCheck className="w-4 h-4 mr-1" />
                    设为默认
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setDeleteId(cluster.id)}
                    className="col-span-2"
                  >
                    <IconTrash className="w-4 h-4 mr-1 text-destructive" />
                    删除
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Delete Confirmation Dialog */}
      <AlertDialog
        open={deleteId !== null}
        onOpenChange={() => setDeleteId(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除</AlertDialogTitle>
            <AlertDialogDescription>
              此操作将永久删除该集群配置。集群内的资源不会受影响。
              此操作无法撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
