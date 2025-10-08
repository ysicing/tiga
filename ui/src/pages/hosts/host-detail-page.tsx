import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ArrowLeft, Terminal, Settings, Loader2 } from 'lucide-react';
import { formatBytes } from '@/lib/utils';
import { devopsAPI } from '@/lib/api-client';
import { HostMonitorTab } from '@/components/hosts/host-monitor-tab';
import { HostActivitiesTab } from '@/components/hosts/host-activities-tab';
import { HostTerminalTab } from '@/components/hosts/host-terminal-tab';

export function HostDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState(() => {
    // Load from localStorage or URL param
    return localStorage.getItem('host-detail-active-tab') || 'activities';
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

  // Save active tab to localStorage
  useEffect(() => {
    localStorage.setItem('host-detail-active-tab', activeTab);
  }, [activeTab]);

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
          <Button onClick={() => navigate(`/vms/hosts/${id}/ssh`)} disabled={!host.online}>
            <Terminal className="mr-2 h-4 w-4" />
            打开终端
          </Button>
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

      {/* Main Tabs */}
      <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="activities">动态</TabsTrigger>
          <TabsTrigger value="monitor">监控</TabsTrigger>
          <TabsTrigger value="terminal">终端</TabsTrigger>
        </TabsList>

        {/* Activities Tab */}
        <TabsContent value="activities" className="space-y-4">
          {id && <HostActivitiesTab hostId={id} />}
        </TabsContent>

        {/* Monitor Tab */}
        <TabsContent value="monitor" className="space-y-4">
          {id && <HostMonitorTab hostId={id} />}
        </TabsContent>

        {/* Terminal Tab */}
        <TabsContent value="terminal" className="space-y-4">
          {id && <HostTerminalTab hostId={id} />}
        </TabsContent>
      </Tabs>
    </div>
  );
}
