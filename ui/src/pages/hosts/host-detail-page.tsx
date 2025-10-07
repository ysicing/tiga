import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useHostStore } from '@/stores/host-store';
import { MonitorChart } from '@/components/hosts/monitor-chart';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ArrowLeft, Terminal, Settings } from 'lucide-react';
import { formatBytes } from '@/lib/utils';
import { devopsAPI } from '@/lib/api-client';

export function HostDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { hosts } = useHostStore();
  const [historyData, setHistoryData] = useState<any>({});

  const host = hosts.find((h) => h.id === id);

  useEffect(() => {
    if (!id) return;

    // Fetch historical data
    const fetchHistory = async () => {
      try {
        const end = new Date().toISOString();
        const start = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(); // Last 24h

        const data: any = await devopsAPI.vms.hosts.getHistoryState(id, {
          start,
          end,
          interval: '5m',
        });

        if (data.code === 0) {
          setHistoryData(processHistoryData(data.data.points));
        }
      } catch (error) {
        console.error('Failed to fetch history:', error);
      }
    };

    fetchHistory();
  }, [id]);

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
    };
  };

  if (!host) {
    return (
      <div className="text-center py-12">
        <p>主机未找到</p>
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
      <Tabs defaultValue="overview" className="space-y-4">
        <TabsList>
          <TabsTrigger value="overview">概览</TabsTrigger>
          <TabsTrigger value="cpu">CPU</TabsTrigger>
          <TabsTrigger value="memory">内存</TabsTrigger>
          <TabsTrigger value="disk">磁盘</TabsTrigger>
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
          </div>
        </TabsContent>

        <TabsContent value="cpu">
          {historyData.cpu && (
            <Card>
              <CardContent className="pt-6">
                <MonitorChart
                  data={historyData.cpu}
                  title="CPU 使用率（24小时）"
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
                  title="内存使用率（24小时）"
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
                  title="磁盘使用率（24小时）"
                  unit="%"
                  color="#f59e0b"
                  height={400}
                />
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="network">
          <div className="text-center py-12 text-muted-foreground">
            网络监控图表
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}
