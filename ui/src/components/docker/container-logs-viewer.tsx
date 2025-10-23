import { useEffect, useRef, useState } from 'react'
import { useContainerLogs } from '@/services/docker-api'
import {
  IconCheck,
  IconPlayerPause,
  IconPlayerPlay,
  IconRefresh,
  IconTrash,
  IconX,
} from '@tabler/icons-react'
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
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'

interface ContainerLogsViewerProps {
  instanceId: string
  containerId: string
  containerName: string
}

interface LogEntry {
  stream: 'stdout' | 'stderr'
  content: string
  timestamp?: string
}

export function ContainerLogsViewer({
  instanceId,
  containerId,
  containerName,
}: ContainerLogsViewerProps) {
  const [streaming, setStreaming] = useState(false)
  const [tail, setTail] = useState('100')
  const [timestamps, setTimestamps] = useState(false)
  const [showStdout, setShowStdout] = useState(true)
  const [showStderr, setShowStderr] = useState(true)
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [autoScroll, setAutoScroll] = useState(true)

  const logsEndRef = useRef<HTMLDivElement>(null)
  const eventSourceRef = useRef<EventSource | null>(null)

  // Fetch historical logs
  const {
    data: historicalData,
    isLoading,
    error,
    refetch,
  } = useContainerLogs(instanceId, containerId, {
    tail: parseInt(tail, 10) || 100,
    timestamps,
    stdout: showStdout,
    stderr: showStderr,
  })

  // Update logs when historical data changes
  useEffect(() => {
    if (historicalData?.data && !streaming) {
      const entries = (historicalData.data as any[]).map((entry) => ({
        stream: entry.stream || 'stdout',
        content: entry.content || entry.message || String(entry),
        timestamp: entry.timestamp,
      }))
      setLogs(entries)
    }
  }, [historicalData, streaming])

  // Auto-scroll to bottom when new logs arrive
  useEffect(() => {
    if (autoScroll && logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [logs, autoScroll])

  // Start SSE streaming
  const startStreaming = () => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
    }

    const params = new URLSearchParams({
      tail,
      timestamps: timestamps.toString(),
      stdout: showStdout.toString(),
      stderr: showStderr.toString(),
    })

    const url = `/api/v1/docker/instances/${instanceId}/containers/${containerId}/logs/stream?${params.toString()}`
    const eventSource = new EventSource(url, {
      withCredentials: true,
    })

    eventSource.onopen = () => {
      console.log('SSE connection opened')
      setStreaming(true)
      setLogs([]) // Clear logs when starting streaming
    }

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        const entry: LogEntry = {
          stream: data.stream || 'stdout',
          content: data.content || data.message || String(data),
          timestamp: data.timestamp,
        }
        setLogs((prev) => [...prev, entry])
      } catch (error) {
        console.error('Failed to parse log entry:', error)
      }
    }

    eventSource.onerror = (error) => {
      console.error('SSE error:', error)
      toast.error('日志流连接中断')
      stopStreaming()
    }

    eventSourceRef.current = eventSource
  }

  // Stop SSE streaming
  const stopStreaming = () => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
    }
    setStreaming(false)
  }

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
      }
    }
  }, [])

  const handleClearLogs = () => {
    setLogs([])
    toast.success('日志已清空')
  }

  const handleRefresh = () => {
    if (streaming) {
      stopStreaming()
      setTimeout(() => startStreaming(), 100)
    } else {
      refetch()
    }
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>容器日志</CardTitle>
            <CardDescription>
              容器 <span className="font-mono">{containerName}</span> 的日志输出
            </CardDescription>
          </div>
          <div className="flex items-center gap-2">
            {streaming ? (
              <Badge className="bg-green-500">
                <IconCheck className="w-3 h-3 mr-1" />
                实时流
              </Badge>
            ) : (
              <Badge variant="secondary">
                <IconX className="w-3 h-3 mr-1" />
                历史记录
              </Badge>
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Controls */}
        <div className="flex flex-wrap items-center gap-4 p-4 border rounded-md bg-muted/50">
          <div className="flex items-center gap-2">
            <Label htmlFor="tail" className="text-sm">
              尾行数:
            </Label>
            <Input
              id="tail"
              type="number"
              value={tail}
              onChange={(e) => setTail(e.target.value)}
              className="w-24"
              disabled={streaming}
            />
          </div>

          <div className="flex items-center gap-2">
            <Checkbox
              id="timestamps"
              checked={timestamps}
              onCheckedChange={(checked) => setTimestamps(checked as boolean)}
              disabled={streaming}
            />
            <Label htmlFor="timestamps" className="text-sm">
              显示时间戳
            </Label>
          </div>

          <div className="flex items-center gap-2">
            <Checkbox
              id="stdout"
              checked={showStdout}
              onCheckedChange={(checked) => setShowStdout(checked as boolean)}
              disabled={streaming}
            />
            <Label htmlFor="stdout" className="text-sm">
              stdout
            </Label>
          </div>

          <div className="flex items-center gap-2">
            <Checkbox
              id="stderr"
              checked={showStderr}
              onCheckedChange={(checked) => setShowStderr(checked as boolean)}
              disabled={streaming}
            />
            <Label htmlFor="stderr" className="text-sm">
              stderr
            </Label>
          </div>

          <div className="flex items-center gap-2">
            <Checkbox
              id="autoscroll"
              checked={autoScroll}
              onCheckedChange={(checked) => setAutoScroll(checked as boolean)}
            />
            <Label htmlFor="autoscroll" className="text-sm">
              自动滚动
            </Label>
          </div>

          <div className="flex-1" />

          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={handleRefresh}
              disabled={isLoading}
            >
              <IconRefresh className="w-4 h-4" />
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={handleClearLogs}
              disabled={logs.length === 0}
            >
              <IconTrash className="w-4 h-4" />
            </Button>
            {streaming ? (
              <Button
                variant="destructive"
                size="sm"
                onClick={stopStreaming}
              >
                <IconPlayerPause className="w-4 h-4 mr-2" />
                停止流
              </Button>
            ) : (
              <Button
                variant="default"
                size="sm"
                onClick={startStreaming}
              >
                <IconPlayerPlay className="w-4 h-4 mr-2" />
                开始流
              </Button>
            )}
          </div>
        </div>

        {/* Logs Display */}
        {isLoading && !streaming ? (
          <div className="space-y-2">
            {[1, 2, 3, 4, 5].map((i) => (
              <Skeleton key={i} className="h-6" />
            ))}
          </div>
        ) : error && !streaming ? (
          <div className="p-4 text-center border rounded-md bg-destructive/10 text-destructive">
            <p className="font-medium">加载日志失败</p>
            <p className="text-sm mt-1">
              {(error as any)?.message || '无法加载容器日志'}
            </p>
          </div>
        ) : (
          <div className="border rounded-md bg-black text-white">
            <div
              className="p-4 font-mono text-xs overflow-auto max-h-[600px]"
              style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}
            >
              {logs.length === 0 ? (
                <div className="text-gray-500 text-center py-8">
                  暂无日志
                </div>
              ) : (
                logs.map((log, index) => (
                  <div
                    key={index}
                    className={`py-0.5 ${
                      log.stream === 'stderr' ? 'text-red-400' : 'text-gray-100'
                    }`}
                  >
                    {log.timestamp && (
                      <span className="text-gray-500 mr-2">
                        [{log.timestamp}]
                      </span>
                    )}
                    {log.content}
                  </div>
                ))
              )}
              <div ref={logsEndRef} />
            </div>
          </div>
        )}

        {/* Stats */}
        <div className="flex items-center justify-between text-sm text-muted-foreground">
          <div>
            共 {logs.length} 行日志
            {logs.length > 0 && (
              <>
                {' '}
                (
                {logs.filter((l) => l.stream === 'stdout').length} stdout,{' '}
                {logs.filter((l) => l.stream === 'stderr').length} stderr)
              </>
            )}
          </div>
          {streaming && (
            <div className="flex items-center gap-1 text-green-600">
              <span className="animate-pulse">●</span>
              <span>实时接收中...</span>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
