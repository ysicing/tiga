import React from 'react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Activity,
  TrendingUp,
  TrendingDown,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Plus,
  Eye,
} from 'lucide-react';
import { ServiceMonitorService, ServiceResponseItem } from '@/services/service-monitor';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { NetworkTopologyMatrix } from '@/components/service-monitor/network-topology-matrix';

const ServiceOverviewPage: React.FC = () => {
  const navigate = useNavigate();

  // Fetch 30-day overview data
  const { data: overviewData, isLoading } = useQuery({
    queryKey: ['service-overview'],
    queryFn: () => ServiceMonitorService.getOverview(),
    refetchInterval: 60000, // Refresh every minute
  });

  // Fetch network topology data
  const { data: topologyData } = useQuery({
    queryKey: ['network-topology'],
    queryFn: () => ServiceMonitorService.getNetworkTopology(1), // Last 1 hour
    refetchInterval: 60000, // Refresh every minute
  });

  const services = overviewData?.services ? Object.values(overviewData.services) : [];

  // Calculate overall statistics
  const totalServices = services.length;
  const goodServices = services.filter(s => s.status_code === 'Good').length;
  const lowAvailServices = services.filter(s => s.status_code === 'LowAvailability').length;
  const downServices = services.filter(s => s.status_code === 'Down').length;

  const avgUptime = services.length > 0
    ? services.reduce((acc, s) => acc + s.uptime_percentage, 0) / services.length
    : 0;

  // Get status badge
  const getStatusBadge = (statusCode: string) => {
    switch (statusCode) {
      case 'Good':
        return <Badge className="bg-green-500">优秀</Badge>;
      case 'LowAvailability':
        return <Badge className="bg-yellow-500">可用</Badge>;
      case 'Down':
        return <Badge variant="destructive">离线</Badge>;
      default:
        return <Badge variant="outline">未知</Badge>;
    }
  };

  // Get status icon
  const getStatusIcon = (statusCode: string) => {
    switch (statusCode) {
      case 'Good':
        return <CheckCircle className="h-4 w-4 text-green-500" />;
      case 'LowAvailability':
        return <AlertTriangle className="h-4 w-4 text-yellow-500" />;
      case 'Down':
        return <XCircle className="h-4 w-4 text-red-500" />;
      default:
        return <Activity className="h-4 w-4 text-gray-500" />;
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto" />
          <p className="mt-2 text-muted-foreground">加载中...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">服务监控概览</h1>
          <p className="text-muted-foreground">30天服务可用性统计和趋势分析</p>
        </div>
        <Button onClick={() => navigate('/vms/service-monitors/list')}>
          <Plus className="mr-2 h-4 w-4" />
          管理监控
        </Button>
      </div>

      {/* Statistics Cards */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">服务总数</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalServices}</div>
            <p className="text-xs text-muted-foreground mt-1">
              监控中的服务数量
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">优秀服务</CardTitle>
            <TrendingUp className="h-4 w-4 text-green-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-500">{goodServices}</div>
            <p className="text-xs text-muted-foreground mt-1">
              可用率 &gt; 95% 的服务
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">可用服务</CardTitle>
            <AlertTriangle className="h-4 w-4 text-yellow-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-yellow-500">{lowAvailServices}</div>
            <p className="text-xs text-muted-foreground mt-1">
              可用率 80-95% 的服务
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">离线服务</CardTitle>
            <TrendingDown className="h-4 w-4 text-red-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-red-500">{downServices}</div>
            <p className="text-xs text-muted-foreground mt-1">
              可用率 &lt; 80% 的服务
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Average Uptime Card */}
      <Card>
        <CardHeader>
          <CardTitle>平均可用率</CardTitle>
          <CardDescription>过去30天所有服务的平均可用率</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-4xl font-bold">{avgUptime.toFixed(2)}%</div>
        </CardContent>
      </Card>

      {/* Network Topology Matrix */}
      <NetworkTopologyMatrix
        data={topologyData || null}
        className="mb-6"
      />

      {/* Services List with 30-day Heatmap */}
      <Card>
        <CardHeader>
          <CardTitle>服务列表与30天可用性热力图</CardTitle>
          <CardDescription>
            每个方格代表一天的可用率 (深绿=高可用, 浅绿=中等, 红=故障, 灰=无数据)
          </CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[200px]">服务名称</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>30天可用率</TableHead>
                <TableHead>30天热力图 (今天→30天前)</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {services.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-center py-8">
                    暂无服务监控数据
                  </TableCell>
                </TableRow>
              ) : (
                services.map((service) => (
                  <TableRow key={service.service_monitor_id}>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        {getStatusIcon(service.status_code)}
                        <span className="font-medium">{service.service_name}</span>
                      </div>
                    </TableCell>
                    <TableCell>{getStatusBadge(service.status_code)}</TableCell>
                    <TableCell>
                      <span className="font-medium">{service.uptime_percentage.toFixed(2)}%</span>
                      <div className="text-xs text-muted-foreground">
                        {service.total_up} 成功 / {service.total_down} 失败
                      </div>
                    </TableCell>
                    <TableCell>
                      <ServiceHeatmap service={service} />
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => navigate(`/vms/service-monitors/${service.service_monitor_id}`)}
                      >
                        <Eye className="h-4 w-4 mr-1" />
                        查看详情
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
};

// 30-day Heatmap Component (GitHub contribution style)
const ServiceHeatmap: React.FC<{ service: ServiceResponseItem }> = ({ service }) => {
  // Calculate uptime percentage for each day
  const getDayUptimePercentage = (index: number) => {
    const up = service.up[index] || 0;
    const down = service.down[index] || 0;
    const total = up + down;
    if (total === 0) return null; // No data
    return (up / total) * 100;
  };

  // Get color based on uptime percentage
  const getColor = (uptimePercent: number | null) => {
    if (uptimePercent === null) return 'bg-gray-200'; // No data
    if (uptimePercent >= 99) return 'bg-green-700'; // Excellent
    if (uptimePercent >= 95) return 'bg-green-500'; // Good
    if (uptimePercent >= 80) return 'bg-yellow-400'; // Warning
    return 'bg-red-500'; // Down
  };

  // Get tooltip text
  const getTooltipText = (index: number, uptimePercent: number | null) => {
    const daysAgo = index === 0 ? '今天' : `${index}天前`;
    if (uptimePercent === null) return `${daysAgo}: 无数据`;
    return `${daysAgo}: ${uptimePercent.toFixed(2)}% 可用`;
  };

  return (
    <TooltipProvider>
      <div className="flex gap-1">
        {Array.from({ length: 30 }, (_, i) => {
          const uptimePercent = getDayUptimePercentage(i);
          return (
            <Tooltip key={i}>
              <TooltipTrigger asChild>
                <div
                  className={`w-3 h-3 rounded-sm ${getColor(uptimePercent)} cursor-pointer`}
                />
              </TooltipTrigger>
              <TooltipContent>
                <p className="text-xs">{getTooltipText(i, uptimePercent)}</p>
              </TooltipContent>
            </Tooltip>
          );
        })}
      </div>
    </TooltipProvider>
  );
};

export default ServiceOverviewPage;
