import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useState, useEffect, useRef } from 'react';
import { Play, Monitor, Download, Loader2, XCircle, AlertTriangle } from 'lucide-react';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { devopsAPI } from '@/lib/api-client';
import { create } from 'asciinema-player';
import 'asciinema-player/dist/bundle/asciinema-player.css';

interface WebSSHSession {
  id: string;
  session_id: string;
  user_id: string;
  host_node_id: string;
  client_ip: string;
  status: 'active' | 'closed';
  start_time: string;
  end_time?: string;
  recording_enabled: boolean;
  recording_size: number;
  recording_format: string;
  cols: number;
  rows: number;
}

interface HostTerminalTabProps {
  hostId: string;
}

const formatDuration = (start: string, end?: string) => {
  const startTime = new Date(start).getTime();
  const endTime = end ? new Date(end).getTime() : Date.now();
  const duration = Math.floor((endTime - startTime) / 1000);

  if (duration < 60) return `${duration}秒`;
  if (duration < 3600) return `${Math.floor(duration / 60)}分钟`;
  return `${Math.floor(duration / 3600)}小时 ${Math.floor((duration % 3600) / 60)}分钟`;
};

const formatBytes = (bytes: number) => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
};

export function HostTerminalTab({ hostId }: HostTerminalTabProps) {
  const [selectedSession, setSelectedSession] = useState<string | null>(null);
  const [terminateSessionId, setTerminateSessionId] = useState<string | null>(null);
  const playerRef = useRef<HTMLDivElement>(null);
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ['host-sessions', hostId],
    queryFn: async () => {
      const response = await devopsAPI.vms.webssh.listAllSessions({
        host_id: hostId,
        page: 1,
        page_size: 50,
      });
      return (response as any).data;
    },
    refetchInterval: 5000, // Auto-refresh every 5 seconds
  });

  // Terminate session mutation
  const terminateMutation = useMutation({
    mutationFn: async (sessionId: string) => {
      return await devopsAPI.vms.webssh.closeSession(sessionId);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['host-sessions', hostId] });
      setTerminateSessionId(null);
    },
    onError: (error: any) => {
      console.error('Failed to terminate session:', error);
      alert('终止会话失败: ' + (error.message || '未知错误'));
    },
  });

  // Initialize asciinema player when selectedSession changes
  useEffect(() => {
    if (!selectedSession) return;

    // Wait for DOM to be ready
    const loadRecording = async () => {
      // Check if ref is available, if not wait a bit
      if (!playerRef.current) {
        console.log('playerRef not ready, waiting...');
        setTimeout(loadRecording, 100);
        return;
      }

      console.log('Loading recording for session:', selectedSession);
      // Clear any existing player
      playerRef.current.innerHTML = '';

      try {
        // Fetch with authentication headers (returns asciicast text string directly)
        const recording = await devopsAPI.vms.webssh.getRecording(selectedSession) as any;
        console.log('Recording fetched, length:', recording.length);

        // Create player with fetched asciicast data (pass as data object, not URL)
        create(
          { data: recording, parser: 'asciicast' },
          playerRef.current!,
          {
            fit: 'width',
            terminalFontSize: '14px',
            pauseOnMarkers: true,
            loop: false,
            autoPlay: false,
          }
        );
        console.log('Player created successfully');
      } catch (error) {
        console.error('Failed to load recording:', error);
        if (playerRef.current) {
          playerRef.current.innerHTML = '<p class="text-red-500 p-4">录制文件加载失败，请刷新重试</p>';
        }
      }
    };

    loadRecording();
  }, [selectedSession]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  const sessions = data?.sessions || [];
  const activeSessions = sessions.filter((s: WebSSHSession) => s.status === 'active');
  const closedSessions = sessions.filter((s: WebSSHSession) => s.status === 'closed');

  if (sessions.length === 0) {
    return (
      <div className="text-center py-12">
        <p className="text-muted-foreground mb-4">暂无终端会话记录</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header with "Open New Terminal" button */}
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold">终端会话</h3>
      </div>

      {/* Active Sessions Table */}
      {activeSessions.length > 0 && (
        <div>
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-lg font-semibold">活动会话</h3>
            <Badge variant="outline" className="bg-green-500/10 text-green-600">
              {activeSessions.length} 个活动中
            </Badge>
          </div>
          <Card>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>状态</TableHead>
                  <TableHead>开始时间</TableHead>
                  <TableHead>持续时间</TableHead>
                  <TableHead>客户端IP</TableHead>
                  <TableHead>终端尺寸</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {activeSessions.map((session: WebSSHSession) => (
                  <TableRow key={session.id}>
                    <TableCell>
                      <Badge variant="outline" className="bg-green-500/10 text-green-600">
                        活动中
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm">
                      {new Date(session.start_time).toLocaleString('zh-CN')}
                    </TableCell>
                    <TableCell className="text-sm">
                      {formatDuration(session.start_time)}
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      {session.client_ip}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {session.cols} × {session.rows}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        size="sm"
                        variant="destructive"
                        onClick={() => setTerminateSessionId(session.session_id)}
                      >
                        <XCircle className="mr-1 h-3 w-3" />
                        终止
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Card>
        </div>
      )}

      {/* Historical Sessions Table */}
      {closedSessions.length > 0 && (
        <div>
          <h3 className="text-lg font-semibold mb-3">历史会话</h3>
          <Card>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>状态</TableHead>
                  <TableHead>开始时间</TableHead>
                  <TableHead>持续时间</TableHead>
                  <TableHead>客户端IP</TableHead>
                  <TableHead>录制大小</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {closedSessions.map((session: WebSSHSession) => (
                  <TableRow key={session.id}>
                    <TableCell>
                      <Badge variant="secondary">已关闭</Badge>
                    </TableCell>
                    <TableCell className="text-sm">
                      {new Date(session.start_time).toLocaleString('zh-CN')}
                    </TableCell>
                    <TableCell className="text-sm">
                      {formatDuration(session.start_time, session.end_time)}
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      {session.client_ip}
                    </TableCell>
                    <TableCell className="text-sm">
                      {session.recording_enabled ? (
                        <div className="flex items-center gap-1 text-muted-foreground">
                          <Monitor className="h-3 w-3" />
                          {formatBytes(session.recording_size)}
                        </div>
                      ) : (
                        <span className="text-muted-foreground">-</span>
                      )}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-2">
                        {session.recording_enabled && (
                          <>
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => setSelectedSession(session.session_id)}
                            >
                              <Play className="mr-1 h-3 w-3" />
                              回放
                            </Button>
                            <Button
                              size="sm"
                              variant="ghost"
                              onClick={async () => {
                                try {
                                  // Fetch with authentication (returns asciicast text string directly)
                                  const recording = await devopsAPI.vms.webssh.getRecording(session.session_id);

                                  // Create blob and download
                                  const blob = new Blob([recording as string], { type: 'text/plain' });
                                  const url = window.URL.createObjectURL(blob);
                                  const link = document.createElement('a');
                                  link.href = url;
                                  link.download = `${session.session_id}.cast`;
                                  document.body.appendChild(link);
                                  link.click();
                                  document.body.removeChild(link);
                                  window.URL.revokeObjectURL(url);
                                } catch (error) {
                                  console.error('Failed to download recording:', error);
                                  alert('下载失败，请重试');
                                }
                              }}
                            >
                              <Download className="h-4 w-4" />
                            </Button>
                          </>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Card>
        </div>
      )}

      {/* Terminate Confirmation Dialog */}
      <Dialog open={!!terminateSessionId} onOpenChange={() => setTerminateSessionId(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-orange-500" />
              确认终止会话
            </DialogTitle>
            <DialogDescription>
              此操作将立即关闭终端会话。会话录制将被保存，但用户连接将被断开。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setTerminateSessionId(null)}>
              取消
            </Button>
            <Button
              variant="destructive"
              onClick={() => terminateSessionId && terminateMutation.mutate(terminateSessionId)}
              disabled={terminateMutation.isPending}
            >
              {terminateMutation.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              确认终止
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Playback Dialog */}
      <Dialog open={!!selectedSession} onOpenChange={() => setSelectedSession(null)}>
        <DialogContent className="max-w-5xl max-h-[90vh]">
          <DialogHeader>
            <DialogTitle>终端会话回放</DialogTitle>
            <DialogDescription>
              播放已录制的终端会话，您可以暂停、快进或重播操作记录
            </DialogDescription>
          </DialogHeader>
          <div className="mt-4">
            <div ref={playerRef} className="w-full" />
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
