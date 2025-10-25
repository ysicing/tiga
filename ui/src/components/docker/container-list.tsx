import { useState } from 'react'
import {
  Container,
  useContainers,
  useStartContainer,
  useStopContainer,
  useRestartContainer,
  useDeleteContainer,
} from '@/services/docker-api'
import {
  IconPlayerPlay,
  IconPlayerStop,
  IconRefresh,
  IconTrash,
  IconTerminal,
  IconFileText,
  IconSearch,
} from '@tabler/icons-react'
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
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ContainerLogsViewer } from '@/components/docker/container-logs-viewer'
import { ContainerTerminal } from '@/components/docker/container-terminal'

interface ContainerListProps {
  instanceId: string
}

const getStateBadge = (state: string) => {
  switch (state.toLowerCase()) {
    case 'running':
      return <Badge className="bg-green-500">运行中</Badge>
    case 'exited':
      return <Badge variant="secondary">已停止</Badge>
    case 'paused':
      return <Badge variant="outline">已暂停</Badge>
    case 'restarting':
      return <Badge className="bg-yellow-500">重启中</Badge>
    case 'dead':
      return <Badge variant="destructive">Dead</Badge>
    default:
      return <Badge variant="outline">{state}</Badge>
  }
}

const formatContainerName = (names: string[]) => {
  if (!names || names.length === 0) return '-'
  return names[0].startsWith('/') ? names[0].substring(1) : names[0]
}

const formatPorts = (ports: any[]) => {
  if (!ports || ports.length === 0) return '-'
  return ports
    .map((p) => {
      if (p.public_port) {
        return `${p.public_port}:${p.private_port}/${p.type}`
      }
      return `${p.private_port}/${p.type}`
    })
    .join(', ')
}

const formatImage = (image: string) => {
  // Shorten long image names
  const parts = image.split('/')
  if (parts.length > 2) {
    return `.../${parts[parts.length - 1]}`
  }
  return image
}

