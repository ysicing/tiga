import { useState } from 'react'
import {
  DatabaseInstance,
  useDeleteInstance,
  useInstances,
  useTestConnection,
} from '@/services/database-api'
import {
  IconAlertCircle,
  IconCheck,
  IconClock,
  IconDatabase,
  IconEdit,
  IconPlugConnected,
  IconPlus,
  IconServer2,
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
    case 'pending':
      return (
        <Badge variant="outline">
          <IconClock className="w-3 h-3 mr-1" /> 待检查
        </Badge>
      )
    case 'error':
      return (
        <Badge variant="destructive">
          <IconAlertCircle className="w-3 h-3 mr-1" /> 错误
        </Badge>
      )
    default:
      return <Badge variant="outline">{status}</Badge>
  }
}

const getTypeIcon = (type: string) => {
  switch (type.toLowerCase()) {
    case 'redis':
      return <IconServer2 className="w-5 h-5" />
    default:
      return <IconDatabase className="w-5 h-5" />
  }
}

export function DatabaseInstanceList() {
  const navigate = useNavigate()
  const [deleteId, setDeleteId] = useState<number | null>(null)

  const { data, isLoading, error } = useInstances()
  const deleteMutation = useDeleteInstance()
  const testMutation = useTestConnection()

  const instances = data?.data?.instances || []

  const handleDelete = async () => {
    if (!deleteId) return

    try {
      await deleteMutation.mutateAsync(deleteId)
      toast.success('数据库实例已删除')
      setDeleteId(null)
    } catch (error: any) {
      toast.error(error?.message || '无法删除实例')
    }
  }

  const handleTest = async (instance: DatabaseInstance) => {
    try {
      const result = await testMutation.mutateAsync(instance.id)
      toast.success(`${instance.name}: ${result.data.message}`)
    } catch (error: any) {
      toast.error(error?.response?.data?.error || '无法连接到数据库')
    }
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-3xl font-bold">数据库实例</h1>
        </div>
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-48" />
          ))}
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-3xl font-bold">数据库实例</h1>
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
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">数据库实例</h1>
          <p className="text-muted-foreground mt-2">
            管理 MySQL、PostgreSQL 和 Redis 数据库实例
          </p>
        </div>
        <Button onClick={() => navigate('/dbs/instances/new')}>
          <IconPlus className="w-4 h-4 mr-2" />
          新建实例
        </Button>
      </div>

      {instances.length === 0 ? (
        <Card>
          <CardHeader>
            <CardTitle>还没有数据库实例</CardTitle>
            <CardDescription>
              点击"新建实例"按钮开始添加 MySQL、PostgreSQL 或 Redis 实例
            </CardDescription>
          </CardHeader>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {instances.map((instance) => (
            <Card
              key={instance.id}
              className="hover:shadow-lg transition-shadow"
            >
              <CardHeader>
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-2">
                    {getTypeIcon(instance.type)}
                    <CardTitle className="text-xl">{instance.name}</CardTitle>
                  </div>
                  {getStatusBadge(instance.status)}
                </div>
                <CardDescription className="mt-2">
                  {instance.description || '无描述'}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">类型:</span>
                    <span className="font-medium">
                      {instance.type.toUpperCase()}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">地址:</span>
                    <span className="font-mono">
                      {instance.host}:{instance.port}
                    </span>
                  </div>
                  {instance.version && (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">版本:</span>
                      <span>{instance.version}</span>
                    </div>
                  )}
                </div>

                <div className="flex gap-2 mt-4">
                  <Button
                    variant="outline"
                    size="sm"
                    className="flex-1"
                    onClick={() => navigate(`/database/${instance.id}`)}
                  >
                    <IconEdit className="w-4 h-4 mr-1" />
                    管理
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleTest(instance)}
                    disabled={testMutation.isPending}
                  >
                    <IconPlugConnected className="w-4 h-4" />
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setDeleteId(instance.id)}
                  >
                    <IconTrash className="w-4 h-4 text-destructive" />
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
              此操作将永久删除该数据库实例及其所有相关数据（数据库、用户、权限）。
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
