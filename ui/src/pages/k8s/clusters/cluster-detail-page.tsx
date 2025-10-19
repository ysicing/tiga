import { useParams, useNavigate, Link } from 'react-router-dom'
import {
  ArrowLeft,
  Check,
  Edit,
  Server,
  Activity,
  Cloud,
  Database,
  RefreshCw,
  Trash2,
  Shield,
  Clock,
} from 'lucide-react'
import { toast } from 'sonner'
import { formatDistanceToNow } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { useState } from 'react'

import {
  useCluster,
  useDeleteCluster,
  useTestClusterConnection,
  usePrometheusRediscover,
} from '@/services/k8s-api'
import { Cluster } from '@/types/api'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Separator } from '@/components/ui/separator'
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
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/ui/tabs'

const getHealthStatusColor = (status: string) => {
  switch (status) {
    case 'healthy':
      return 'text-green-600 bg-green-100 dark:bg-green-900/30'
    case 'unhealthy':
      return 'text-red-600 bg-red-100 dark:bg-red-900/30'
    default:
      return 'text-gray-600 bg-gray-100 dark:bg-gray-900/30'
  }
}

const getHealthStatusIcon = (status: string) => {
  switch (status) {
    case 'healthy':
      return <Activity className="h-4 w-4" />
    case 'unhealthy':
      return <Shield className="h-4 w-4" />
    default:
      return <Cloud className="h-4 w-4" />
  }
}