export function ContainerList({ instanceId }: ContainerListProps) {
  const [searchTerm, setSearchTerm] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<{
    id: string
    name: string
  } | null>(null)
  const [logsTarget, setLogsTarget] = useState<{
    id: string
    name: string
  } | null>(null)
  const [terminalTarget, setTerminalTarget] = useState<{
    id: string
    name: string
  } | null>(null)

  const { data, isLoading, error, refetch } = useContainers(instanceId, {
    all: true,
  })

  const startMutation = useStartContainer()
  const stopMutation = useStopContainer()
  const restartMutation = useRestartContainer()
  const deleteMutation = useDeleteContainer()

  const containers = data?.data?.containers || []
  const filteredContainers = containers.filter((container) => {
    const name = formatContainerName(container.names).toLowerCase()
    const image = container.image.toLowerCase()
    const search = searchTerm.toLowerCase()
    return name.includes(search) || image.includes(search)
  })

  const handleStart = async (container: Container) => {
    try {
      await startMutation.mutateAsync({
        instanceId,
        containerId: container.id,
      })
      toast.success(`容器 ${formatContainerName(container.names)} 已启动`)
    } catch (error: any) {
      toast.error(error?.message || '无法启动容器')
    }
  }

  const handleStop = async (container: Container) => {
    try {
      await stopMutation.mutateAsync({
        instanceId,
        containerId: container.id,
        timeout: 10,
      })
      toast.success(`容器 ${formatContainerName(container.names)} 已停止`)
    } catch (error: any) {
      toast.error(error?.message || '无法停止容器')
    }
  }

  const handleRestart = async (container: Container) => {
    try {
      await restartMutation.mutateAsync({
        instanceId,
        containerId: container.id,
        timeout: 10,
      })
      toast.success(`容器 ${formatContainerName(container.names)} 已重启`)
    } catch (error: any) {
      toast.error(error?.message || '无法重启容器')
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return

    try {
      await deleteMutation.mutateAsync({
        instanceId,
        containerId: deleteTarget.id,
        force: true,
        removeVolumes: false,
      })
      toast.success(`容器 ${deleteTarget.name} 已删除`)
      setDeleteTarget(null)
    } catch (error: any) {
      toast.error(error?.message || '无法删除容器')
    }
  }

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>容器列表</CardTitle>
              <CardDescription className="mt-1 flex items-center gap-2">
                <div className="h-4 w-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />
                正在加载容器列表...
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[200px]">名称</TableHead>
                  <TableHead>镜像</TableHead>
                  <TableHead className="w-[100px]">状态</TableHead>
                  <TableHead>端口</TableHead>
                  <TableHead className="text-right w-[200px]">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {[1, 2, 3, 4, 5].map((i) => (
                  <TableRow key={i}>
                    <TableCell>
                      <div className="space-y-2">
                        <Skeleton className="h-4 w-32" />
                        <Skeleton className="h-3 w-24" />
                      </div>
                    </TableCell>
                    <TableCell>
                      <Skeleton className="h-4 w-40" />
                    </TableCell>
                    <TableCell>
                      <Skeleton className="h-6 w-16 rounded-full" />
                    </TableCell>
                    <TableCell>
                      <Skeleton className="h-4 w-28" />
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end gap-1">
                        {[1, 2, 3, 4, 5].map((j) => (
                          <Skeleton key={j} className="h-8 w-8 rounded-md" />
                        ))}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-destructive">加载失败</CardTitle>
          <CardDescription>
            {(error as any)?.message || '无法加载容器列表'}
          </CardDescription>
        </CardHeader>
      </Card>
    )
  }

  return (
    <>
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>容器列表</CardTitle>
              <CardDescription className="mt-1">
                共 {containers.length} 个容器
                {filteredContainers.length !== containers.length &&
                  ` (显示 ${filteredContainers.length} 个)`}
              </CardDescription>
            </div>
            <div className="flex gap-2">
              <div className="relative">
                <IconSearch className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                <Input
                  placeholder="搜索容器或镜像..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-9 w-64"
                />
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => refetch()}
                disabled={isLoading}
              >
                <IconRefresh className={`w-4 h-4 ${isLoading ? 'animate-spin' : ''}`} />
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {filteredContainers.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              {searchTerm ? '没有匹配的容器' : '还没有容器'}
            </div>
          ) : (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-[200px]">名称</TableHead>
                    <TableHead>镜像</TableHead>
                    <TableHead className="w-[100px]">状态</TableHead>
                    <TableHead>端口</TableHead>
                    <TableHead className="text-right w-[200px]">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredContainers.map((container) => {
                    const isRunning = container.state.toLowerCase() === 'running'
                    const name = formatContainerName(container.names)

                    return (
                      <TableRow key={container.id}>
                        <TableCell className="font-medium">
                          <div>
                            <div className="font-mono text-sm">{name}</div>
                            <div className="text-xs text-muted-foreground truncate max-w-[180px]">
                              {container.id.substring(0, 12)}
                            </div>
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="font-mono text-xs" title={container.image}>
                            {formatImage(container.image)}
                          </div>
                        </TableCell>
                        <TableCell>{getStateBadge(container.state)}</TableCell>
                        <TableCell className="font-mono text-xs">
                          {formatPorts(container.ports)}
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex justify-end gap-1">
                            {!isRunning ? (
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-8 w-8"
                                onClick={() => handleStart(container)}
                                disabled={startMutation.isPending}
                                title="启动容器"
                              >
                                <IconPlayerPlay className="w-4 h-4 text-green-600" />
                              </Button>
                            ) : (
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-8 w-8"
                                onClick={() => handleStop(container)}
                                disabled={stopMutation.isPending}
                                title="停止容器"
                              >
                                <IconPlayerStop className="w-4 h-4 text-red-600" />
                              </Button>
                            )}
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-8 w-8"
                              onClick={() => handleRestart(container)}
                              disabled={restartMutation.isPending}
                              title="重启容器"
                            >
                              <IconRefresh className="w-4 h-4" />
                            </Button>
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-8 w-8"
                              onClick={() => setLogsTarget({ id: container.id, name })}
                              title="查看日志"
                            >
                              <IconFileText className="w-4 h-4" />
                            </Button>
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-8 w-8"
                              disabled={!isRunning}
                              onClick={() => setTerminalTarget({ id: container.id, name })}
                              title="进入终端"
                            >
                              <IconTerminal className="w-4 h-4" />
                            </Button>
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-8 w-8"
                              onClick={() =>
                                setDeleteTarget({ id: container.id, name })
                              }
                              title="删除容器"
                            >
                              <IconTrash className="w-4 h-4 text-destructive" />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      <AlertDialog
        open={deleteTarget !== null}
        onOpenChange={() => setDeleteTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除容器</AlertDialogTitle>
            <AlertDialogDescription>
              此操作将强制删除容器 <strong>{deleteTarget?.name}</strong>。
              容器将被永久移除，但挂载的卷不会被删除。
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

      {/* Logs Dialog */}
      <Dialog open={logsTarget !== null} onOpenChange={() => setLogsTarget(null)}>
        <DialogContent className="max-w-6xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>容器日志</DialogTitle>
            <DialogDescription>
              查看容器 {logsTarget?.name} 的实时日志输出
            </DialogDescription>
          </DialogHeader>
          {logsTarget && (
            <ContainerLogsViewer
              instanceId={instanceId}
              containerId={logsTarget.id}
              containerName={logsTarget.name}
            />
          )}
        </DialogContent>
      </Dialog>

      {/* Terminal Dialog */}
      <Dialog
        open={terminalTarget !== null}
        onOpenChange={() => setTerminalTarget(null)}
      >
        <DialogContent className="max-w-7xl max-h-[95vh] p-0">
          <DialogHeader className="sr-only">
            <DialogTitle>容器终端</DialogTitle>
            <DialogDescription>
              连接到容器 {terminalTarget?.name} 的交互式终端
            </DialogDescription>
          </DialogHeader>
          {terminalTarget && (
            <ContainerTerminal
              instanceId={instanceId}
              containerId={terminalTarget.id}
              containerName={terminalTarget.name}
              open={true}
            />
          )}
        </DialogContent>
      </Dialog>
    </>
  )
}
