import { useEffect, useState } from 'react';
import { AlertBadge, AlertSeverity } from '@/components/hosts/alert-badge';
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
import { Search, RefreshCw, CheckCircle2, AlertCircle } from 'lucide-react';

type AlertEvent = {
  id: number;
  alert_id: number;
  host_id: number;
  severity: AlertSeverity;
  message: string;
  triggered_at: string;
  resolved_at?: string;
  acknowledged_at?: string;
  acknowledged_by?: string;
  host_name?: string;
  alert_name?: string;
};

export function AlertEventsPage() {
  const [events, setEvents] = useState<AlertEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [severityFilter, setSeverityFilter] = useState<string>('all');
  const [statusFilter, setStatusFilter] = useState<string>('active');

  useEffect(() => {
    fetchEvents();
  }, [statusFilter]);

  const fetchEvents = async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (statusFilter !== 'all') {
        if (statusFilter === 'active') {
          params.append('resolved', 'false');
        } else if (statusFilter === 'resolved') {
          params.append('resolved', 'true');
        }
      }

      const response = await fetch(`/api/v1/vms/alert-events?${params}`, {
        headers: {
          Authorization: `Bearer ${localStorage.getItem('token')}`,
        },
      });
      const data = await response.json();
      if (data.code === 0) {
        setEvents(data.data.items || []);
      }
    } catch (error) {
      console.error('Failed to fetch alert events:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleAcknowledge = async (eventId: number) => {
    try {
      const response = await fetch(
        `/api/v1/vms/alert-events/${eventId}/acknowledge`,
        {
          method: 'POST',
          headers: {
            Authorization: `Bearer ${localStorage.getItem('token')}`,
          },
        }
      );
      const data = await response.json();
      if (data.code === 0) {
        fetchEvents();
      }
    } catch (error) {
      console.error('Failed to acknowledge alert:', error);
    }
  };

  const handleResolve = async (eventId: number) => {
    try {
      const response = await fetch(
        `/api/v1/vms/alert-events/${eventId}/resolve`,
        {
          method: 'POST',
          headers: {
            Authorization: `Bearer ${localStorage.getItem('token')}`,
          },
        }
      );
      const data = await response.json();
      if (data.code === 0) {
        fetchEvents();
      }
    } catch (error) {
      console.error('Failed to resolve alert:', error);
    }
  };

  const filteredEvents = events.filter((event) => {
    const matchesSearch =
      event.message.toLowerCase().includes(searchTerm.toLowerCase()) ||
      event.host_name?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      event.alert_name?.toLowerCase().includes(searchTerm.toLowerCase());

    const matchesSeverity =
      severityFilter === 'all' || event.severity === severityFilter;

    return matchesSearch && matchesSeverity;
  });

  const stats = {
    total: events.length,
    critical: events.filter((e) => e.severity === 'critical' && !e.resolved_at)
      .length,
    warning: events.filter((e) => e.severity === 'warning' && !e.resolved_at)
      .length,
    acknowledged: events.filter((e) => e.acknowledged_at && !e.resolved_at)
      .length,
  };

  const getStatusBadge = (event: AlertEvent) => {
    if (event.resolved_at) {
      return <Badge variant="secondary">已解决</Badge>;
    }
    if (event.acknowledged_at) {
      return <Badge variant="outline">已确认</Badge>;
    }
    return <Badge variant="destructive">活跃</Badge>;
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

        <Select value={statusFilter} onValueChange={setStatusFilter}>
          <SelectTrigger className="w-[150px]">
            <SelectValue placeholder="状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="active">活跃</SelectItem>
            <SelectItem value="acknowledged">已确认</SelectItem>
            <SelectItem value="resolved">已解决</SelectItem>
            <SelectItem value="all">全部</SelectItem>
          </SelectContent>
        </Select>

        <Button variant="outline" onClick={fetchEvents} disabled={loading}>
          <RefreshCw
            className={`mr-2 h-4 w-4 ${loading ? 'animate-spin' : ''}`}
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
                  <TableHead>主机</TableHead>
                  <TableHead>告警</TableHead>
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
                      {event.host_name || `主机 #${event.host_id}`}
                    </TableCell>
                    <TableCell>
                      {event.alert_name || `告警 #${event.alert_id}`}
                    </TableCell>
                    <TableCell className="max-w-md truncate">
                      {event.message}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {new Date(event.triggered_at).toLocaleString()}
                    </TableCell>
                    <TableCell>{getStatusBadge(event)}</TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-2">
                        {!event.acknowledged_at && !event.resolved_at && (
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => handleAcknowledge(event.id)}
                          >
                            <CheckCircle2 className="mr-2 h-4 w-4" />
                            确认
                          </Button>
                        )}
                        {event.acknowledged_at && !event.resolved_at && (
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
    </div>
  );
}
