import { useState } from 'react'
import {
  Volume,
  useVolumes,
  useDeleteVolume,
  usePruneVolumes,
  useCreateVolume,
} from '@/services/docker-api'
import {
  IconTrash,
  IconRefresh,
  IconPlus,
  IconSearch,
  IconAlertTriangle,
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
import { Label } from '@/components/ui/label'
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
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

interface VolumeListProps {
  instanceId: string
}

const formatBytes = (bytes?: number) => {
  if (!bytes) return '-'
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  if (bytes === 0) return '0 B'
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(2)} ${sizes[i]}`
}

const getScopeBadge = (scope: string) => {
  switch (scope.toLowerCase()) {
    case 'local':
      return <Badge variant="secondary">本地</Badge>
    case 'global':
      return <Badge>全局</Badge>
    default:
      return <Badge variant="outline">{scope}</Badge>
  }
}

export function VolumeList({ instanceId }: VolumeListProps) {
  const [searchTerm, setSearchTerm] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<{
    name: string
  } | null>(null)
  const [showPruneDialog, setShowPruneDialog] = useState(false)
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [newVolumeName, setNewVolumeName] = useState('')
  const [newVolumeDriver, setNewVolumeDriver] = useState('local')

  const { data, isLoading, error, refetch } = useVolumes(instanceId)
  const deleteMutation = useDeleteVolume()
  const pruneMutation = usePruneVolumes()
  const createMutation = useCreateVolume()

  const volumes = data?.data?.volumes || []
  const filteredVolumes = volumes.filter((volume) => {
    const name = volume.name.toLowerCase()
    const driver = volume.driver.toLowerCase()
    const search = searchTerm.toLowerCase()
    return name.includes(search) || driver.includes(search)
  })

  const handleDelete = async () => {
    if (!deleteTarget) return

    try {
      await deleteMutation.mutateAsync({
        instanceId,
        name: deleteTarget.name,
        force: true,
      })
      toast.success(`卷 ${deleteTarget.name} 已删除`)
      setDeleteTarget(null)
      refetch()
    } catch (error: any) {
      toast.error(error?.message || '无法删除卷')
    }
  }

  const handlePrune = async () => {
    try {
      await pruneMutation.mutateAsync({
        instanceId,
      })
      toast.success('未使用的卷已清理')
      setShowPruneDialog(false)
      refetch()
    } catch (error: any) {
      toast.error(error?.message || '无法清理卷')
    }
  }

  const handleCreate = async () => {
    if (!newVolumeName.trim()) {
      toast.error('请输入卷名称')
      return
    }

    try {
      await createMutation.mutateAsync({
        instanceId,
        data: {
          name: newVolumeName.trim(),
          driver: newVolumeDriver,
        },
      })
      toast.success(`卷 ${newVolumeName} 已创建`)
      setShowCreateDialog(false)
      setNewVolumeName('')
      setNewVolumeDriver('local')
      refetch()
    } catch (error: any) {
      toast.error(error?.message || '无法创建卷')
    }
  }

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>卷列表</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            {[1, 2, 3].map((i) => (
              <Skeleton key={i} className="h-16" />
            ))}
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
            {(error as any)?.message || '无法加载卷列表'}
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
              <CardTitle>卷列表</CardTitle>
              <CardDescription className="mt-1">
                共 {volumes.length} 个卷
                {filteredVolumes.length !== volumes.length &&
                  ` (显示 ${filteredVolumes.length} 个)`}
              </CardDescription>
            </div>
            <div className="flex gap-2">
              <div className="relative">
                <IconSearch className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                <Input
                  placeholder="搜索卷名称或驱动..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-9 w-64"
                />
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowPruneDialog(true)}
                disabled={volumes.length === 0}
              >
                <IconAlertTriangle className="w-4 h-4 mr-2" />
                清理未使用
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowCreateDialog(true)}
              >
                <IconPlus className="w-4 h-4 mr-2" />
                创建卷
              </Button>
              <Button variant="outline" size="sm" onClick={() => refetch()}>
                <IconRefresh className="w-4 h-4" />
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {filteredVolumes.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              {searchTerm ? '未找到匹配的卷' : '暂无卷'}
            </div>
          ) : (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>名称</TableHead>
                    <TableHead>驱动</TableHead>
                    <TableHead>挂载点</TableHead>
                    <TableHead>范围</TableHead>
                    <TableHead>大小</TableHead>
                    <TableHead>引用计数</TableHead>
                    <TableHead className="text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredVolumes.map((volume) => (
                    <TableRow key={volume.name}>
                      <TableCell className="font-mono">
                        {volume.name}
                      </TableCell>
                      <TableCell>{volume.driver}</TableCell>
                      <TableCell className="max-w-xs truncate font-mono text-xs">
                        {volume.mountpoint}
                      </TableCell>
                      <TableCell>{getScopeBadge(volume.scope)}</TableCell>
                      <TableCell>
                        {formatBytes(volume.usage_data?.size)}
                      </TableCell>
                      <TableCell>
                        {volume.usage_data?.ref_count ?? '-'}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex justify-end gap-2">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() =>
                              setDeleteTarget({ name: volume.name })
                            }
                            disabled={
                              deleteMutation.isPending ||
                              (volume.usage_data?.ref_count ?? 0) > 0
                            }
                            title={
                              (volume.usage_data?.ref_count ?? 0) > 0
                                ? '卷正在使用中，无法删除'
                                : '删除卷'
                            }
                          >
                            <IconTrash className="w-4 h-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Delete Confirmation Dialog */}
      <AlertDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除卷</AlertDialogTitle>
            <AlertDialogDescription>
              确定要删除卷 <strong>{deleteTarget?.name}</strong> 吗？此操作无法撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleteMutation.isPending}>
              取消
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              disabled={deleteMutation.isPending}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {deleteMutation.isPending ? '删除中...' : '删除'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Prune Confirmation Dialog */}
      <AlertDialog
        open={showPruneDialog}
        onOpenChange={setShowPruneDialog}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认清理未使用的卷</AlertDialogTitle>
            <AlertDialogDescription>
              此操作将删除所有未被任何容器使用的卷。此操作无法撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={pruneMutation.isPending}>
              取消
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handlePrune}
              disabled={pruneMutation.isPending}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {pruneMutation.isPending ? '清理中...' : '清理'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Create Volume Dialog */}
      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>创建卷</DialogTitle>
            <DialogDescription>
              创建一个新的 Docker 卷用于持久化数据存储
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="volume-name">卷名称</Label>
              <Input
                id="volume-name"
                placeholder="my-volume"
                value={newVolumeName}
                onChange={(e) => setNewVolumeName(e.target.value)}
                disabled={createMutation.isPending}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="volume-driver">驱动</Label>
              <Input
                id="volume-driver"
                value={newVolumeDriver}
                onChange={(e) => setNewVolumeDriver(e.target.value)}
                disabled={createMutation.isPending}
              />
              <p className="text-xs text-muted-foreground">
                默认为 local，通常不需要修改
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setShowCreateDialog(false)
                setNewVolumeName('')
                setNewVolumeDriver('local')
              }}
              disabled={createMutation.isPending}
            >
              取消
            </Button>
            <Button
              onClick={handleCreate}
              disabled={createMutation.isPending || !newVolumeName.trim()}
            >
              {createMutation.isPending ? '创建中...' : '创建'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