export function ClusterDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)

  const { data, isLoading, error } = useCluster(id!)
  const deleteMutation = useDeleteCluster()
  const testMutation = useTestClusterConnection()
  const rediscoverMutation = usePrometheusRediscover()

  const cluster = data?.data as Cluster | undefined

  const handleDelete = async () => {
    if (!id) return

    try {
      await deleteMutation.mutateAsync(id)
      toast.success('集群已删除')
      navigate('/k8s/clusters')
    } catch (error: any) {
      toast.error(error?.message || '无法删除集群')
    }
  }

  const handleTest = async () => {
    if (!id) return

    try {
      const result = await testMutation.mutateAsync(id)
      const msg = result.data.message || '连接成功'
      const version = result.data.version ? ` (版本: ${result.data.version})` : ''
      toast.success(`${msg}${version}`)
    } catch (error: any) {
      toast.error(error?.response?.data?.error || '无法连接到集群')
    }
  }

  const handlePrometheusRediscover = async () => {
    if (!id) return

    try {
      await rediscoverMutation.mutateAsync(id)
      toast.success('Prometheus 重新检测已启动')
    } catch (error: any) {
      toast.error(error?.message || 'Prometheus 重新检测失败')
    }
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate(-1)}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <Skeleton className="h-8 w-48" />
        </div>

        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3, 4, 5, 6].map((i) => (
            <Skeleton key={i} className="h-32" />
          ))}
        </div>
      </div>
    )
  }

  if (error || !cluster) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate(-1)}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <h1 className="text-2xl font-bold">集群未找到</h1>
        </div>

        <Card>
          <CardContent className="pt-6">
            <p className="text-muted-foreground">
              无法找到请求的集群或加载集群信息时出错
            </p>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate(-1)}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-3xl font-bold">{cluster.name}</h1>
              {cluster.is_default && (
                <Badge variant="secondary">默认</Badge>
              )}
              <Badge
                className={getHealthStatusColor(cluster.health_status)}
                variant="outline"
              >
                <span className="flex items-center gap-1.5">
                  {getHealthStatusIcon(cluster.health_status)}
                  {cluster.health_status}
                </span>
              </Badge>
            </div>
            {cluster.description && (
              <p className="text-muted-foreground mt-1">
                {cluster.description}
              </p>
            )}
          </div>
        </div>

        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={handleTest}
            disabled={testMutation.isPending}
          >
            {testMutation.isPending ? (
              <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Activity className="mr-2 h-4 w-4" />
            )}
            测试连接
          </Button>
          <Button variant="outline" onClick={() => navigate(`/k8s/clusters/${id}/edit`)}>
            <Edit className="mr-2 h-4 w-4" />
            编辑
          </Button>
          <Button
            variant="destructive"
            onClick={() => setDeleteDialogOpen(true)}
            disabled={cluster.is_default}
          >
            <Trash2 className="mr-2 h-4 w-4" />
            删除
          </Button>
        </div>
      </div>

      <Tabs defaultValue="overview" className="space-y-4">
        <TabsList>
          <TabsTrigger value="overview">概览</TabsTrigger>
          <TabsTrigger value="config">配置</TabsTrigger>
          <TabsTrigger value="prometheus">Prometheus</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4">
          {/* Stats Cards */}
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">节点数</CardTitle>
                <Server className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{cluster.node_count}</div>
                <p className="text-xs text-muted-foreground">
                  集群中的节点总数
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Pod 数</CardTitle>
                <Database className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{cluster.pod_count}</div>
                <p className="text-xs text-muted-foreground">
                  集群中的 Pod 总数
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">版本</CardTitle>
                <Cloud className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {cluster.version || 'N/A'}
                </div>
                <p className="text-xs text-muted-foreground">
                  Kubernetes 版本
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">最后连接</CardTitle>
                <Clock className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {cluster.last_connected_at
                    ? formatDistanceToNow(new Date(cluster.last_connected_at), {
                        addSuffix: true,
                        locale: zhCN,
                      })
                    : 'Never'}
                </div>
                <p className="text-xs text-muted-foreground">
                  上次成功连接时间
                </p>
              </CardContent>
            </Card>
          </div>

          {/* Details */}
          <Card>
            <CardHeader>
              <CardTitle>集群信息</CardTitle>
              <CardDescription>查看集群的详细信息</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <div>
                  <label className="text-sm font-medium text-muted-foreground">
                    集群 ID
                  </label>
                  <p className="text-sm font-mono">{cluster.id}</p>
                </div>

                <div>
                  <label className="text-sm font-medium text-muted-foreground">
                    集群名称
                  </label>
                  <p className="text-sm">{cluster.name}</p>
                </div>

                <div>
                  <label className="text-sm font-medium text-muted-foreground">
                    类型
                  </label>
                  <p className="text-sm">
                    {cluster.in_cluster ? 'In-Cluster' : 'External'}
                  </p>
                </div>

                <div>
                  <label className="text-sm font-medium text-muted-foreground">
                    状态
                  </label>
                  <p className="text-sm">
                    {cluster.enable ? '已启用' : '已禁用'}
                  </p>
                </div>

                <div>
                  <label className="text-sm font-medium text-muted-foreground">
                    创建时间
                  </label>
                  <p className="text-sm">
                    {new Date(cluster.created_at).toLocaleString('zh-CN')}
                  </p>
                </div>

                <div>
                  <label className="text-sm font-medium text-muted-foreground">
                    更新时间
                  </label>
                  <p className="text-sm">
                    {new Date(cluster.updated_at).toLocaleString('zh-CN')}
                  </p>
                </div>
              </div>

              {cluster.description && (
                <>
                  <Separator />
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">
                      描述
                    </label>
                    <p className="text-sm mt-1">{cluster.description}</p>
                  </div>
                </>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="config" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Kubeconfig</CardTitle>
              <CardDescription>
                集群的 Kubernetes 配置（敏感信息已隐藏）
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="rounded-md bg-muted p-4 font-mono text-sm">
                <pre className="overflow-x-auto">
                  {cluster.config ? '*** HIDDEN ***' : '未配置'}
                </pre>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="prometheus" className="space-y-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>Prometheus 配置</CardTitle>
                  <CardDescription>
                    集群的 Prometheus 监控配置
                  </CardDescription>
                </div>
                <Button
                  variant="outline"
                  onClick={handlePrometheusRediscover}
                  disabled={rediscoverMutation.isPending}
                >
                  {rediscoverMutation.isPending ? (
                    <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    <RefreshCw className="mr-2 h-4 w-4" />
                  )}
                  重新检测
                </Button>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <label className="text-sm font-medium text-muted-foreground">
                  Prometheus URL
                </label>
                {cluster.prometheus_url ? (
                  <div className="mt-1 flex items-center gap-2">
                    <p className="text-sm font-mono">
                      {cluster.prometheus_url}
                    </p>
                    <Badge variant="secondary">
                      <Check className="mr-1 h-3 w-3" />
                      已配置
                    </Badge>
                  </div>
                ) : (
                  <div className="mt-1">
                    <Badge variant="outline">未配置</Badge>
                    <p className="text-xs text-muted-foreground mt-1">
                      点击"重新检测"按钮自动发现 Prometheus 实例
                    </p>
                  </div>
                )}
              </div>

              {cluster.prometheus_url && (
                <div className="rounded-md bg-muted p-4">
                  <h4 className="text-sm font-medium mb-2">监控信息</h4>
                  <p className="text-sm text-muted-foreground">
                    Prometheus 已成功配置，可以在
                    <Link
                      to={`/k8s/monitoring?cluster=${cluster.name}`}
                      className="text-primary hover:underline mx-1"
                    >
                      监控页面
                    </Link>
                    查看集群指标
                  </p>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除集群</AlertDialogTitle>
            <AlertDialogDescription>
              您确定要删除集群 <strong>{cluster.name}</strong> 吗？
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
