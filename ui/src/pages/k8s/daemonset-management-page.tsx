import { useState } from 'react'
import { useDaemonSets, useRestartDaemonSet } from '@/services/k8s-api'
import { IconRefresh, IconReload } from '@tabler/icons-react'
import { format } from 'date-fns'
import { useParams } from 'react-router-dom'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

export function DaemonSetManagementPage() {
  const { clusterId } = useParams<{ clusterId: string }>()
  const [selectedNamespace, setSelectedNamespace] = useState<
    string | undefined
  >(undefined)
  const [restartDialogOpen, setRestartDialogOpen] = useState(false)
  const [selectedDaemonSet, setSelectedDaemonSet] = useState<{
    namespace: string
    name: string
  } | null>(null)

  const { data, isLoading, error, refetch } = useDaemonSets(
    clusterId || '',
    selectedNamespace
  )
  const restartMutation = useRestartDaemonSet()

  const daemonSets = Array.isArray(data?.data) ? data.data : []

  // Extract unique namespaces
  const namespaces = Array.from(
    new Set(daemonSets.map((ds) => ds.namespace))
  ).sort()

  const handleRestart = (namespace: string, name: string) => {
    setSelectedDaemonSet({ namespace, name })
    setRestartDialogOpen(true)
  }

  const executeRestart = async () => {
    if (!selectedDaemonSet || !clusterId) return

    try {
      await restartMutation.mutateAsync({
        clusterId,
        namespace: selectedDaemonSet.namespace,
        name: selectedDaemonSet.name,
      })
      toast.success(`DaemonSet ${selectedDaemonSet.name} 重启已触发`)
      setRestartDialogOpen(false)
      setSelectedDaemonSet(null)
    } catch (err) {
      toast.error(`重启失败: ${(err as Error).message}`)
    }
  }

  if (!clusterId) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-destructive">缺少集群 ID</CardTitle>
          <CardDescription>请从集群列表选择一个集群</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <Skeleton className="h-8 w-64" />
          <Skeleton className="h-10 w-32" />
        </div>
        <Card>
          <CardHeader>
            <Skeleton className="h-6 w-32" />
          </CardHeader>
          <CardContent className="space-y-4">
            {[1, 2, 3].map((i) => (
              <Skeleton key={i} className="h-16 w-full" />
            ))}
          </CardContent>
        </Card>
      </div>
    )
  }

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-destructive">加载失败</CardTitle>
          <CardDescription>
            {(error as Error)?.message || '无法加载 DaemonSet 列表'}
          </CardDescription>
        </CardHeader>
      </Card>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">DaemonSet 管理</h1>
          <p className="text-muted-foreground mt-2">
            管理 Kubernetes DaemonSet 工作负载的重启
          </p>
        </div>
        <Button onClick={() => refetch()} variant="outline">
          <IconRefresh className="w-4 h-4 mr-2" />
          刷新
        </Button>
      </div>

      {/* Filters */}
      <Card>
        <CardHeader>
          <CardTitle>筛选条件</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-4">
            <div className="w-64">
              <Label>命名空间</Label>
              <Select
                value={selectedNamespace || 'all'}
                onValueChange={(value) =>
                  setSelectedNamespace(value === 'all' ? undefined : value)
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择命名空间" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部命名空间</SelectItem>
                  {namespaces.map((ns) => (
                    <SelectItem key={ns} value={ns}>
                      {ns}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* DaemonSet List */}
      <Card>
        <CardHeader>
          <CardTitle>
            DaemonSet 列表
            <span className="text-muted-foreground text-sm font-normal ml-2">
              (共 {daemonSets.length} 个)
            </span>
          </CardTitle>
        </CardHeader>
        <CardContent>
          {daemonSets.length === 0 ? (
            <div className="text-center py-12 text-muted-foreground">
              暂无 DaemonSet
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>名称</TableHead>
                  <TableHead>命名空间</TableHead>
                  <TableHead>节点状态</TableHead>
                  <TableHead>创建时间</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {daemonSets.map((ds) => (
                  <TableRow key={`${ds.namespace}/${ds.name}`}>
                    <TableCell className="font-medium font-mono">
                      {ds.name}
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">{ds.namespace}</Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <Badge
                          variant={
                            ds.number_ready === ds.desired_number_scheduled
                              ? 'default'
                              : 'secondary'
                          }
                        >
                          {ds.number_ready}/{ds.desired_number_scheduled} Ready
                        </Badge>
                        {ds.current_number_scheduled !==
                          ds.desired_number_scheduled && (
                          <Badge variant="outline">
                            {ds.current_number_scheduled}/
                            {ds.desired_number_scheduled} Scheduled
                          </Badge>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="text-sm font-mono">
                      {format(new Date(ds.created_at), 'yyyy-MM-dd HH:mm:ss')}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleRestart(ds.namespace, ds.name)}
                      >
                        <IconReload className="w-4 h-4 mr-1" />
                        重启
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Restart Dialog */}
      <Dialog open={restartDialogOpen} onOpenChange={setRestartDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>重启 DaemonSet</DialogTitle>
            <DialogDescription>
              这将触发 DaemonSet 的滚动重启，所有 Pod 将被重新创建
            </DialogDescription>
          </DialogHeader>
          {selectedDaemonSet && (
            <div className="space-y-2">
              <Label className="text-sm text-muted-foreground">名称</Label>
              <div className="font-medium font-mono">
                {selectedDaemonSet.namespace}/{selectedDaemonSet.name}
              </div>
            </div>
          )}
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setRestartDialogOpen(false)}
            >
              取消
            </Button>
            <Button
              onClick={executeRestart}
              disabled={restartMutation.isPending}
              variant="destructive"
            >
              {restartMutation.isPending ? '执行中...' : '确认重启'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
