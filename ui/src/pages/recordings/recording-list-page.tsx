import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import {
  IconClock,
  IconDatabase,
  IconDownload,
  IconFileText,
  IconPlayerPlay,
  IconSearch,
  IconTerminal,
  IconTrash,
  IconX,
} from '@tabler/icons-react'

import { recordingService, RecordingMetadata, RecordingFilters } from '@/services/recording-service'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
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
import { Skeleton } from '@/components/ui/skeleton'

const getRecordingTypeIcon = (type: string) => {
  switch (type) {
    case 'docker':
      return <IconDatabase className="w-4 h-4" />
    case 'webssh':
      return <IconTerminal className="w-4 h-4" />
    case 'k8s_node':
    case 'k8s_pod':
      return <IconFileText className="w-4 h-4" />
    default:
      return <IconFileText className="w-4 h-4" />
  }
}

const getRecordingTypeBadge = (type: string) => {
  const config = {
    docker: { label: 'Docker', className: 'bg-blue-500' },
    webssh: { label: 'WebSSH', className: 'bg-purple-500' },
    k8s_node: { label: 'K8s 节点', className: 'bg-green-500' },
    k8s_pod: { label: 'K8s Pod', className: 'bg-teal-500' },
  }[type] || { label: type, className: 'bg-gray-500' }

  return (
    <Badge className={config.className}>
      {getRecordingTypeIcon(type)}
      <span className="ml-1">{config.label}</span>
    </Badge>
  )
}

const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
}

const formatDuration = (seconds: number): string => {
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = seconds % 60

  if (hours > 0) {
    return `${hours}h ${minutes}m ${secs}s`
  }
  if (minutes > 0) {
    return `${minutes}m ${secs}s`
  }
  return `${secs}s`
}

