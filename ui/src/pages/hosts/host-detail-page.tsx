import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { MonitorChart } from '@/components/hosts/monitor-chart';
import { MultiLineChart } from '@/components/hosts/multi-line-chart';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group';
import { ArrowLeft, Terminal, Settings, Loader2 } from 'lucide-react';
import { formatBytes } from '@/lib/utils';
import { devopsAPI } from '@/lib/api-client';

type TimeRange = '15m' | '1h' | '6h' | '12h' | '1d' | '7d' | '30d';

const TIME_RANGES = {
  '15m': { label: '最近15分钟', duration: 15 * 60 * 1000, interval: '1m' },
  '1h': { label: '最近1小时', duration: 60 * 60 * 1000, interval: '1m' },
  '6h': { label: '最近6小时', duration: 6 * 60 * 60 * 1000, interval: '5m' },
  '12h': { label: '最近12小时', duration: 12 * 60 * 60 * 1000, interval: '5m' },
  '1d': { label: '最近1天', duration: 24 * 60 * 60 * 1000, interval: '5m' },
  '7d': { label: '最近一周', duration: 7 * 24 * 60 * 60 * 1000, interval: '1h' },
  '30d': { label: '最近一个月', duration: 30 * 24 * 60 * 60 * 1000, interval: '4h' },
};

