import { useState } from 'react'
import {
  DockerInstance,
  useDeleteDockerInstance,
  useDockerInstances,
  useTestDockerConnection,
} from '@/services/docker-api'
import {
  IconAlertCircle,
  IconArchive,
  IconCheck,
  IconDotsVertical,
  IconEdit,
  IconPlus,
  IconPlugConnected,
  IconServer,
  IconTrash,
  IconX,
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Skeleton } from '@/components/ui/skeleton'

const getStatusBadge = (status: string) => {
  switch (status) {
    case 'online':
      return (
        <Badge className="bg-green-500">
          <IconCheck className="w-3 h-3 mr-1" /> 在线
        </Badge>
      )
    case 'offline':
      return (
        <Badge variant="secondary">
          <IconX className="w-3 h-3 mr-1" /> 离线
        </Badge>
      )
    case 'archived':
      return (
        <Badge variant="outline">
          <IconArchive className="w-3 h-3 mr-1" /> 已归档
        </Badge>
      )
    default:
      return <Badge variant="outline">{status}</Badge>
  }
}

export function DockerInstanceList() {
  const navigate = useNavigate()
  const [deleteId, setDeleteId] = useState<string | null>(null)
  const [searchTerm, setSearchTerm] = useState('')

  const { data, isLoading, error } = useDockerInstances()
  const deleteMutation = useDeleteDockerInstance()
  const testMutation = useTestDockerConnection()

  const instances = data?.data?.instances || []
  const filteredInstances = instances.filter(
    (instance) =>
      instance.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      instance.agent_name?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      instance.host.toLowerCase().includes(searchTerm.toLowerCase())
  )

  const handleDelete = async () => {
    if (!deleteId) return

    try {
      await deleteMutation.mutateAsync(deleteId)
      toast.success('Docker 实例已删除')
      setDeleteId(null)
    } catch (error: any) {
      toast.error(error?.message || '无法删除实例')
    }
  }

  const handleTest = async (instance: DockerInstance) => {
    try {
      const result = await testMutation.mutateAsync(instance.id)
      toast.success(`${instance.name}: ${result.data.message}`)
    } catch (error: any) {
      toast.error(
        error?.response?.data?.error || '无法连接到 Docker daemon'
      )
    }
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-3xl font-bold">Docker 实例</h1>
        </div>
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-64" />
          ))}
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-3xl font-bold">Docker 实例</h1>
        </div>
        <Card>
          <CardHeader>
            <CardTitle className="text-destructive">加载失败</CardTitle>
            <CardDescription>
              {(error as any)?.message || '无法加载实例列表'}
            </CardDescription>
          </CardHeader>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-3xl font-bold">Docker 实例</h1>
          <p className="text-muted-foreground mt-2">
            管理远程 Docker 实例，查看容器和镜像
          </p>
        </div>
        <div className="flex gap-2">
          <input
            type="text"
            placeholder="搜索实例..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="px-3 py-2 border rounded-md w-64"
          />
          <Button onClick={() => navigate('/docker/instances/new')}>
            <IconPlus className="w-4 h-4 mr-2" />
            新建实例
          </Button>
        </div>
      </div>

      {filteredInstances.length === 0 ? (
        <Card>
          <CardHeader>
            <CardTitle>
              {searchTerm ? '没有匹配的实例' : '还没有 Docker 实例'}
            </CardTitle>
            <CardDescription>
              {searchTerm
                ? '尝试使用其他搜索条件'
                : '点击"新建实例"按钮开始添加远程 Docker 实例'}
            </CardDescription>
          </CardHeader>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {filteredInstances.map((instance) => (
            <Card
              key={instance.id}
              className="hover:shadow-lg transition-shadow"
            >
              <CardHeader>
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-2">
                    <IconServer className="w-5 h-5" />
                    <CardTitle className="text-xl">{instance.name}</CardTitle>
                  </div>
                  <div className="flex gap-2">
                    {getStatusBadge(instance.status)}
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon" className="h-8 w-8">
                          <IconDotsVertical className="w-4 h-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem
                          onClick={() => navigate(`/docker/instances/${instance.id}`)}
                        >
                          <IconEdit className="w-4 h-4 mr-2" />
                          查看详情
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => handleTest(instance)}>
                          <IconPlugConnected className="w-4 h-4 mr-2" />
                          测试连接
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          onClick={() => setDeleteId(instance.id)}
                          className="text-destructive"
                        >
                          <IconTrash className="w-4 h-4 mr-2" />
                          删除实例
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                </div>
                <CardDescription className="mt-2 line-clamp-2">
                  {instance.description || '无描述'}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  <div className="grid grid-cols-2 gap-2 text-sm">
                    <div className="space-y-1">
                      <span className="text-muted-foreground block">Agent:</span>
                      <span className="font-medium">{instance.agent_name}</span>
                    </div>
                    <div className="space-y-1">
                      <span className="text-muted-foreground block">地址:</span>
                      <span className="font-mono text-xs">
                        {instance.host}:{instance.port}
                      </span>
                    </div>
                  </div>

                  {instance.version && (
                    <div className="flex justify-between text-sm pt-2 border-t">
                      <span className="text-muted-foreground">Docker 版本:</span>
                      <span className="font-medium">{instance.version}</span>
                    </div>
                  )}

                  <div className="grid grid-cols-2 gap-4 pt-2 border-t">
                    <div className="text-center">
                      <div className="text-2xl font-bold text-blue-600">
                        {instance.total_containers ?? '-'}
                      </div>
                      <div className="text-xs text-muted-foreground">
                        容器数
                        {instance.running_containers !== undefined && (
                          <span className="ml-1 text-green-600">
                            ({instance.running_containers} 运行中)
                          </span>
                        )}
                      </div>
                    </div>
                    <div className="text-center">
                      <div className="text-2xl font-bold text-purple-600">
                        {instance.total_images ?? '-'}
                      </div>
                      <div className="text-xs text-muted-foreground">
                        镜像数
                      </div>
                    </div>
                  </div>

                  {instance.last_seen_at && (
                    <div className="text-xs text-muted-foreground text-right pt-2">
                      最后通信: {new Date(instance.last_seen_at).toLocaleString()}
                    </div>
                  )}

                  <Button
                    variant="outline"
                    className="w-full mt-4"
                    onClick={() => navigate(`/docker/instances/${instance.id}`)}
                  >
                    <IconEdit className="w-4 h-4 mr-2" />
                    管理容器和镜像
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <AlertDialog
        open={deleteId !== null}
        onOpenChange={() => setDeleteId(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除</AlertDialogTitle>
            <AlertDialogDescription>
              此操作将永久删除该 Docker 实例记录。实际的 Docker daemon
              不会受到影响。此操作无法撤销。
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
