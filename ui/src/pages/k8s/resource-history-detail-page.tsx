import { useState } from 'react'
import { useResourceHistoryDetail } from '@/services/k8s-api'
import {
  IconArrowLeft,
  IconCheck,
  IconClock,
  IconCode,
  IconUser,
  IconX,
} from '@tabler/icons-react'
import { format } from 'date-fns'
import { useNavigate, useParams } from 'react-router-dom'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

const operationTypeColors: Record<
  string,
  'default' | 'secondary' | 'destructive' | 'outline'
> = {
  create: 'default',
  update: 'secondary',
  delete: 'destructive',
  apply: 'outline',
}

const operationTypeLabels: Record<string, string> = {
  create: '创建',
  update: '更新',
  delete: '删除',
  apply: '应用',
}

export function ResourceHistoryDetailPage() {
  const navigate = useNavigate()
  const { clusterId, historyId } = useParams<{
    clusterId: string
    historyId: string
  }>()

  const [activeTab, setActiveTab] = useState<string>('current')

  const { data, isLoading, error } = useResourceHistoryDetail(
    clusterId || '',
    historyId || ''
  )

  const history = data?.data

  if (!clusterId || !historyId) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-destructive">缺少参数</CardTitle>
          <CardDescription>请从资源历史列表选择一个记录</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Skeleton className="h-10 w-32" />
          <Skeleton className="h-8 w-64" />
        </div>
        <Card>
          <CardHeader>
            <Skeleton className="h-6 w-48" />
          </CardHeader>
          <CardContent className="space-y-4">
            {[1, 2, 3].map((i) => (
              <Skeleton key={i} className="h-24 w-full" />
            ))}
          </CardContent>
        </Card>
      </div>
    )
  }

  if (error || !history) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-destructive">加载失败</CardTitle>
          <CardDescription>
            {(error as Error)?.message || '无法加载资源历史详情'}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Button
            variant="outline"
            onClick={() => navigate(`/k8s/clusters/${clusterId}/resource-history`)}
          >
            <IconArrowLeft className="w-4 h-4 mr-2" />
            返回列表
          </Button>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button
          variant="outline"
          size="sm"
          onClick={() => navigate(`/k8s/clusters/${clusterId}/resource-history`)}
        >
          <IconArrowLeft className="w-4 h-4 mr-2" />
          返回
        </Button>
        <div>
          <h1 className="text-3xl font-bold">资源操作详情</h1>
          <p className="text-muted-foreground mt-2">
            {history.resource_type}/{history.resource_name}
          </p>
        </div>
      </div>

      {/* Metadata Card */}
      <Card>
        <CardHeader>
          <CardTitle>操作信息</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {/* Operation Type */}
            <div className="space-y-2">
              <div className="text-sm text-muted-foreground flex items-center gap-2">
                <IconCode className="w-4 h-4" />
                操作类型
              </div>
              <Badge
                variant={
                  operationTypeColors[history.operation_type] || 'secondary'
                }
              >
                {operationTypeLabels[history.operation_type] ||
                  history.operation_type}
              </Badge>
            </div>

            {/* Status */}
            <div className="space-y-2">
              <div className="text-sm text-muted-foreground">执行状态</div>
              {history.success ? (
                <Badge className="bg-green-500">
                  <IconCheck className="w-3 h-3 mr-1" /> 成功
                </Badge>
              ) : (
                <Badge variant="destructive">
                  <IconX className="w-3 h-3 mr-1" /> 失败
                </Badge>
              )}
            </div>

            {/* Timestamp */}
            <div className="space-y-2">
              <div className="text-sm text-muted-foreground flex items-center gap-2">
                <IconClock className="w-4 h-4" />
                操作时间
              </div>
              <div className="text-sm font-mono">
                {format(new Date(history.created_at), 'yyyy-MM-dd HH:mm:ss')}
              </div>
            </div>

            {/* Operator */}
            <div className="space-y-2">
              <div className="text-sm text-muted-foreground flex items-center gap-2">
                <IconUser className="w-4 h-4" />
                操作者
              </div>
              <div>
                <div className="font-medium">{history.operator_name}</div>
                <div className="text-xs text-muted-foreground font-mono">
                  {history.operator_id.substring(0, 8)}
                </div>
              </div>
            </div>

            {/* Namespace */}
            {history.namespace && (
              <div className="space-y-2">
                <div className="text-sm text-muted-foreground">命名空间</div>
                <div className="font-medium font-mono">{history.namespace}</div>
              </div>
            )}

            {/* API Group */}
            {history.api_group && (
              <div className="space-y-2">
                <div className="text-sm text-muted-foreground">API 版本</div>
                <div className="font-medium font-mono">
                  {history.api_group}/{history.api_version}
                </div>
              </div>
            )}
          </div>

          {/* Error Message */}
          {history.error_message && (
            <>
              <Separator className="my-4" />
              <div className="space-y-2">
                <div className="text-sm font-medium text-destructive">
                  错误信息
                </div>
                <div className="rounded-md bg-destructive/10 p-4 text-sm text-destructive font-mono">
                  {history.error_message}
                </div>
              </div>
            </>
          )}
        </CardContent>
      </Card>

      {/* YAML Content */}
      <Card>
        <CardHeader>
          <CardTitle>资源配置</CardTitle>
          <CardDescription>
            {history.previous_yaml
              ? '查看当前配置和之前的配置对比'
              : '查看资源的 YAML 配置'}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {history.previous_yaml ? (
            <Tabs value={activeTab} onValueChange={setActiveTab}>
              <TabsList className="grid w-full grid-cols-2">
                <TabsTrigger value="current">当前配置</TabsTrigger>
                <TabsTrigger value="previous">之前配置</TabsTrigger>
              </TabsList>
              <TabsContent value="current" className="mt-4">
                <div className="rounded-md bg-muted p-4 overflow-auto max-h-[600px]">
                  <pre className="text-sm font-mono">
                    <code>{history.resource_yaml}</code>
                  </pre>
                </div>
              </TabsContent>
              <TabsContent value="previous" className="mt-4">
                <div className="rounded-md bg-muted p-4 overflow-auto max-h-[600px]">
                  <pre className="text-sm font-mono">
                    <code>{history.previous_yaml}</code>
                  </pre>
                </div>
              </TabsContent>
            </Tabs>
          ) : (
            <div className="rounded-md bg-muted p-4 overflow-auto max-h-[600px]">
              <pre className="text-sm font-mono">
                <code>{history.resource_yaml}</code>
              </pre>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
