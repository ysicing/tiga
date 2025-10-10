import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { ServiceMonitor } from '@/stores/host-store';
import { CheckCircle2, XCircle, Clock } from 'lucide-react';

interface ServiceStatusProps {
  monitor: ServiceMonitor;
  lastProbeSuccess?: boolean;
  lastProbeTime?: string;
  uptime?: number; // Percentage 0-100
}

export function ServiceStatus({
  monitor,
  lastProbeSuccess,
  lastProbeTime,
  uptime = 0,
}: ServiceStatusProps) {
  const getStatusIcon = () => {
    if (lastProbeSuccess === undefined) {
      return <Clock className="h-5 w-5 text-yellow-500" />;
    }
    return lastProbeSuccess ? (
      <CheckCircle2 className="h-5 w-5 text-green-500" />
    ) : (
      <XCircle className="h-5 w-5 text-red-500" />
    );
  };

  const getStatusText = () => {
    if (lastProbeSuccess === undefined) return '等待探测';
    return lastProbeSuccess ? '正常' : '异常';
  };

  const getUptimeColor = () => {
    if (uptime >= 99) return 'text-green-600';
    if (uptime >= 95) return 'text-yellow-600';
    return 'text-red-600';
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <span className="text-base">{monitor.name}</span>
          <Badge variant={monitor.enabled ? 'default' : 'secondary'}>
            {monitor.enabled ? '已启用' : '已禁用'}
          </Badge>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-sm text-muted-foreground">状态</span>
            <div className="flex items-center gap-2">
              {getStatusIcon()}
              <span className="font-medium">{getStatusText()}</span>
            </div>
          </div>

          <div className="flex items-center justify-between">
            <span className="text-sm text-muted-foreground">类型</span>
            <Badge variant="outline">{monitor.type}</Badge>
          </div>

          <div className="flex items-center justify-between">
            <span className="text-sm text-muted-foreground">目标</span>
            <span className="text-sm font-mono truncate max-w-[200px]">
              {monitor.target}
            </span>
          </div>

          {uptime > 0 && (
            <div className="flex items-center justify-between pt-2 border-t">
              <span className="text-sm text-muted-foreground">可用性</span>
              <span className={`text-lg font-bold ${getUptimeColor()}`}>
                {uptime.toFixed(2)}%
              </span>
            </div>
          )}

          {lastProbeTime && (
            <div className="text-xs text-muted-foreground text-right">
              最后探测: {new Date(lastProbeTime).toLocaleString()}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
