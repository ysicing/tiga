import { useState, useRef, useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { IconDownload, IconCheck, IconX, IconAlertCircle } from '@tabler/icons-react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
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
import { Textarea } from '@/components/ui/textarea'
import { Progress } from '@/components/ui/progress'
import { Badge } from '@/components/ui/badge'

interface ImagePullDialogProps {
  instanceId: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

interface PullProgress {
  id?: string
  status: string
  progress?: string
  progressDetail?: {
    current?: number
    total?: number
  }
}

type PullState = 'idle' | 'pulling' | 'success' | 'error'

export function ImagePullDialog({
  instanceId,
  open,
  onOpenChange,
}: ImagePullDialogProps) {
  const queryClient = useQueryClient()
  const [imageName, setImageName] = useState('')
  const [registryAuth, setRegistryAuth] = useState('')
  const [pullState, setPullState] = useState<PullState>('idle')
  const [progressLogs, setProgressLogs] = useState<PullProgress[]>([])
  const [overallProgress, setOverallProgress] = useState(0)
  const [errorMessage, setErrorMessage] = useState('')

  const eventSourceRef = useRef<EventSource | null>(null)
  const logsEndRef = useRef<HTMLDivElement>(null)

  // Auto-scroll to bottom when new logs arrive
  useEffect(() => {
    if (logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [progressLogs])

  // Cleanup on unmount or dialog close
  useEffect(() => {
    if (!open) {
      stopPull()
      // Reset state when dialog closes
      setTimeout(() => {
        if (!open) {
          setImageName('')
          setRegistryAuth('')
          setPullState('idle')
          setProgressLogs([])
          setOverallProgress(0)
          setErrorMessage('')
        }
      }, 300) // Delay to allow close animation
    }
  }, [open])

  const startPull = async () => {
    if (!imageName.trim()) {
      toast.error('请输入镜像名称')
      return
    }

    setPullState('pulling')
    setProgressLogs([])
    setOverallProgress(0)
    setErrorMessage('')

    try {
      // Make POST request to initiate pull and get SSE stream URL
      const response = await fetch(
        `/api/v1/docker/instances/${instanceId}/images/pull`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          credentials: 'include', // Use cookie-based auth like apiClient
          body: JSON.stringify({
            image: imageName.trim(),
            registry_auth: registryAuth.trim() || undefined,
          }),
        }
      )

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.message || '启动拉取失败')
      }

      // Response is SSE stream
      const reader = response.body?.getReader()
      if (!reader) {
        throw new Error('无法读取响应流')
      }

      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()

        if (done) {
          // Stream ended successfully
          setPullState('success')
          toast.success(`镜像 ${imageName} 拉取成功`)
          // Invalidate images cache
          queryClient.invalidateQueries({
            queryKey: ['docker', 'instance', instanceId, 'images'],
          })
          break
        }

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || '' // Keep incomplete line in buffer

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const data = JSON.parse(line.slice(6))

              // Handle completion event
              if (data.status === 'completed') {
                setPullState('success')
                toast.success(`镜像 ${imageName} 拉取成功`)
                // Invalidate all images queries for this instance (prefix match)
                queryClient.invalidateQueries({
                  queryKey: ['docker', 'instance', instanceId, 'images'],
                  exact: false, // Match all queries with this prefix
                })
                return // Exit the read loop
              }

              // Handle different event types
              if (data.error) {
                setPullState('error')
                setErrorMessage(data.error)
                toast.error(`拉取失败: ${data.error}`)
                break
              }

              // Add to progress logs
              setProgressLogs((prev) => [...prev, data])

              // Calculate overall progress
              if (data.progressDetail?.current && data.progressDetail?.total) {
                const percent = (data.progressDetail.current / data.progressDetail.total) * 100
                setOverallProgress(percent)
              }
            } catch (error) {
              console.error('Failed to parse SSE data:', error)
            }
          }
        }
      }
    } catch (error: any) {
      console.error('Pull error:', error)
      setPullState('error')
      setErrorMessage(error.message || '拉取镜像时发生错误')
      toast.error(error.message || '拉取失败')
    }
  }

  const stopPull = () => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
    }
  }

  const handleClose = () => {
    if (pullState === 'pulling') {
      if (!confirm('拉取正在进行中，确定要关闭吗？')) {
        return
      }
      stopPull()
    }
    onOpenChange(false)
  }

  const canStartPull = imageName.trim() !== '' && pullState === 'idle'

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <IconDownload className="w-5 h-5" />
            拉取 Docker 镜像
          </DialogTitle>
          <DialogDescription>
            从 Docker Hub 或其他镜像仓库拉取镜像
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* Image Name Input */}
          <div className="space-y-2">
            <Label htmlFor="image-name">镜像名称 *</Label>
            <Input
              id="image-name"
              placeholder="例如: nginx:latest 或 redis:7-alpine"
              value={imageName}
              onChange={(e) => setImageName(e.target.value)}
              disabled={pullState === 'pulling'}
            />
            <p className="text-xs text-muted-foreground">
              格式: [仓库地址/]镜像名称[:标签]
            </p>
          </div>

          {/* Registry Auth (Optional) */}
          <div className="space-y-2">
            <Label htmlFor="registry-auth">
              镜像仓库认证 (可选)
            </Label>
            <Textarea
              id="registry-auth"
              placeholder="Base64 编码的认证信息 (格式: username:password)"
              value={registryAuth}
              onChange={(e) => setRegistryAuth(e.target.value)}
              disabled={pullState === 'pulling'}
              rows={3}
              className="font-mono text-xs"
            />
            <p className="text-xs text-muted-foreground">
              私有镜像仓库需要提供认证信息
            </p>
          </div>

          {/* Pull State Badge */}
          {pullState !== 'idle' && (
            <div className="flex items-center gap-2">
              {pullState === 'pulling' && (
                <Badge className="bg-blue-500">
                  <IconDownload className="w-3 h-3 mr-1 animate-pulse" />
                  拉取中...
                </Badge>
              )}
              {pullState === 'success' && (
                <Badge className="bg-green-500">
                  <IconCheck className="w-3 h-3 mr-1" />
                  拉取成功
                </Badge>
              )}
              {pullState === 'error' && (
                <Badge variant="destructive">
                  <IconX className="w-3 h-3 mr-1" />
                  拉取失败
                </Badge>
              )}
            </div>
          )}

          {/* Overall Progress Bar */}
          {pullState === 'pulling' && overallProgress > 0 && (
            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">总体进度</span>
                <span className="font-medium">{overallProgress.toFixed(0)}%</span>
              </div>
              <Progress value={overallProgress} />
            </div>
          )}

          {/* Error Message */}
          {pullState === 'error' && errorMessage && (
            <div className="p-3 border border-destructive rounded-md bg-destructive/10 text-destructive flex items-start gap-2">
              <IconAlertCircle className="w-5 h-5 flex-shrink-0 mt-0.5" />
              <div className="flex-1">
                <p className="font-medium">拉取失败</p>
                <p className="text-sm mt-1">{errorMessage}</p>
              </div>
            </div>
          )}

          {/* Progress Logs */}
          {progressLogs.length > 0 && (
            <div className="border rounded-md bg-black text-white">
              <div className="p-3 border-b border-gray-700">
                <p className="text-xs font-medium text-gray-400">
                  拉取日志 ({progressLogs.length} 条)
                </p>
              </div>
              <div
                className="p-3 font-mono text-xs overflow-auto max-h-[300px]"
                style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}
              >
                {progressLogs.map((log, index) => (
                  <div key={index} className="py-0.5 text-gray-200">
                    {log.id && (
                      <span className="text-blue-400">[{log.id.substring(0, 12)}]</span>
                    )}{' '}
                    <span className="text-green-400">{log.status}</span>
                    {log.progress && (
                      <span className="text-yellow-400 ml-2">{log.progress}</span>
                    )}
                  </div>
                ))}
                <div ref={logsEndRef} />
              </div>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={handleClose}
            disabled={pullState === 'pulling'}
          >
            {pullState === 'success' ? '关闭' : '取消'}
          </Button>
          {pullState === 'idle' && (
            <Button onClick={startPull} disabled={!canStartPull}>
              <IconDownload className="w-4 h-4 mr-2" />
              开始拉取
            </Button>
          )}
          {pullState === 'pulling' && (
            <Button variant="destructive" onClick={stopPull}>
              停止拉取
            </Button>
          )}
          {(pullState === 'success' || pullState === 'error') && (
            <Button
              onClick={() => {
                setPullState('idle')
                setProgressLogs([])
                setOverallProgress(0)
                setErrorMessage('')
              }}
            >
              重新拉取
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
