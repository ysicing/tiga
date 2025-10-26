import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import {
  IconArrowLeft,
  IconClock,
  IconDatabase,
  IconDownload,
  IconFileText,
  IconPlayerPlay,
  IconTerminal,
  IconTrash,
  IconUser,
} from '@tabler/icons-react'

import { recordingService, RecordingMetadata } from '@/services/recording-service'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
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
import { Separator } from '@/components/ui/separator'

const getRecordingTypeIcon = (type: string) => {
  switch (type) {
    case 'docker':
      return <IconDatabase className="w-5 h-5" />
    case 'webssh':
      return <IconTerminal className="w-5 h-5" />
    case 'k8s_node':
    case 'k8s_pod':
      return <IconFileText className="w-5 h-5" />
    default:
      return <IconFileText className="w-5 h-5" />
  }
}

const getRecordingTypeLabel = (type: string): string => {
  const labels: Record<string, string> = {
    docker: 'Docker 容器',
    webssh: 'WebSSH 主机',
    k8s_node: 'Kubernetes 节点',
    k8s_pod: 'Kubernetes Pod',
  }
  return labels[type] || type
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

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
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
    second: '2-digit',
  }).format(date)
}

const renderTypeMetadata = (type: string, metadata: Record<string, any>) => {
  if (!metadata || Object.keys(metadata).length === 0) {
    return <p className="text-muted-foreground">无类型特定元数据</p>
  }

  switch (type) {
    case 'docker':
      return (
        <dl className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {metadata.container_id && (
            <>
              <dt className="font-medium">容器 ID</dt>
              <dd className="text-muted-foreground font-mono text-sm">
                {metadata.container_id}
              </dd>
            </>
          )}
          {metadata.container_name && (
            <>
              <dt className="font-medium">容器名称</dt>
              <dd className="text-muted-foreground">{metadata.container_name}</dd>
            </>
          )}
          {metadata.image && (
            <>
              <dt className="font-medium">镜像</dt>
              <dd className="text-muted-foreground font-mono text-sm">
                {metadata.image}
              </dd>
            </>
          )}
          {metadata.instance_id && (
            <>
              <dt className="font-medium">实例 ID</dt>
              <dd className="text-muted-foreground font-mono text-sm">
                {metadata.instance_id}
              </dd>
            </>
          )}
        </dl>
      )

    case 'webssh':
      return (
        <dl className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {metadata.host && (
            <>
              <dt className="font-medium">主机名称</dt>
              <dd className="text-muted-foreground">{metadata.host}</dd>
            </>
          )}
          {metadata.host_id && (
            <>
              <dt className="font-medium">节点 ID</dt>
              <dd className="text-muted-foreground font-mono text-sm">{metadata.host_id}</dd>
            </>
          )}
        </dl>
      )

    case 'k8s_node':
      return (
        <dl className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {metadata.cluster_id && (
            <>
              <dt className="font-medium">集群 ID</dt>
              <dd className="text-muted-foreground font-mono text-sm">
                {metadata.cluster_id}
              </dd>
            </>
          )}
          {metadata.node_name && (
            <>
              <dt className="font-medium">节点名称</dt>
              <dd className="text-muted-foreground">{metadata.node_name}</dd>
            </>
          )}
          {metadata.node_ip && (
            <>
              <dt className="font-medium">节点 IP</dt>
              <dd className="text-muted-foreground">{metadata.node_ip}</dd>
            </>
          )}
        </dl>
      )

    case 'k8s_pod':
      return (
        <dl className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {metadata.cluster_id && (
            <>
              <dt className="font-medium">集群 ID</dt>
              <dd className="text-muted-foreground font-mono text-sm">
                {metadata.cluster_id}
              </dd>
            </>
          )}
          {metadata.namespace && (
            <>
              <dt className="font-medium">命名空间</dt>
              <dd className="text-muted-foreground">{metadata.namespace}</dd>
            </>
          )}
          {metadata.pod_name && (
            <>
              <dt className="font-medium">Pod 名称</dt>
              <dd className="text-muted-foreground">{metadata.pod_name}</dd>
            </>
          )}
          {metadata.container_name && (
            <>
              <dt className="font-medium">容器名称</dt>
              <dd className="text-muted-foreground">{metadata.container_name}</dd>
            </>
          )}
        </dl>
      )

    default:
      return (
        <pre className="bg-muted p-4 rounded-md overflow-auto">
          {JSON.stringify(metadata, null, 2)}
        </pre>
      )
  }
}

