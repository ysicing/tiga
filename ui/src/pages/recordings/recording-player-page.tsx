import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { toast } from 'sonner'
import {
  IconArrowLeft,
  IconDownload,
  IconFileText,
  IconSettings,
} from '@tabler/icons-react'

import { recordingService } from '@/services/recording-service'
import { AsciinemaPlayer } from '@/components/recording/asciinema-player'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Separator } from '@/components/ui/separator'

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

export function RecordingPlayerPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const recordingsPath = '/system/recordings'

  // Player settings state
  const [autoPlay, setAutoPlay] = useState(false)
  const [loop, setLoop] = useState(false)
  const [speed, setSpeed] = useState('1.0')
  const [theme, setTheme] = useState('asciinema')
  const [showSettings, setShowSettings] = useState(false)

  // Fetch recording metadata
  const { data: recording, isLoading: metadataLoading } = useQuery({
    queryKey: ['recording', id],
    queryFn: () => recordingService.getRecording(id!),
    enabled: !!id,
  })

  // Fetch playback content
  const {
    data: content,
    isLoading: contentLoading,
    error,
  } = useQuery({
    queryKey: ['recording-playback', id],
    queryFn: () => recordingService.getPlaybackContent(id!),
    enabled: !!id,
    retry: 1,
  })

  const isLoading = metadataLoading || contentLoading

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

  // Handle player error
  const handlePlayerError = (error: Error) => {
    console.error('Player error:', error)
    toast.error('播放器错误: ' + error.message)
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => navigate(`${recordingsPath}/${id}`)}
          >
            <IconArrowLeft className="w-5 h-5" />
          </Button>
          <Skeleton className="h-8 w-48" />
        </div>
        <Card>
          <CardContent className="p-6">
            <Skeleton className="h-[500px]" />
          </CardContent>
        </Card>
      </div>
    )
  }

  if (error || !content) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => navigate(`${recordingsPath}/${id}`)}
          >
            <IconArrowLeft className="w-5 h-5" />
          </Button>
          <h1 className="text-3xl font-bold">录制播放</h1>
        </div>
        <Card>
          <CardHeader>
            <CardTitle className="text-destructive">加载失败</CardTitle>
            <CardDescription>
              {(error as any)?.message || '无法加载录制内容'}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex gap-2">
              <Button onClick={() => navigate(`${recordingsPath}/${id}`)}>
                <IconFileText className="w-4 h-4 mr-2" />
                查看详情
              </Button>
              <Button variant="outline" onClick={() => navigate(recordingsPath)}>
                返回列表
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => navigate(`${recordingsPath}/${id}`)}
          >
            <IconArrowLeft className="w-5 h-5" />
          </Button>
          <div>
            <h1 className="text-3xl font-bold">录制播放</h1>
            {recording && (
              <p className="text-muted-foreground text-sm mt-1">
                {recording.username} · {formatDuration(recording.duration)}
              </p>
            )}
          </div>
        </div>

        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="icon"
            onClick={() => setShowSettings(!showSettings)}
          >
            <IconSettings className="w-4 h-4" />
          </Button>
          <Button variant="outline" onClick={handleDownload}>
            <IconDownload className="w-4 h-4 mr-2" />
            下载
          </Button>
          {/* <Button variant="outline" onClick={() => navigate(`${recordingsPath}/${id}`)}>
            <IconFileText className="w-4 h-4 mr-2" />
            详情
          </Button> */}
        </div>
      </div>

      {/* Player Settings */}
      {showSettings && (
        <Card>
          <CardHeader>
            <CardTitle>播放器设置</CardTitle>
            <CardDescription>调整播放器行为和外观</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
              {/* Auto Play */}
              <div className="flex items-center justify-between space-x-2">
                <Label htmlFor="auto-play" className="text-sm font-medium">
                  自动播放
                </Label>
                <Switch
                  id="auto-play"
                  checked={autoPlay}
                  onCheckedChange={setAutoPlay}
                />
              </div>

              {/* Loop */}
              <div className="flex items-center justify-between space-x-2">
                <Label htmlFor="loop" className="text-sm font-medium">
                  循环播放
                </Label>
                <Switch id="loop" checked={loop} onCheckedChange={setLoop} />
              </div>

              {/* Speed */}
              <div className="space-y-2">
                <Label htmlFor="speed" className="text-sm font-medium">
                  播放速度
                </Label>
                <Select value={speed} onValueChange={setSpeed}>
                  <SelectTrigger id="speed">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="0.5">0.5x</SelectItem>
                    <SelectItem value="0.75">0.75x</SelectItem>
                    <SelectItem value="1.0">1.0x</SelectItem>
                    <SelectItem value="1.25">1.25x</SelectItem>
                    <SelectItem value="1.5">1.5x</SelectItem>
                    <SelectItem value="2.0">2.0x</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              {/* Theme */}
              <div className="space-y-2">
                <Label htmlFor="theme" className="text-sm font-medium">
                  主题
                </Label>
                <Select value={theme} onValueChange={setTheme}>
                  <SelectTrigger id="theme">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="asciinema">Asciinema</SelectItem>
                    <SelectItem value="tango">Tango</SelectItem>
                    <SelectItem value="solarized-dark">Solarized Dark</SelectItem>
                    <SelectItem value="solarized-light">Solarized Light</SelectItem>
                    <SelectItem value="monokai">Monokai</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Player */}
      <Card>
        <CardContent className="p-6">
          <AsciinemaPlayer
            content={content}
            options={{
              autoPlay,
              loop,
              speed: parseFloat(speed),
              theme: theme as any,
              fit: 'width',
              terminalFontSize: '15px',
              terminalLineHeight: 1.33,
            }}
            onError={handlePlayerError}
            className="rounded-lg overflow-hidden border"
          />
        </CardContent>
      </Card>

      {/* Recording Info (Collapsible) */}
      {recording && (
        <Card>
          <CardHeader>
            <CardTitle>录制信息</CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
              <div>
                <dt className="font-medium text-muted-foreground mb-1">用户</dt>
                <dd>{recording.username}</dd>
              </div>

              <div>
                <dt className="font-medium text-muted-foreground mb-1">时长</dt>
                <dd>{formatDuration(recording.duration)}</dd>
              </div>

              <div>
                <dt className="font-medium text-muted-foreground mb-1">
                  终端尺寸
                </dt>
                <dd>
                  {recording.rows} × {recording.cols}
                </dd>
              </div>

              <div>
                <dt className="font-medium text-muted-foreground mb-1">Shell</dt>
                <dd className="font-mono">{recording.shell}</dd>
              </div>

              <div>
                <dt className="font-medium text-muted-foreground mb-1">
                  录制类型
                </dt>
                <dd>{recording.recording_type}</dd>
              </div>

              <div>
                <dt className="font-medium text-muted-foreground mb-1">
                  存储类型
                </dt>
                <dd>{recording.storage_type}</dd>
              </div>
            </dl>

            {recording.description && (
              <>
                <Separator className="my-4" />
                <div>
                  <dt className="font-medium text-muted-foreground mb-2 text-sm">
                    描述
                  </dt>
                  <dd className="text-sm">{recording.description}</dd>
                </div>
              </>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  )
}
