import { useEffect, useState, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ArrowLeft, Terminal, Settings, Loader2, Key } from 'lucide-react';
import { formatBytes } from '@/lib/utils';
import { devopsAPI } from '@/lib/api-client';
import { HostMonitorTab } from '@/components/hosts/host-monitor-tab';
import { HostActivitiesTab } from '@/components/hosts/host-activities-tab';
import { HostTerminalTab } from '@/components/hosts/host-terminal-tab';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { toast } from 'sonner';
import { SecretKeyDisplay } from '@/components/hosts/SecretKeyDisplay';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';

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

  const host = (hostResponse as any)?.data;

  const [secretKey, setSecretKey] = useState<string | undefined>();
  const [installCommand, setInstallCommand] = useState<string | undefined>();
  const [credentialsOpen, setCredentialsOpen] = useState(false);
  const [credentialsLoading, setCredentialsLoading] = useState(false);
  const [credentialsError, setCredentialsError] = useState<string | null>(null);
  const credentialsFetchedRef = useRef(false);

  useEffect(() => {
    setSecretKey(host?.secret_key);
    setInstallCommand(host?.agent_install_cmd);
  }, [host?.secret_key, host?.agent_install_cmd]);

  useEffect(() => {
    if (!credentialsOpen || !id) {
      credentialsFetchedRef.current = false;
      return;
    }

    if (secretKey && installCommand) {
      credentialsFetchedRef.current = true;
      setCredentialsError(null);
      setCredentialsLoading(false);
      return;
    }

    if (credentialsFetchedRef.current) {
      return;
    }
    credentialsFetchedRef.current = true;

    let cancelled = false;

    const fetchCredentials = async () => {
      setCredentialsLoading(true);
      setCredentialsError(null);

      try {
        const response: any = await devopsAPI.vms.hosts.get(id);
        if (!cancelled) {
          if (response?.code === 0) {
            setSecretKey(response.data?.secret_key ?? '');
            setInstallCommand(response.data?.agent_install_cmd ?? '');
          } else {
            setCredentialsError(response?.message || '获取凭证失败');
          }
        }

        const cmdResponse: any = await devopsAPI.vms.hosts.getAgentInstallCommand(id);
        if (!cancelled) {
          if (cmdResponse?.code === 0) {
            setInstallCommand(cmdResponse.data?.agent_install_cmd ?? '');
          } else {
            setCredentialsError((prev) => prev || cmdResponse?.message || '获取安装命令失败');
          }
        }
      } catch (err) {
        if (!cancelled) {
          setCredentialsError('无法获取凭证');
        }
      } finally {
        if (!cancelled) {
          setCredentialsLoading(false);
        }
      }
    };

    fetchCredentials();

    return () => {
      cancelled = true;
    };
  }, [credentialsOpen, id, secretKey, installCommand]);

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
          {host.host_info?.ssh_enabled && (
            <Button onClick={() => navigate(`/vms/hosts/${id}/ssh`)} disabled={!host.online}>
              <Terminal className="mr-2 h-4 w-4" />
              终端
            </Button>
          )}
          <Button variant="outline" onClick={() => setCredentialsOpen(true)}>
            <Key className="mr-2 h-4 w-4" />
            凭证
          </Button>
          <Button variant="outline" onClick={() => navigate(`/vms/hosts/${id}/edit`)}>
            <Settings className="mr-2 h-4 w-4" />
            设置
          </Button>
        </div>
      </div>

      <Dialog open={credentialsOpen} onOpenChange={setCredentialsOpen}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Agent 凭证</DialogTitle>
            <DialogDescription>
              用于部署和认证主机 Agent，若凭证泄露请立即重置。
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="dialog-host-id" className="text-sm font-semibold">
                主机 ID
              </Label>
              <div className="flex gap-2">
                <Input
                  id="dialog-host-id"
                  value={host.id}
                  readOnly
                  className="font-mono text-sm"
                />
                <Button
                  type="button"
                  variant="outline"
                  onClick={async () => {
                    try {
                      await navigator.clipboard.writeText(host.id);
                      toast.success('已复制主机 ID');
                    } catch {
                      toast.error('无法复制主机 ID');
                    }
                  }}
                >
                  复制
                </Button>
              </div>
              <p className="text-xs text-muted-foreground">
                主机 ID 用于 Agent 连接认证，请在可信环境中使用。
              </p>
            </div>

            {credentialsLoading ? (
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin" />
                正在加载凭证...
              </div>
            ) : credentialsError ? (
              <div className="text-sm text-red-500">{credentialsError}</div>
            ) : (
              <SecretKeyDisplay
                hostId={host.id}
                secretKey={secretKey}
                agentInstallCmd={installCommand}
                onKeyRegenerated={(newKey, newCmd) => {
                  setSecretKey(newKey);
                  setInstallCommand(newCmd);
                }}
              />
            )}
          </div>
        </DialogContent>
      </Dialog>

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
        <TabsList className={`grid w-full ${host.host_info?.ssh_enabled ? 'grid-cols-3' : 'grid-cols-2'}`}>
          <TabsTrigger value="activities">动态</TabsTrigger>
          <TabsTrigger value="monitor">监控</TabsTrigger>
          {host.host_info?.ssh_enabled && <TabsTrigger value="terminal">终端</TabsTrigger>}
        </TabsList>

        {/* Activities Tab */}
        <TabsContent value="activities" className="space-y-4">
          {id && <HostActivitiesTab hostId={id} />}
        </TabsContent>

        {/* Monitor Tab */}
        <TabsContent value="monitor" className="space-y-4">
          {id && <HostMonitorTab hostId={id} />}
        </TabsContent>

        {/* Terminal Tab - only render if WebSSH is enabled */}
        {host.host_info?.ssh_enabled && (
          <TabsContent value="terminal" className="space-y-4">
            {id && <HostTerminalTab hostId={id} />}
          </TabsContent>
        )}
      </Tabs>
    </div>
  );
}
