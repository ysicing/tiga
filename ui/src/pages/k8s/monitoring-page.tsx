import { useSearchParams } from 'react-router-dom'
import { Activity, AlertCircle, CheckCircle, ExternalLink, RefreshCw, Server } from 'lucide-react'
import { toast } from 'sonner'

import { useCluster, usePrometheusRediscover } from '@/services/k8s-api'
import { useCluster as useClusterContext } from '@/contexts/cluster-context'
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
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { ClusterSelector } from '@/components/k8s/cluster-selector'
import { Label } from '@/components/ui/label'

export function MonitoringPage() {
  const [searchParams] = useSearchParams()
  const clusterNameFromUrl = searchParams.get('cluster')

  const { currentCluster, clusters } = useClusterContext()
  const selectedClusterName = clusterNameFromUrl || currentCluster

  const cluster = clusters.find((c: Cluster) => c.name === selectedClusterName)

  const { data, isLoading, error, refetch } = useCluster(String(cluster?.id || ''))
  const rediscoverMutation = usePrometheusRediscover()

  const clusterData = data?.data as Cluster | undefined

  const handleRediscover = async () => {
    if (!cluster?.id) return

    try {
      await rediscoverMutation.mutateAsync(String(cluster.id))
      toast.success('Prometheus 重新检测已启动，请稍后刷新页面查看结果')
      setTimeout(() => refetch(), 3000)
    } catch (error: any) {
      toast.error(error?.message || 'Prometheus 重新检测失败')
    }
  }

  const handleOpenPrometheus = () => {
    if (clusterData?.prometheus_url) {
      window.open(clusterData.prometheus_url, '_blank')
    }
  }

  if (!cluster) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold">Prometheus 监控</h1>
          <p className="text-muted-foreground mt-1">
            查看和管理集群的 Prometheus 监控配置
          </p>
        </div>

        <Alert>
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>未选择集群</AlertTitle>
          <AlertDescription>
            请选择一个集群以查看 Prometheus 监控配置
          </AlertDescription>
        </Alert>

        <div className="flex items-center gap-4">
          <Label className="text-sm font-medium shrink-0">选择集群:</Label>
          <ClusterSelector />
        </div>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold">Prometheus 监控</h1>
          <p className="text-muted-foreground mt-1">
            查看和管理集群的 Prometheus 监控配置
          </p>
        </div>

        <div className="grid gap-6 md:grid-cols-2">
          <Skeleton className="h-48" />
          <Skeleton className="h-48" />
        </div>
      </div>
    )
  }

  if (error || !clusterData) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold">Prometheus 监控</h1>
          <p className="text-muted-foreground mt-1">
            查看和管理集群的 Prometheus 监控配置
          </p>
        </div>

        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>加载失败</AlertTitle>
          <AlertDescription>
            无法加载集群信息，请稍后重试
          </AlertDescription>
        </Alert>
      </div>
    )
  }

  const hasPrometheus = !!clusterData.prometheus_url
  const isHealthy = clusterData.health_status === 'healthy'

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Prometheus 监控</h1>
          <p className="text-muted-foreground mt-1">
            查看和管理集群 {clusterData.name} 的 Prometheus 监控配置
          </p>
        </div>

        <div className="flex items-center gap-2">
          <ClusterSelector variant="compact" />
        </div>
      </div>

      {/* Status Overview */}
      <div className="grid gap-6 md:grid-cols-2">
        {/* Prometheus Status Card */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center gap-2">
                <Activity className="h-5 w-5" />
                Prometheus 状态
              </CardTitle>
              <Button
                variant="outline"
                size="sm"
                onClick={handleRediscover}
                disabled={rediscoverMutation.isPending || !isHealthy}
              >
                {rediscoverMutation.isPending ? (
                  <>
                    <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                    检测中
                  </>
                ) : (
                  <>
                    <RefreshCw className="mr-2 h-4 w-4" />
                    重新检测
                  </>
                )}
              </Button>
            </div>
            <CardDescription>
              Prometheus 实例的发现和配置状态
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">配置状态</span>
              {hasPrometheus ? (
                <Badge className="bg-green-600">
                  <CheckCircle className="mr-1 h-3 w-3" />
                  已配置
                </Badge>
              ) : (
                <Badge variant="outline">
                  <AlertCircle className="mr-1 h-3 w-3" />
                  未配置
                </Badge>
              )}
            </div>

            <Separator />

            {hasPrometheus ? (
              <>
                <div className="space-y-2">
                  <label className="text-sm font-medium">Prometheus URL</label>
                  <div className="flex items-center gap-2">
                    <code className="flex-1 rounded bg-muted px-3 py-2 text-sm font-mono break-all">
                      {clusterData.prometheus_url}
                    </code>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={handleOpenPrometheus}
                      title="在新窗口打开"
                    >
                      <ExternalLink className="h-4 w-4" />
                    </Button>
                  </div>
                </div>

                <Alert>
                  <CheckCircle className="h-4 w-4" />
                  <AlertTitle>Prometheus 已就绪</AlertTitle>
                  <AlertDescription>
                    您可以通过上面的 URL 访问 Prometheus UI 查看集群指标
                  </AlertDescription>
                </Alert>
              </>
            ) : (
              <Alert>
                <AlertCircle className="h-4 w-4" />
                <AlertTitle>未检测到 Prometheus</AlertTitle>
                <AlertDescription className="space-y-2">
                  <p>
                    系统未找到 Prometheus 实例。可能的原因：
                  </p>
                  <ul className="list-disc list-inside space-y-1 text-sm">
                    <li>Prometheus 未安装在集群中</li>
                    <li>Prometheus Service 配置不正确</li>
                    <li>网络连接问题导致无法访问</li>
                  </ul>
                  <div className="pt-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handleRediscover}
                      disabled={rediscoverMutation.isPending || !isHealthy}
                    >
                      <RefreshCw className="mr-2 h-4 w-4" />
                      重新检测
                    </Button>
                  </div>
                </AlertDescription>
              </Alert>
            )}
          </CardContent>
        </Card>

        {/* Cluster Health Card */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Server className="h-5 w-5" />
              集群健康状态
            </CardTitle>
            <CardDescription>
              当前集群的健康状况和统计信息
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-1">
                <label className="text-sm font-medium text-muted-foreground">
                  集群状态
                </label>
                <div className="flex items-center gap-2">
                  {isHealthy ? (
                    <>
                      <div className="h-2 w-2 rounded-full bg-green-500" />
                      <span className="text-sm font-medium">健康</span>
                    </>
                  ) : (
                    <>
                      <div className="h-2 w-2 rounded-full bg-red-500" />
                      <span className="text-sm font-medium">异常</span>
                    </>
                  )}
                </div>
              </div>

              <div className="space-y-1">
                <label className="text-sm font-medium text-muted-foreground">
                  节点数
                </label>
                <p className="text-2xl font-bold">{clusterData.node_count}</p>
              </div>

              <div className="space-y-1">
                <label className="text-sm font-medium text-muted-foreground">
                  Pod 数
                </label>
                <p className="text-2xl font-bold">{clusterData.pod_count}</p>
              </div>

              <div className="space-y-1">
                <label className="text-sm font-medium text-muted-foreground">
                  K8s 版本
                </label>
                <p className="text-sm font-medium">
                  {clusterData.version || 'N/A'}
                </p>
              </div>
            </div>

            <Separator />

            <div className="space-y-2">
              <label className="text-sm font-medium text-muted-foreground">
                最后连接时间
              </label>
              <p className="text-sm">
                {clusterData.last_connected_at
                  ? new Date(clusterData.last_connected_at).toLocaleString('zh-CN')
                  : '从未连接'}
              </p>
            </div>

            {!isHealthy && (
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertTitle>集群不健康</AlertTitle>
                <AlertDescription>
                  集群当前状态异常，Prometheus 自动发现可能无法正常工作
                </AlertDescription>
              </Alert>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Quick Actions */}
      {hasPrometheus && (
        <Card>
          <CardHeader>
            <CardTitle>快速操作</CardTitle>
            <CardDescription>
              常用的 Prometheus 相关操作
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-3">
              <Button
                variant="outline"
                className="justify-start"
                onClick={handleOpenPrometheus}
              >
                <ExternalLink className="mr-2 h-4 w-4" />
                打开 Prometheus UI
              </Button>
              <Button
                variant="outline"
                className="justify-start"
                onClick={() => window.open(`${clusterData.prometheus_url}/graph`, '_blank')}
              >
                <Activity className="mr-2 h-4 w-4" />
                查询指标
              </Button>
              <Button
                variant="outline"
                className="justify-start"
                onClick={() => window.open(`${clusterData.prometheus_url}/alerts`, '_blank')}
              >
                <AlertCircle className="mr-2 h-4 w-4" />
                查看告警
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Help Information */}
      <Card>
        <CardHeader>
          <CardTitle>帮助信息</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <h4 className="font-medium mb-2">自动发现工作原理</h4>
            <p className="text-sm text-muted-foreground">
              系统会自动扫描集群中的 Prometheus Service，优先选择 LoadBalancer、
              NodePort 或 ClusterIP 类型的服务。发现过程通常在集群变为健康状态后 30 秒内完成。
            </p>
          </div>

          <Separator />

          <div>
            <h4 className="font-medium mb-2">手动配置</h4>
            <p className="text-sm text-muted-foreground">
              如果自动发现失败，您可以在集群编辑页面手动配置 Prometheus URL。
              手动配置的 URL 优先级高于自动发现结果。
            </p>
          </div>

          <Separator />

          <div>
            <h4 className="font-medium mb-2">常见问题</h4>
            <ul className="list-disc list-inside space-y-1 text-sm text-muted-foreground">
              <li>确保 Prometheus 已正确安装在集群中</li>
              <li>检查 Prometheus Service 的类型和端口配置</li>
              <li>验证网络连接和防火墙规则</li>
              <li>查看集群健康检查日志获取更多信息</li>
            </ul>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
