import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { AlertBadge } from '@/components/hosts/alert-badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Search, RefreshCw, CheckCircle2, AlertCircle } from 'lucide-react';
import { AlertRuleService, AlertStatus } from '@/services/alert-rule';
import { toast } from 'sonner';

export function AlertEventsPage() {
  const queryClient = useQueryClient();
  const [searchTerm, setSearchTerm] = useState('');
  const [severityFilter, setSeverityFilter] = useState<string>('all');
  const [statusFilter, setStatusFilter] = useState<AlertStatus | 'all'>('firing');
  const [isAckDialogOpen, setIsAckDialogOpen] = useState(false);
  const [isResolveDialogOpen, setIsResolveDialogOpen] = useState(false);
  const [selectedEventId, setSelectedEventId] = useState<string | null>(null);
  const [note, setNote] = useState('');

  // Fetch alert events
  const { data: eventsData, isLoading } = useQuery({
    queryKey: ['alert-events', statusFilter],
    queryFn: () => AlertRuleService.listEvents({
      status: statusFilter === 'all' ? undefined : statusFilter,
      page: 1,
      page_size: 100,
    }),
    refetchInterval: 30000, // Refresh every 30 seconds
  });

  // Acknowledge mutation
  const acknowledgeMutation = useMutation({
    mutationFn: ({ id, note }: { id: string; note: string }) =>
      AlertRuleService.acknowledgeEvent(id, note),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['alert-events'] });
      setIsAckDialogOpen(false);
      setNote('');
      toast.success('告警已确认');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.message || '确认告警失败');
    },
  });

  // Resolve mutation
  const resolveMutation = useMutation({
    mutationFn: ({ id, note }: { id: string; note: string }) =>
      AlertRuleService.resolveEvent(id, note),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['alert-events'] });
      setIsResolveDialogOpen(false);
      setNote('');
      toast.success('告警已解决');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.message || '解决告警失败');
    },
  });

  const handleAcknowledge = (eventId: string) => {
    setSelectedEventId(eventId);
    setIsAckDialogOpen(true);
  };

  const confirmAcknowledge = () => {
    if (selectedEventId) {
      acknowledgeMutation.mutate({ id: selectedEventId, note });
    }
  };

  const handleResolve = (eventId: string) => {
    setSelectedEventId(eventId);
    setIsResolveDialogOpen(true);
  };

  const confirmResolve = () => {
    if (selectedEventId) {
      resolveMutation.mutate({ id: selectedEventId, note });
    }
  };

  const events = eventsData?.items || [];

  const filteredEvents = events.filter((event) => {
    const matchesSearch =
      event.message.toLowerCase().includes(searchTerm.toLowerCase()) ||
      event.target_name?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      event.rule_name?.toLowerCase().includes(searchTerm.toLowerCase());

    const matchesSeverity =
      severityFilter === 'all' || event.severity === severityFilter;

    return matchesSearch && matchesSeverity;
  });

  const stats = {
    total: events.length,
    critical: events.filter((e) => e.severity === 'critical' && e.status === 'firing')
      .length,
    warning: events.filter((e) => e.severity === 'warning' && e.status === 'firing')
      .length,
    acknowledged: events.filter((e) => e.status === 'acknowledged').length,
  };

  const getStatusBadge = (status: AlertStatus) => {
    switch (status) {
      case 'resolved':
        return <Badge variant="secondary">已解决</Badge>;
      case 'acknowledged':
        return <Badge variant="outline">已确认</Badge>;
      case 'firing':
        return <Badge variant="destructive">触发中</Badge>;
      default:
        return <Badge variant="outline">{status}</Badge>;
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">告警事件</h1>
          <p className="text-muted-foreground mt-1">
            查看和管理系统告警事件
          </p>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              总事件数
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{stats.total}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              严重告警
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold text-red-600">{stats.critical}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              警告告警
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold text-yellow-600">{stats.warning}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              已确认
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold text-blue-600">
              {stats.acknowledged}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-4">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="搜索事件..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="pl-9"
          />
        </div>

        <Select value={severityFilter} onValueChange={setSeverityFilter}>
          <SelectTrigger className="w-[150px]">
            <SelectValue placeholder="严重程度" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部</SelectItem>
            <SelectItem value="critical">严重</SelectItem>
            <SelectItem value="warning">警告</SelectItem>
            <SelectItem value="info">信息</SelectItem>
          </SelectContent>
        </Select>

        <Select value={statusFilter} onValueChange={(v) => setStatusFilter(v as AlertStatus | 'all')}>
          <SelectTrigger className="w-[150px]">
            <SelectValue placeholder="状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="firing">触发中</SelectItem>
            <SelectItem value="acknowledged">已确认</SelectItem>
            <SelectItem value="resolved">已解决</SelectItem>
            <SelectItem value="all">全部</SelectItem>
          </SelectContent>
        </Select>

        <Button
          variant="outline"
          onClick={() => queryClient.invalidateQueries({ queryKey: ['alert-events'] })}
          disabled={isLoading}
        >
          <RefreshCw
            className={`mr-2 h-4 w-4 ${isLoading ? 'animate-spin' : ''}`}
          />
          刷新
        </Button>
      </div>

      {/* Events Table */}
      {filteredEvents.length === 0 ? (
        <div className="text-center py-12">
          <p className="text-muted-foreground">
            {events.length === 0
              ? '暂无告警事件'
              : '没有找到匹配的告警事件'}
          </p>
        </div>
      ) : (
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>严重程度</TableHead>
                  <TableHead>告警规则</TableHead>
                  <TableHead>目标</TableHead>
                  <TableHead>消息</TableHead>
                  <TableHead>触发时间</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredEvents.map((event) => (
                  <TableRow key={event.id}>
                    <TableCell>
                      <AlertBadge severity={event.severity} showIcon />
                    </TableCell>
                    <TableCell className="font-medium">
                      {event.rule_name || '未知规则'}
                    </TableCell>
                    <TableCell>
                      {event.target_name || event.target_id.substring(0, 8)}
                    </TableCell>
                    <TableCell className="max-w-md truncate">
                      {event.message}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {new Date(event.started_at).toLocaleString()}
                    </TableCell>
                    <TableCell>{getStatusBadge(event.status)}</TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-2">
                        {event.status === 'firing' && (
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => handleAcknowledge(event.id)}
                          >
                            <CheckCircle2 className="mr-2 h-4 w-4" />
                            确认
                          </Button>
                        )}
                        {event.status === 'acknowledged' && (
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => handleResolve(event.id)}
                          >
                            <AlertCircle className="mr-2 h-4 w-4" />
                            解决
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {/* Acknowledge Dialog */}
      <Dialog open={isAckDialogOpen} onOpenChange={setIsAckDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认告警</DialogTitle>
            <DialogDescription>
              标记此告警为已确认状态，并添加确认备注。
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="ack-note">确认备注（可选）</Label>
              <Textarea
                id="ack-note"
                value={note}
                onChange={(e) => setNote(e.target.value)}
                placeholder="描述已采取的措施..."
                rows={4}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setIsAckDialogOpen(false);
                setNote('');
              }}
            >
              取消
            </Button>
            <Button
              onClick={confirmAcknowledge}
              disabled={acknowledgeMutation.isPending}
            >
              {acknowledgeMutation.isPending ? '确认中...' : '确认'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Resolve Dialog */}
      <Dialog open={isResolveDialogOpen} onOpenChange={setIsResolveDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>解决告警</DialogTitle>
            <DialogDescription>
              标记此告警为已解决状态，并添加解决备注。
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="resolve-note">解决备注（可选）</Label>
              <Textarea
                id="resolve-note"
                value={note}
                onChange={(e) => setNote(e.target.value)}
                placeholder="描述解决方案..."
                rows={4}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setIsResolveDialogOpen(false);
                setNote('');
              }}
            >
              取消
            </Button>
            <Button
              onClick={confirmResolve}
              disabled={resolveMutation.isPending}
            >
              {resolveMutation.isPending ? '解决中...' : '解决'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
