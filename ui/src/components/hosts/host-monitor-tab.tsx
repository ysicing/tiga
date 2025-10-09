import { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { MonitorChart } from '@/components/hosts/monitor-chart';
import { MultiLineChart } from '@/components/hosts/multi-line-chart';
import { NodeProbeChart } from '@/components/service-monitor/node-probe-chart';
import { HttpProbeHeatmap } from '@/components/service-monitor/http-probe-heatmap';
import { Card, CardContent } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group';
import { formatBytes } from '@/lib/utils';
import { devopsAPI } from '@/lib/api-client';
import { ServiceMonitorService, ServiceHistoryInfo } from '@/services/service-monitor';

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

interface HostMonitorTabProps {
  hostId: string;
}

export function HostMonitorTab({ hostId }: HostMonitorTabProps) {
  const [historyData, setHistoryData] = useState<any>({});
  const [timeRange, setTimeRange] = useState<TimeRange>(() => {
    const saved = localStorage.getItem('host-monitor-time-range');
    return (saved as TimeRange) || '1d';
  });

  // Fetch probe history data
  const { data: probeHistory = [] } = useQuery({
    queryKey: ['host-probe-history', hostId, timeRange],
    queryFn: () => {
      const hours = parseInt(TIME_RANGES[timeRange].duration / (60 * 60 * 1000) as any);
      return ServiceMonitorService.getHostProbeHistory(hostId, hours);
    },
    enabled: !!hostId,
    refetchInterval: 30000, // Refresh every 30 seconds
  });

  // Group probe history by service type
  const groupedProbeHistory = {
    http: probeHistory.filter((item: ServiceHistoryInfo) => item.service_monitor_type === 'HTTP'),
    tcp: probeHistory.filter((item: ServiceHistoryInfo) => item.service_monitor_type === 'TCP'),
    icmp: probeHistory.filter((item: ServiceHistoryInfo) => item.service_monitor_type === 'ICMP'),
  };

  useEffect(() => {
    localStorage.setItem('host-monitor-time-range', timeRange);
  }, [timeRange]);

  useEffect(() => {
    if (!hostId) return;

    const fetchHistory = async () => {
      try {
        const config = TIME_RANGES[timeRange];
        const end = new Date().toISOString();
        const start = new Date(Date.now() - config.duration).toISOString();

        const data: any = await devopsAPI.vms.hosts.getHistoryState(hostId, {
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
  }, [hostId, timeRange]);

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

  return (
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
          <TabsTrigger value="probes">服务探测</TabsTrigger>
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

        <TabsContent value="probes" className="space-y-4">
          <div className="grid gap-4">
            {probeHistory.length > 0 ? (
              <>
                {/* HTTP Probes - Display with heatmap blocks */}
                {groupedProbeHistory.http.length > 0 && (
                  <HttpProbeHeatmap
                    data={groupedProbeHistory.http}
                    title={`HTTP 服务探测延迟（${TIME_RANGES[timeRange].label}）`}
                    description="使用方块显示 HTTP 服务的探测延迟，颜色表示响应速度"
                  />
                )}

                {/* TCP Probes - Display with line chart */}
                {groupedProbeHistory.tcp.length > 0 && (
                  <NodeProbeChart
                    data={groupedProbeHistory.tcp}
                    title={`TCP 服务探测延迟（${TIME_RANGES[timeRange].label}）`}
                    description="显示此节点对 TCP 服务的探测延迟变化趋势"
                    metricType="latency"
                  />
                )}

                {/* ICMP Probes - Display with line chart */}
                {groupedProbeHistory.icmp.length > 0 && (
                  <NodeProbeChart
                    data={groupedProbeHistory.icmp}
                    title={`ICMP 服务探测延迟（${TIME_RANGES[timeRange].label}）`}
                    description="显示此节点对 ICMP 服务的探测延迟变化趋势"
                    metricType="latency"
                  />
                )}
              </>
            ) : (
              <Card>
                <CardContent className="pt-6">
                  <div className="text-center py-8 text-gray-500">
                    该节点暂无服务探测任务
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}