const formatDate = (dateStr: string): string => {
  const date = new Date(dateStr)
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

export function RecordingListPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const recordingsPath = '/system/recordings'

  const [filters, setFilters] = useState<RecordingFilters>({
    page: 1,
    limit: 20,
  })
  const [searchQuery, setSearchQuery] = useState('')
  const [deleteId, setDeleteId] = useState<string | null>(null)

  // Fetch recordings
  const { data, isLoading, error } = useQuery({
    queryKey: ['recordings', filters],
    queryFn: () => recordingService.listRecordings(filters),
  })

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => recordingService.deleteRecording(id),
    onSuccess: () => {
      toast.success('录制已删除')
      queryClient.invalidateQueries({ queryKey: ['recordings'] })
      setDeleteId(null)
    },
    onError: (error: any) => {
      toast.error(error?.message || '删除失败')
    },
  })

  // Download recording
  const handleDownload = async (recording: RecordingMetadata) => {
    try {
      await recordingService.downloadRecording(recording.id)
      toast.success('录制下载已开始')
    } catch (error: any) {
      toast.error(error?.message || '下载失败')
    }
  }

  // Search recordings
  const handleSearch = () => {
    if (searchQuery.trim()) {
      navigate(`${recordingsPath}/search?q=${encodeURIComponent(searchQuery)}`)
    }
  }

  const recordings = data?.items || []
  const totalPages = data?.total_pages || 0

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-3xl font-bold">终端录制</h1>
        </div>
        <Card>
          <CardContent className="p-6">
            <div className="space-y-4">
              {[1, 2, 3, 4, 5].map((i) => (
                <Skeleton key={i} className="h-16" />
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (error) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-3xl font-bold">终端录制</h1>
        </div>
        <Card>
          <CardHeader>
            <CardTitle className="text-destructive">加载失败</CardTitle>
            <CardDescription>
              {(error as any)?.message || '无法加载录制列表'}
            </CardDescription>
          </CardHeader>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">终端录制</h1>
        <Button onClick={() => navigate(`${recordingsPath}/statistics`)}>
          <IconFileText className="w-4 h-4 mr-2" />
          查看统计
        </Button>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="p-6">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            {/* Search */}
            <div className="md:col-span-2">
              <div className="flex gap-2">
                <Input
                  placeholder="搜索录制（用户名、描述、标签）"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
                />
                <Button onClick={handleSearch}>
                  <IconSearch className="w-4 h-4" />
                </Button>
              </div>
            </div>

            {/* Type Filter */}
            <Select
              value={filters.recording_type || 'all'}
              onValueChange={(value) =>
                setFilters({
                  ...filters,
                  recording_type: value === 'all' ? undefined : value as any,
                  page: 1,
                })
              }
            >
              <SelectTrigger>
                <SelectValue placeholder="类型" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">所有类型</SelectItem>
                <SelectItem value="docker">Docker</SelectItem>
                <SelectItem value="webssh">WebSSH</SelectItem>
                <SelectItem value="k8s_node">K8s 节点</SelectItem>
                <SelectItem value="k8s_pod">K8s Pod</SelectItem>
              </SelectContent>
            </Select>

            {/* Sort */}
            <Select
              value={filters.sort_by || 'started_at'}
              onValueChange={(value) =>
                setFilters({ ...filters, sort_by: value as any, page: 1 })
              }
            >
              <SelectTrigger>
                <SelectValue placeholder="排序" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="started_at">开始时间</SelectItem>
                <SelectItem value="ended_at">结束时间</SelectItem>
                <SelectItem value="file_size">文件大小</SelectItem>
                <SelectItem value="duration">时长</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      {/* Recordings Table */}
      <Card>
        <CardHeader>
          <CardTitle>录制列表</CardTitle>
          <CardDescription>
            共 {data?.total || 0} 条录制，第 {filters.page || 1} / {totalPages} 页
          </CardDescription>
        </CardHeader>
        <CardContent>
          {recordings.length === 0 ? (
            <div className="text-center py-12 text-muted-foreground">
              <IconX className="w-12 h-12 mx-auto mb-4" />
              <p>暂无录制数据</p>
            </div>
          ) : (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>类型</TableHead>
                    <TableHead>用户</TableHead>
                    <TableHead>开始时间</TableHead>
                    <TableHead>时长</TableHead>
                    <TableHead>大小</TableHead>
                    <TableHead>终端</TableHead>
                    <TableHead className="text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {recordings.map((recording) => (
                    <TableRow key={recording.id}>
                      <TableCell>
                        {getRecordingTypeBadge(recording.recording_type)}
                      </TableCell>
                      <TableCell className="font-medium">
                        {recording.username}
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center">
                          <IconClock className="w-4 h-4 mr-2 text-muted-foreground" />
                          {formatDate(recording.started_at)}
                        </div>
                      </TableCell>
                      <TableCell>{formatDuration(recording.duration)}</TableCell>
                      <TableCell>{formatFileSize(recording.file_size)}</TableCell>
                      <TableCell className="text-muted-foreground">
                        {recording.rows}×{recording.cols}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-2">
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => navigate(`${recordingsPath}/${recording.id}`)}
                          >
                            <IconFileText className="w-4 h-4" />
                          </Button>
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => navigate(`${recordingsPath}/${recording.id}/player`)}
                          >
                            <IconPlayerPlay className="w-4 h-4" />
                          </Button>
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => handleDownload(recording)}
                          >
                            <IconDownload className="w-4 h-4" />
                          </Button>
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => setDeleteId(recording.id)}
                          >
                            <IconTrash className="w-4 h-4 text-destructive" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between mt-4">
              <Button
                variant="outline"
                disabled={filters.page === 1}
                onClick={() => setFilters({ ...filters, page: (filters.page || 1) - 1 })}
              >
                上一页
              </Button>
              <span className="text-sm text-muted-foreground">
                第 {filters.page || 1} / {totalPages} 页
              </span>
              <Button
                variant="outline"
                disabled={filters.page === totalPages}
                onClick={() => setFilters({ ...filters, page: (filters.page || 1) + 1 })}
              >
                下一页
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={!!deleteId} onOpenChange={() => setDeleteId(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除</AlertDialogTitle>
            <AlertDialogDescription>
              此操作将永久删除该录制及其文件，无法恢复。是否继续？
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteId && deleteMutation.mutate(deleteId)}
              className="bg-destructive text-destructive-foreground"
            >
              删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
