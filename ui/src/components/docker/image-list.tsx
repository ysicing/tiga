import { useState } from 'react'
import {
  useImages,
  useDeleteImage,
} from '@/services/docker-api'
import {
  IconTrash,
  IconTag,
  IconDownload,
  IconRefresh,
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
import { ImagePullDialog } from '@/components/docker/image-pull-dialog'

interface ImageListProps {
  instanceId: string
}

const formatSize = (bytes: number) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}

const formatTags = (tags: string[] | undefined) => {
  if (!tags || tags.length === 0) return '<none>'
  return tags.join(', ')
}

const formatCreated = (timestamp: number) => {
  const date = new Date(timestamp * 1000)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))

  if (diffDays === 0) return '今天'
  if (diffDays === 1) return '昨天'
  if (diffDays < 7) return `${diffDays} 天前`
  if (diffDays < 30) return `${Math.floor(diffDays / 7)} 周前`
  if (diffDays < 365) return `${Math.floor(diffDays / 30)} 个月前`
  return `${Math.floor(diffDays / 365)} 年前`
}

export function ImageList({ instanceId }: ImageListProps) {
  const [searchTerm, setSearchTerm] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<{
    id: string
    tags: string
  } | null>(null)
  const [showPullDialog, setShowPullDialog] = useState(false)

  const { data, isLoading, error, refetch } = useImages(instanceId, { all: true })
  const deleteMutation = useDeleteImage()

  const images = data?.data?.images || []
  const filteredImages = images.filter((image) => {
    const tags = formatTags(image.repo_tags).toLowerCase()
    const id = image.id.toLowerCase()
    const search = searchTerm.toLowerCase()
    return tags.includes(search) || id.includes(search)
  })

  const handleDelete = async () => {
    if (!deleteTarget) return

    try {
      await deleteMutation.mutateAsync({
        instanceId,
        imageId: deleteTarget.id,
        force: false,
        noPrune: false,
      })
      toast.success(`镜像已删除`)
      setDeleteTarget(null)
    } catch (error: any) {
      toast.error(error?.message || '无法删除镜像')
    }
  }

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>镜像列表</CardTitle>
              <CardDescription className="mt-1 flex items-center gap-2">
                <div className="h-4 w-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />
                正在加载镜像列表...
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[300px]">仓库标签</TableHead>
                  <TableHead>镜像 ID</TableHead>
                  <TableHead>大小</TableHead>
                  <TableHead>创建时间</TableHead>
                  <TableHead className="text-right w-[120px]">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {[1, 2, 3, 4, 5].map((i) => (
                  <TableRow key={i}>
                    <TableCell>
                      <Skeleton className="h-4 w-48" />
                    </TableCell>
                    <TableCell>
                      <Skeleton className="h-4 w-24" />
                    </TableCell>
                    <TableCell>
                      <Skeleton className="h-4 w-16" />
                    </TableCell>
                    <TableCell>
                      <Skeleton className="h-4 w-28" />
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end gap-1">
                        <Skeleton className="h-8 w-8 rounded-md" />
                        <Skeleton className="h-8 w-8 rounded-md" />
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
            {(error as any)?.message || '无法加载镜像列表'}
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
              <CardTitle>镜像列表</CardTitle>
              <CardDescription className="mt-1">
                共 {images.length} 个镜像
                {filteredImages.length !== images.length &&
                  ` (显示 ${filteredImages.length} 个)`}
              </CardDescription>
            </div>
            <div className="flex gap-2">
              <div className="relative">
                <IconSearch className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                <Input
                  placeholder="搜索镜像标签或 ID..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-9 w-64"
                />
              </div>
              <Button variant="outline" size="sm" onClick={() => setShowPullDialog(true)}>
                <IconDownload className="w-4 h-4 mr-2" />
                拉取镜像
              </Button>
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
          {filteredImages.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              {searchTerm ? '没有匹配的镜像' : '还没有镜像'}
            </div>
          ) : (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-[300px]">仓库标签</TableHead>
                    <TableHead>镜像 ID</TableHead>
                    <TableHead>大小</TableHead>
                    <TableHead>创建时间</TableHead>
                    <TableHead className="text-right w-[120px]">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredImages.map((image) => {
                    const tags = formatTags(image.repo_tags)
                    const imageId = image.id.replace('sha256:', '').substring(0, 12)

                    return (
                      <TableRow key={image.id}>
                        <TableCell className="font-medium">
                          <div className="flex flex-wrap gap-1">
                            {image.repo_tags && image.repo_tags.length > 0 ? (
                              image.repo_tags.map((tag, idx) => (
                                <Badge key={idx} variant="outline" className="font-mono text-xs">
                                  {tag}
                                </Badge>
                              ))
                            ) : (
                              <span className="text-muted-foreground text-sm">&lt;none&gt;</span>
                            )}
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="font-mono text-xs text-muted-foreground">
                            {imageId}
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="text-sm">{formatSize(image.size)}</div>
                        </TableCell>
                        <TableCell>
                          <div className="text-sm text-muted-foreground">
                            {formatCreated(image.created)}
                          </div>
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex justify-end gap-1">
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-8 w-8"
                              title="添加标签"
                            >
                              <IconTag className="w-4 h-4" />
                            </Button>
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-8 w-8"
                              onClick={() =>
                                setDeleteTarget({ id: image.id, tags })
                              }
                              title="删除镜像"
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
            <AlertDialogTitle>确认删除镜像</AlertDialogTitle>
            <AlertDialogDescription>
              此操作将删除镜像 <strong>{deleteTarget?.tags}</strong>。
              如果有容器正在使用此镜像，删除将失败。
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

      {/* Pull Image Dialog */}
      <ImagePullDialog
        instanceId={instanceId}
        open={showPullDialog}
        onOpenChange={setShowPullDialog}
      />
    </>
  )
}
