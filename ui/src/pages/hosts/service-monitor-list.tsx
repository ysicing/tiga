import React, { useMemo, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Search,
  Plus,
  MoreVertical,
  Activity,
  Globe,
  Server,
  Wifi,
  Edit,
  Trash,
  Play,
  Pause,
  RefreshCw
} from 'lucide-react';
import { format } from 'date-fns';
import { toast } from 'sonner';
import { ServiceMonitorService, ServiceMonitorListResponse } from '@/services/service-monitor';

// Type for service monitor
interface ServiceMonitor {
  id: string;
  name: string;
  type: 'HTTP' | 'TCP' | 'ICMP';
  target: string;
  interval: number;
  enabled: boolean;
  status?: 'up' | 'down' | 'degraded' | 'unknown';
  last_check_time?: string;
  uptime_24h?: number;
  failure_threshold: number;
  notify_on_failure: boolean;
}

const ServiceMonitorListPage: React.FC = () => {
  const queryClient = useQueryClient();
  const [searchQuery, setSearchQuery] = useState('');
  const [typeFilter, setTypeFilter] = useState<string>('all');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedMonitor, setSelectedMonitor] = useState<ServiceMonitor | null>(null);

  // Fetch monitors
  const { data, isLoading } = useQuery<ServiceMonitorListResponse>({
    queryKey: ['service-monitors', searchQuery, typeFilter, statusFilter],
    queryFn: () => ServiceMonitorService.list({
      search: searchQuery,
      type: typeFilter === 'all' ? undefined : typeFilter,
      status: statusFilter === 'all' ? undefined : statusFilter,
    }),
  });
  const monitors = data?.items ?? [];
  const totalMonitors = data?.total ?? monitors.length;

  const filteredMonitors = useMemo(() => {
    return monitors.filter((monitor) => {
      const matchSearch =
        !searchQuery ||
        monitor.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        monitor.target.toLowerCase().includes(searchQuery.toLowerCase());

      const matchType = typeFilter === 'all' || monitor.type === typeFilter;
      const matchStatus = statusFilter === 'all' || monitor.status === statusFilter;

      return matchSearch && matchType && matchStatus;
    });
  }, [monitors, searchQuery, typeFilter, statusFilter]);

  // Delete monitor mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => ServiceMonitorService.delete(id),
    onSuccess: () => {
      toast.success('服务监控已删除');
      queryClient.invalidateQueries({ queryKey: ['service-monitors'] });
      setDeleteDialogOpen(false);
    },
    onError: () => {
      toast.error('删除服务监控失败');
    },
  });

  // Toggle enabled status
  const toggleMutation = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      ServiceMonitorService.update(id, { enabled }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['service-monitors'] });
      toast.success('监控状态已更新');
    },
  });

  // Trigger manual probe
  const triggerProbeMutation = useMutation({
    mutationFn: (id: string) => ServiceMonitorService.triggerProbe(id),
    onSuccess: () => {
      toast.success('探测任务已触发');
      queryClient.invalidateQueries({ queryKey: ['service-monitors'] });
    },
  });

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'HTTP':
        return <Globe className="h-4 w-4" />;
      case 'TCP':
        return <Server className="h-4 w-4" />;
      case 'ICMP':
        return <Wifi className="h-4 w-4" />;
      default:
        return <Activity className="h-4 w-4" />;
    }
  };

  const getStatusBadge = (status?: string) => {
    switch (status) {
      case 'up':
        return <Badge className="bg-green-500">在线</Badge>;
      case 'down':
        return <Badge variant="destructive">离线</Badge>;
      case 'degraded':
        return <Badge className="bg-yellow-500">降级</Badge>;
      default:
        return <Badge variant="outline">未知</Badge>;
    }
  };

  const getUptimeBadge = (uptime?: number) => {
    if (!uptime) return null;

    let variant: 'default' | 'destructive' | 'outline' = 'default';
    if (uptime >= 99.9) {
      variant = 'default';
    } else if (uptime >= 95) {
      variant = 'outline';
    } else {
      variant = 'destructive';
    }

    return (
      <Badge variant={variant}>
        {uptime.toFixed(2)}%
      </Badge>
    );
  };

  const handleDelete = (monitor: ServiceMonitor) => {
    setSelectedMonitor(monitor);
    setDeleteDialogOpen(true);
  };

  const confirmDelete = () => {
    if (selectedMonitor) {
      deleteMutation.mutate(selectedMonitor.id);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">服务监控</h1>
          <p className="text-muted-foreground">管理和监控服务健康状态</p>
        </div>
        <Button onClick={() => window.location.href = '/vms/service-monitors/new'}>
          <Plus className="mr-2 h-4 w-4" />
          新建监控
        </Button>
      </div>

      {/* Stats Cards */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">总监控数</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalMonitors}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">在线服务</CardTitle>
            <Activity className="h-4 w-4 text-green-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {filteredMonitors.filter(m => m.status === 'up').length}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">离线服务</CardTitle>
            <Activity className="h-4 w-4 text-red-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {filteredMonitors.filter(m => m.status === 'down').length}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">平均可用率</CardTitle>
            <Activity className="h-4 w-4 text-blue-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {filteredMonitors.length > 0
                ? (
                  filteredMonitors.reduce((acc, m) => acc + (m.uptime_24h || 0), 0) /
                  filteredMonitors.length
                ).toFixed(2)
                : '0.00'}%
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex gap-4">
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="搜索服务名称或目标..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-8"
                />
              </div>
            </div>
            <Select value={typeFilter} onValueChange={setTypeFilter}>
              <SelectTrigger className="w-[150px]">
                <SelectValue placeholder="类型" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部类型</SelectItem>
                <SelectItem value="HTTP">HTTP</SelectItem>
                <SelectItem value="TCP">TCP</SelectItem>
                <SelectItem value="ICMP">ICMP</SelectItem>
              </SelectContent>
            </Select>
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="w-[150px]">
                <SelectValue placeholder="状态" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部状态</SelectItem>
                <SelectItem value="up">在线</SelectItem>
                <SelectItem value="down">离线</SelectItem>
                <SelectItem value="degraded">降级</SelectItem>
                <SelectItem value="unknown">未知</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      {/* Monitor List */}
      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>服务名称</TableHead>
                <TableHead>类型</TableHead>
                <TableHead>目标</TableHead>
                <TableHead>检测间隔</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>24h可用率</TableHead>
                <TableHead>最后检测</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={8} className="text-center py-8">
                    加载中...
                  </TableCell>
                </TableRow>
              ) : filteredMonitors.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8} className="text-center py-8">
                    暂无监控服务
                  </TableCell>
                </TableRow>
              ) : (
                filteredMonitors.map((monitor) => (
                  <TableRow key={monitor.id}>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        {getTypeIcon(monitor.type)}
                        <a
                          href={`/vms/service-monitors/${monitor.id}`}
                          className="font-medium hover:underline"
                        >
                          {monitor.name}
                        </a>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">{monitor.type}</Badge>
                    </TableCell>
                    <TableCell className="font-mono text-sm">
                      {monitor.target}
                    </TableCell>
                    <TableCell>{monitor.interval}秒</TableCell>
                    <TableCell>{getStatusBadge(monitor.status)}</TableCell>
                    <TableCell>{getUptimeBadge(monitor.uptime_24h)}</TableCell>
                    <TableCell>
                      {monitor.last_check_time
                        ? format(new Date(monitor.last_check_time), 'MM-dd HH:mm:ss')
                        : '-'}
                    </TableCell>
                    <TableCell className="text-right">
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="icon">
                            <MoreVertical className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem
                            onClick={() => window.location.href = `/vms/service-monitors/${monitor.id}`}
                          >
                            <Edit className="mr-2 h-4 w-4" />
                            查看详情
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            onClick={() => triggerProbeMutation.mutate(monitor.id)}
                          >
                            <RefreshCw className="mr-2 h-4 w-4" />
                            手动探测
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            onClick={() =>
                              toggleMutation.mutate({
                                id: monitor.id,
                                enabled: !monitor.enabled,
                              })
                            }
                          >
                            {monitor.enabled ? (
                              <>
                                <Pause className="mr-2 h-4 w-4" />
                                暂停监控
                              </>
                            ) : (
                              <>
                                <Play className="mr-2 h-4 w-4" />
                                启用监控
                              </>
                            )}
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            onClick={() => handleDelete(monitor)}
                            className="text-red-500"
                          >
                            <Trash className="mr-2 h-4 w-4" />
                            删除
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
            <DialogDescription>
              确定要删除服务监控 "{selectedMonitor?.name}" 吗？此操作不可恢复。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteDialogOpen(false)}>
              取消
            </Button>
            <Button variant="destructive" onClick={confirmDelete}>
              删除
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default ServiceMonitorListPage;