export function HostDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [historyData, setHistoryData] = useState<any>({});
  const [timeRange, setTimeRange] = useState<TimeRange>(() => {
    // Load from localStorage
    const saved = localStorage.getItem('host-monitor-time-range');
    return (saved as TimeRange) || '1d';
  });

  // Fetch host data from API
  const { data: hostResponse, isLoading, isError } = useQuery({
    queryKey: ['host', id],
    queryFn: async () => {
      if (!id) throw new Error('No host ID provided');
      return devopsAPI.vms.hosts.get(id);
    },
    enabled: !!id,
    staleTime: 10000, // 10 seconds
  });

  const host = hostResponse?.data;

  // Save time range to localStorage
  useEffect(() => {
    localStorage.setItem('host-monitor-time-range', timeRange);
  }, [timeRange]);

  useEffect(() => {
    if (!id) return;

    // Fetch historical data
    const fetchHistory = async () => {
      try {
        const config = TIME_RANGES[timeRange];
        const end = new Date().toISOString();
        const start = new Date(Date.now() - config.duration).toISOString();

        const data: any = await devopsAPI.vms.hosts.getHistoryState(id, {
          start,
          end,
          interval: config.interval,
        });

        if (data.code === 0) {
          setHistoryData(processHistoryData(data.data.points));
        }
      } catch (error) {
        console.error('Failed to fetch history:', error);
      }
    };

    fetchHistory();
  }, [id, timeRange]);

  const processHistoryData = (points: any[]) => {
    return {
      cpu: points.map((p) => ({
        timestamp: p.timestamp,
        value: p.cpu_usage,
      })),
      memory: points.map((p) => ({
        timestamp: p.timestamp,
        value: p.mem_usage,
      })),
      disk: points.map((p) => ({
        timestamp: p.timestamp,
        value: p.disk_usage,
      })),
      network: points.map((p) => ({
        timestamp: p.timestamp,
        in: p.net_in_speed,
        out: p.net_out_speed,
      })),
      connections: points.map((p) => ({
        timestamp: p.timestamp,
        tcp: p.tcp_conn_count || 0,
        udp: p.udp_conn_count || 0,
      })),
    };
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (isError || !host) {
    return (
      <div className="text-center py-12">
        <p className="text-muted-foreground mb-4">
          {isError ? '加载失败' : '主机未找到'}
        </p>
        <Button onClick={() => navigate('/vms/hosts')} className="mt-4">
          返回列表
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate('/vms/hosts')}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-3xl font-bold">{host.name}</h1>
            {host.note && (
              <p className="text-muted-foreground mt-1">{host.note}</p>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          {host.host_info?.ssh_enabled && (
            <Button onClick={() => navigate(`/vms/hosts/${id}/ssh`)}>
              <Terminal className="mr-2 h-4 w-4" />
              SSH连接
            </Button>
          )}
          <Button variant="outline" onClick={() => navigate(`/vms/hosts/${id}/edit`)}>
            <Settings className="mr-2 h-4 w-4" />
            设置
          </Button>
        </div>
      </div>

      {/* Info Cards */}
      {host.host_info && (
        <div className="grid gap-4 md:grid-cols-4">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">
                操作系统
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-lg font-bold">
                {host.host_info.platform} {host.host_info.platform_version}
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">
                CPU
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-lg font-bold">
                {host.host_info.cpu_cores} 核心
              </p>
              <p className="text-xs text-muted-foreground truncate">
                {host.host_info.cpu_model}
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">
                内存
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-lg font-bold">
                {formatBytes(host.host_info.mem_total)}
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">
                磁盘
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-lg font-bold">
                {formatBytes(host.host_info.disk_total)}
              </p>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Monitoring Tabs */}
      <div className="space-y-4">
        {/* Time Range Selector */}
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">监控数据</h2>
          <ToggleGroup
            type="single"
            value={timeRange}
            onValueChange={(value) => value && setTimeRange(value as TimeRange)}
            className="justify-start"
          >
            {Object.entries(TIME_RANGES).map(([key, config]) => (
              <ToggleGroupItem key={key} value={key} aria-label={config.label}>
                {config.label}
              </ToggleGroupItem>
            ))}
          </ToggleGroup>
        </div>

        <Tabs defaultValue="overview" className="space-y-4">
        <TabsList>
          <TabsTrigger value="overview">概览</TabsTrigger>
          <TabsTrigger value="cpu">CPU</TabsTrigger>
          <TabsTrigger value="memory">内存</TabsTrigger>
          <TabsTrigger value="disk">磁盘</TabsTrigger>
          <TabsTrigger value="connections">连接数</TabsTrigger>
          <TabsTrigger value="network">网络</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            {historyData.cpu && (
              <Card>
                <CardContent className="pt-6">
                  <MonitorChart
                    data={historyData.cpu}
                    title="CPU 使用率"
                    unit="%"
                    color="#3b82f6"
                  />
                </CardContent>
              </Card>
            )}
            {historyData.memory && (
              <Card>
                <CardContent className="pt-6">
                  <MonitorChart
                    data={historyData.memory}
                    title="内存使用率"
                    unit="%"
                    color="#10b981"
                  />
                </CardContent>
              </Card>
            )}
            {historyData.connections && (
              <Card>
                <CardContent className="pt-6">
                  <MultiLineChart
                    data={historyData.connections}
                    title="连接数"
                    lines={[
                      { dataKey: 'tcp', name: 'TCP连接', color: '#3b82f6' },
                      { dataKey: 'udp', name: 'UDP连接', color: '#10b981' },
                    ]}
                    unit=" 个"
                  />
                </CardContent>
              </Card>
            )}
            {historyData.network && (
              <Card>
                <CardContent className="pt-6">
                  <MultiLineChart
                    data={historyData.network}
                    title="网络带宽"
                    lines={[
                      { dataKey: 'in', name: '下载', color: '#3b82f6' },
                      { dataKey: 'out', name: '上传', color: '#10b981' },
                    ]}
                    formatValue={(value) => `${formatBytes(value)}/s`}
                  />
                </CardContent>
              </Card>
            )}
          </div>
        </TabsContent>

        <TabsContent value="cpu">
          {historyData.cpu && (
            <Card>
              <CardContent className="pt-6">
                <MonitorChart
                  data={historyData.cpu}
                  title={`CPU 使用率（${TIME_RANGES[timeRange].label}）`}
                  unit="%"
                  color="#3b82f6"
                  height={400}
                />
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="memory">
          {historyData.memory && (
            <Card>
              <CardContent className="pt-6">
                <MonitorChart
                  data={historyData.memory}
                  title={`内存使用率（${TIME_RANGES[timeRange].label}）`}
                  unit="%"
                  color="#10b981"
                  height={400}
                />
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="disk">
          {historyData.disk && (
            <Card>
              <CardContent className="pt-6">
                <MonitorChart
                  data={historyData.disk}
                  title={`磁盘使用率（${TIME_RANGES[timeRange].label}）`}
                  unit="%"
                  color="#f59e0b"
                  height={400}
                />
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="connections">
          {historyData.connections && (
            <Card>
              <CardContent className="pt-6">
                <MultiLineChart
                  data={historyData.connections}
                  title={`连接数（${TIME_RANGES[timeRange].label}）`}
                  lines={[
                    { dataKey: 'tcp', name: 'TCP连接', color: '#3b82f6' },
                    { dataKey: 'udp', name: 'UDP连接', color: '#10b981' },
                  ]}
                  height={400}
                  unit=" 个"
                />
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="network">
          {historyData.network && (
            <Card>
              <CardContent className="pt-6">
                <MultiLineChart
                  data={historyData.network}
                  title={`网络带宽（${TIME_RANGES[timeRange].label}）`}
                  lines={[
                    { dataKey: 'in', name: '下载', color: '#3b82f6' },
                    { dataKey: 'out', name: '上传', color: '#10b981' },
                  ]}
                  height={400}
                  formatValue={(value) => `${formatBytes(value)}/s`}
                />
              </CardContent>
            </Card>
          )}
        </TabsContent>
      </Tabs>
      </div>
    </div>
  );
}
