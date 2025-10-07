import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { Host } from '@/stores/host-store';
import { formatBytes, formatUptime } from '@/lib/utils';
import {
  Server,
  Cpu,
  HardDrive,
  Network,
  Activity,
  Circle
} from 'lucide-react';

interface HostCardProps {
  host: Host;
  onClick?: () => void;
  compact?: boolean;
}

export function HostCard({ host, onClick, compact = false }: HostCardProps) {
  const state = host.current_state;
  const info = host.host_info;

  const getStatusColor = () => {
    if (!host.online) return 'bg-red-500';
    if (state && state.cpu_usage > 80) return 'bg-yellow-500';
    return 'bg-green-500';
  };

  const getUsageColor = (usage: number) => {
    if (usage >= 90) return 'bg-red-500';
    if (usage >= 70) return 'bg-yellow-500';
    return 'bg-blue-500';
  };

  return (
    <Card
      className={`hover:shadow-lg transition-shadow cursor-pointer ${
        !host.online ? 'opacity-60' : ''
      }`}
      onClick={onClick}
    >
      <CardHeader className={compact ? 'p-4 pb-2' : undefined}>
        <div className="flex items-center justify-between">
          <CardTitle className={`flex items-center gap-2 ${
            compact ? 'text-base' : 'text-lg'
          }`}>
            <Server className={compact ? 'h-4 w-4' : 'h-5 w-5'} />
            {host.name}
          </CardTitle>
          <div className="flex items-center gap-2">
            <Circle className={`h-2 w-2 ${getStatusColor()} fill-current`} />
            <Badge variant={host.online ? 'default' : 'secondary'}>
              {host.online ? '在线' : '离线'}
            </Badge>
          </div>
        </div>
        {host.note && !compact && (
          <p className="text-sm text-muted-foreground mt-1">{host.note}</p>
        )}
      </CardHeader>

      <CardContent className={compact ? 'p-4 pt-0' : undefined}>
        {!host.online ? (
          <div className="text-sm text-muted-foreground">
            主机离线 {host.last_active && `(最后在线: ${new Date(host.last_active).toLocaleString()})`}
          </div>
        ) : state ? (
          <div className="space-y-3">
            {/* CPU */}
            <div className="space-y-1">
              <div className="flex items-center justify-between text-sm">
                <div className="flex items-center gap-2">
                  <Cpu className="h-4 w-4" />
                  <span>CPU</span>
                </div>
                <span className="font-medium">{state.cpu_usage.toFixed(1)}%</span>
              </div>
              <Progress
                value={state.cpu_usage}
                className={`h-2 ${getUsageColor(state.cpu_usage)}`}
              />
            </div>

            {/* Memory */}
            <div className="space-y-1">
              <div className="flex items-center justify-between text-sm">
                <div className="flex items-center gap-2">
                  <Activity className="h-4 w-4" />
                  <span>内存</span>
                </div>
                <span className="font-medium">
                  {formatBytes(state.mem_used)} / {info ? formatBytes(info.mem_total) : 'N/A'}
                </span>
              </div>
              <Progress
                value={state.mem_usage}
                className={`h-2 ${getUsageColor(state.mem_usage)}`}
              />
            </div>

            {/* Disk */}
            <div className="space-y-1">
              <div className="flex items-center justify-between text-sm">
                <div className="flex items-center gap-2">
                  <HardDrive className="h-4 w-4" />
                  <span>磁盘</span>
                </div>
                <span className="font-medium">
                  {formatBytes(state.disk_used)} / {info ? formatBytes(info.disk_total) : 'N/A'}
                </span>
              </div>
              <Progress
                value={state.disk_usage}
                className={`h-2 ${getUsageColor(state.disk_usage)}`}
              />
            </div>

            {/* Network */}
            {!compact && (
              <div className="flex items-center justify-between text-sm pt-2 border-t">
                <div className="flex items-center gap-2">
                  <Network className="h-4 w-4" />
                  <span>网络</span>
                </div>
                <span className="text-xs">
                  ↓ {formatBytes(state.net_in_speed)}/s |
                  ↑ {formatBytes(state.net_out_speed)}/s
                </span>
              </div>
            )}

            {/* Uptime */}
            {!compact && state.uptime > 0 && (
              <div className="text-xs text-muted-foreground text-right">
                运行时间: {formatUptime(state.uptime)}
              </div>
            )}
          </div>
        ) : (
          <div className="text-sm text-muted-foreground">
            等待监控数据...
          </div>
        )}
      </CardContent>
    </Card>
  );
}