export function RecordingDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)

  const recordingsPath = '/system/recordings'

  // Fetch recording details
  const { data: recording, isLoading, error } = useQuery({
    queryKey: ['recording', id],
    queryFn: () => recordingService.getRecording(id!),
    enabled: !!id,
  })

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: () => recordingService.deleteRecording(id!),
    onSuccess: () => {
      toast.success('录制已删除')
      queryClient.invalidateQueries({ queryKey: ['recordings'] })
      navigate(recordingsPath)
    },
    onError: (error: any) => {
      toast.error(error?.message || '删除失败')
    },
  })

  // Download recording
  const handleDownload = async () => {
    if (!recording) return
    try {
      await recordingService.downloadRecording(recording.id)
      toast.success('录制下载已开始')
    } catch (error: any) {
      toast.error(error?.message || '下载失败')
    }
  }

  // Play recording
  const handlePlay = () => {
    if (!recording) return
    navigate(`${recordingsPath}/${recording.id}/player`)
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate(recordingsPath)}>
            <IconArrowLeft className="w-5 h-5" />
          </Button>
          <Skeleton className="h-8 w-48" />
        </div>
        <Card>
          <CardContent className="p-6 space-y-4">
            {[1, 2, 3, 4, 5].map((i) => (
              <Skeleton key={i} className="h-16" />
            ))}
          </CardContent>
        </Card>
      </div>
    )
  }

  if (error || !recording) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate(recordingsPath)}>
            <IconArrowLeft className="w-5 h-5" />
          </Button>
          <h1 className="text-3xl font-bold">录制详情</h1>
        </div>
        <Card>
          <CardHeader>
            <CardTitle className="text-destructive">加载失败</CardTitle>
            <CardDescription>
              {(error as any)?.message || '无法加载录制详情'}
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
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate(recordingsPath)}>
            <IconArrowLeft className="w-5 h-5" />
          </Button>
          <h1 className="text-3xl font-bold">录制详情</h1>
        </div>

        <div className="flex items-center gap-2">
          <Button onClick={handlePlay} variant="default">
            <IconPlayerPlay className="w-4 h-4 mr-2" />
            播放
          </Button>
          <Button onClick={handleDownload} variant="outline">
            <IconDownload className="w-4 h-4 mr-2" />
            下载
          </Button>
          <Button
            onClick={() => setDeleteDialogOpen(true)}
            variant="destructive"
          >
            <IconTrash className="w-4 h-4 mr-2" />
            删除
          </Button>
        </div>
      </div>

      {/* Basic Information */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>基本信息</CardTitle>
            {getRecordingTypeBadge(recording.recording_type)}
          </div>
          <CardDescription>
            {getRecordingTypeLabel(recording.recording_type)}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <dl className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <dt className="text-sm font-medium text-muted-foreground mb-1">
                录制 ID
              </dt>
              <dd className="font-mono text-sm">{recording.id}</dd>
            </div>

            <div>
              <dt className="text-sm font-medium text-muted-foreground mb-1">
                用户
              </dt>
              <dd className="flex items-center">
                <IconUser className="w-4 h-4 mr-2 text-muted-foreground" />
                {recording.username}
              </dd>
            </div>

            <div>
              <dt className="text-sm font-medium text-muted-foreground mb-1">
                开始时间
              </dt>
              <dd className="flex items-center">
                <IconClock className="w-4 h-4 mr-2 text-muted-foreground" />
                {formatDate(recording.started_at)}
              </dd>
            </div>

            {recording.ended_at && (
              <div>
                <dt className="text-sm font-medium text-muted-foreground mb-1">
                  结束时间
                </dt>
                <dd className="flex items-center">
                  <IconClock className="w-4 h-4 mr-2 text-muted-foreground" />
                  {formatDate(recording.ended_at)}
                </dd>
              </div>
            )}

            <div>
              <dt className="text-sm font-medium text-muted-foreground mb-1">
                录制时长
              </dt>
              <dd>{formatDuration(recording.duration)}</dd>
            </div>

            <div>
              <dt className="text-sm font-medium text-muted-foreground mb-1">
                文件大小
              </dt>
              <dd>{formatFileSize(recording.file_size)}</dd>
            </div>

            <div>
              <dt className="text-sm font-medium text-muted-foreground mb-1">
                终端尺寸
              </dt>
              <dd>
                {recording.rows} × {recording.cols}
              </dd>
            </div>

            <div>
              <dt className="text-sm font-medium text-muted-foreground mb-1">
                Shell
              </dt>
              <dd className="font-mono text-sm">{recording.shell}</dd>
            </div>

            <div>
              <dt className="text-sm font-medium text-muted-foreground mb-1">
                客户端 IP
              </dt>
              <dd className="font-mono text-sm">{recording.client_ip}</dd>
            </div>

            <div>
              <dt className="text-sm font-medium text-muted-foreground mb-1">
                会话 ID
              </dt>
              <dd className="font-mono text-sm">{recording.session_id}</dd>
            </div>
          </dl>

          {recording.description && (
            <>
              <Separator />
              <div>
                <dt className="text-sm font-medium text-muted-foreground mb-2">
                  描述
                </dt>
                <dd className="text-sm">{recording.description}</dd>
              </div>
            </>
          )}

          {recording.tags && (
            <>
              <Separator />
              <div>
                <dt className="text-sm font-medium text-muted-foreground mb-2">
                  标签
                </dt>
                <dd className="flex flex-wrap gap-2">
                  {recording.tags.split(',').map((tag, index) => (
                    <Badge key={index} variant="secondary">
                      {tag.trim()}
                    </Badge>
                  ))}
                </dd>
              </div>
            </>
          )}
        </CardContent>
      </Card>

      {/* Type-Specific Metadata */}
      <Card>
        <CardHeader>
          <CardTitle>类型特定信息</CardTitle>
          <CardDescription>
            {getRecordingTypeLabel(recording.recording_type)} 相关元数据
          </CardDescription>
        </CardHeader>
        <CardContent>
          {renderTypeMetadata(recording.recording_type, recording.type_metadata)}
        </CardContent>
      </Card>

      {/* Storage Information */}
      <Card>
        <CardHeader>
          <CardTitle>存储信息</CardTitle>
          <CardDescription>录制文件存储相关信息</CardDescription>
        </CardHeader>
        <CardContent>
          <dl className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <dt className="text-sm font-medium text-muted-foreground mb-1">
                存储类型
              </dt>
              <dd>
                <Badge variant={recording.storage_type === 'local' ? 'secondary' : 'default'}>
                  {recording.storage_type === 'local' ? '本地存储' : 'MinIO'}
                </Badge>
              </dd>
            </div>

            <div>
              <dt className="text-sm font-medium text-muted-foreground mb-1">
                文件格式
              </dt>
              <dd>
                <Badge variant="outline">
                  {recording.format}
                </Badge>
              </dd>
            </div>

            <div>
              <dt className="text-sm font-medium text-muted-foreground mb-1">
                文件大小
              </dt>
              <dd>{formatFileSize(recording.file_size)}</dd>
            </div>

            {recording.storage_path && (
              <div className="md:col-span-2">
                <dt className="text-sm font-medium text-muted-foreground mb-1">
                  存储路径
                </dt>
                <dd className="font-mono text-sm p-3 bg-muted rounded-md break-all">
                  {recording.storage_path}
                </dd>
              </div>
            )}
          </dl>
        </CardContent>
      </Card>

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
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
              onClick={() => deleteMutation.mutate()}
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
