import { useState } from 'react'
import {
  useStatefulSets,
  useRestartStatefulSet,
  useScaleStatefulSet,
} from '@/services/k8s-api'
import { IconRefresh, IconReload, IconScale } from '@tabler/icons-react'
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
import { Input } from '@/components/ui/input'
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

export function StatefulSetManagementPage() {
  const { clusterId } = useParams<{ clusterId: string }>()
  const [selectedNamespace, setSelectedNamespace] = useState<
    string | undefined
  >(undefined)
  const [scaleDialogOpen, setScaleDialogOpen] = useState(false)
  const [restartDialogOpen, setRestartDialogOpen] = useState(false)
  const [selectedStatefulSet, setSelectedStatefulSet] = useState<{
    namespace: string
    name: string
    currentReplicas: number
  } | null>(null)
  const [targetReplicas, setTargetReplicas] = useState<number>(0)

  const { data, isLoading, error, refetch } = useStatefulSets(
    clusterId || '',
    selectedNamespace
  )
  const scaleMutation = useScaleStatefulSet()
  const restartMutation = useRestartStatefulSet()

  const statefulSets = Array.isArray(data?.data) ? data.data : []

  // Extract unique namespaces
  const namespaces = Array.from(
    new Set(statefulSets.map((sts) => sts.namespace))
  ).sort()

  const handleScale = (namespace: string, name: string, replicas: number) => {
    setSelectedStatefulSet({ namespace, name, currentReplicas: replicas })
    setTargetReplicas(replicas)
    setScaleDialogOpen(true)
  }

  const handleRestart = (namespace: string, name: string) => {
    setSelectedStatefulSet({ namespace, name, currentReplicas: 0 })
    setRestartDialogOpen(true)
  }

  const executeScale = async () => {
    if (!selectedStatefulSet || !clusterId) return

    try {
      await scaleMutation.mutateAsync({
        clusterId,
        namespace: selectedStatefulSet.namespace,
        name: selectedStatefulSet.name,
        replicas: targetReplicas,
      })
      toast.success(
        `StatefulSet ${selectedStatefulSet.name} 已成功扩缩容至 ${targetReplicas} 副本`
      )
      setScaleDialogOpen(false)
      setSelectedStatefulSet(null)
    } catch (err) {
      toast.error(`扩缩容失败: ${(err as Error).message}`)
    }
  }

  const executeRestart = async () => {
    if (!selectedStatefulSet || !clusterId) return

    try {
      await restartMutation.mutateAsync({
        clusterId,
        namespace: selectedStatefulSet.namespace,
        name: selectedStatefulSet.name,
      })
      toast.success(`StatefulSet ${selectedStatefulSet.name} 重启已触发`)
      setRestartDialogOpen(false)
      setSelectedStatefulSet(null)
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
            {(error as Error)?.message || '无法加载 StatefulSet 列表'}
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
          <h1 className="text-3xl font-bold">StatefulSet 管理</h1>
          <p className="text-muted-foreground mt-2">
            管理 Kubernetes StatefulSet 工作负载的扩缩容和重启
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

      {/* StatefulSet List */}
      <Card>
        <CardHeader>
          <CardTitle>
            StatefulSet 列表
            <span className="text-muted-foreground text-sm font-normal ml-2">
              (共 {statefulSets.length} 个)
            </span>
          </CardTitle>
        </CardHeader>
        <CardContent>
          {statefulSets.length === 0 ? (
            <div className="text-center py-12 text-muted-foreground">
              暂无 StatefulSet
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>名称</TableHead>
                  <TableHead>命名空间</TableHead>
                  <TableHead>副本状态</TableHead>
                  <TableHead>创建时间</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {statefulSets.map((sts) => (
                  <TableRow key={`${sts.namespace}/${sts.name}`}>
                    <TableCell className="font-medium font-mono">
                      {sts.name}
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">{sts.namespace}</Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <Badge
                          variant={
                            sts.ready_replicas === sts.replicas
                              ? 'default'
                              : 'secondary'
                          }
                        >
                          {sts.ready_replicas}/{sts.replicas} Ready
                        </Badge>
                        {sts.current_replicas !== sts.replicas && (
                          <Badge variant="outline">
                            {sts.current_replicas}/{sts.replicas} Current
                          </Badge>
                        )}
                        {sts.updated_replicas !== sts.replicas && (
                          <Badge variant="outline">
                            {sts.updated_replicas}/{sts.replicas} Updated
                          </Badge>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="text-sm font-mono">
                      {format(new Date(sts.created_at), 'yyyy-MM-dd HH:mm:ss')}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() =>
                            handleScale(sts.namespace, sts.name, sts.replicas)
                          }
                        >
                          <IconScale className="w-4 h-4 mr-1" />
                          扩缩容
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleRestart(sts.namespace, sts.name)}
                        >
                          <IconReload className="w-4 h-4 mr-1" />
                          重启
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Scale Dialog */}
      <Dialog open={scaleDialogOpen} onOpenChange={setScaleDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>扩缩容 StatefulSet</DialogTitle>
            <DialogDescription>
              调整 StatefulSet 的副本数量
            </DialogDescription>
          </DialogHeader>
          {selectedStatefulSet && (
            <div className="space-y-4">
              <div>
                <Label className="text-sm text-muted-foreground">名称</Label>
                <div className="font-medium font-mono">
                  {selectedStatefulSet.namespace}/{selectedStatefulSet.name}
                </div>
              </div>
              <div>
                <Label className="text-sm text-muted-foreground">
                  当前副本数
                </Label>
                <div className="font-medium">
                  {selectedStatefulSet.currentReplicas}
                </div>
              </div>
              <div>
                <Label htmlFor="replicas">目标副本数</Label>
                <Input
                  id="replicas"
                  type="number"
                  min="0"
                  value={targetReplicas}
                  onChange={(e) => setTargetReplicas(parseInt(e.target.value))}
                />
              </div>
            </div>
          )}
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setScaleDialogOpen(false)}
            >
              取消
            </Button>
            <Button onClick={executeScale} disabled={scaleMutation.isPending}>
              {scaleMutation.isPending ? '执行中...' : '确认扩缩容'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Restart Dialog */}
      <Dialog open={restartDialogOpen} onOpenChange={setRestartDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>重启 StatefulSet</DialogTitle>
            <DialogDescription>
              这将触发 StatefulSet 的滚动重启，所有 Pod 将被重新创建
            </DialogDescription>
          </DialogHeader>
          {selectedStatefulSet && (
            <div className="space-y-2">
              <Label className="text-sm text-muted-foreground">名称</Label>
              <div className="font-medium font-mono">
                {selectedStatefulSet.namespace}/{selectedStatefulSet.name}
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
