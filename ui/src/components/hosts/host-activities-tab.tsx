import { useQuery } from '@tanstack/react-query';
import { Clock, Terminal, User, Server, AlertCircle } from 'lucide-react';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Loader2 } from 'lucide-react';
import { devopsAPI } from '@/lib/api-client';

interface HostActivity {
  id: string;
  action: string;
  action_type: string;
  description: string;
  created_at: string;
  user_id?: string;
  client_ip?: string;
  metadata?: string;
}

interface HostActivitiesTabProps {
  hostId: string;
}

const ACTION_ICONS: Record<string, any> = {
  terminal: Terminal,
  agent: Server,
  user: User,
  system: AlertCircle,
};

const ACTION_TYPE_COLORS: Record<string, string> = {
  terminal: 'bg-blue-500',
  agent: 'bg-green-500',
  user: 'bg-purple-500',
  system: 'bg-orange-500',
};

const ACTION_LABELS: Record<string, string> = {
  terminal_created: '创建终端会话',
  terminal_closed: '关闭终端会话',
  terminal_replay: '回放终端会话',
  agent_connected: 'Agent 已连接',
  agent_disconnected: 'Agent 已断开',
  agent_reconnected: 'Agent 已重连',
  node_created: '创建节点',
  node_updated: '更新节点信息',
  node_deleted: '删除节点',
  system_alert: '系统告警',
  system_error: '系统错误',
};

export function HostActivitiesTab({ hostId }: HostActivitiesTabProps) {
  const { data, isLoading } = useQuery({
    queryKey: ['host-activities', hostId],
    queryFn: async () => {
      const response = await devopsAPI.vms.hosts.getActivities(hostId);
      return response.data;
    },
    refetchInterval: 30000, // Refresh every 30 seconds
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  const activities = data?.activities || [];

  if (activities.length === 0) {
    return (
      <div className="text-center py-12">
        <p className="text-muted-foreground">暂无活动记录</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="space-y-3">
        {activities.map((activity: HostActivity) => {
          const Icon = ACTION_ICONS[activity.action_type] || AlertCircle;
          const iconColor = ACTION_TYPE_COLORS[activity.action_type] || 'bg-gray-500';
          const actionLabel = ACTION_LABELS[activity.action] || activity.action;

          return (
            <Card key={activity.id}>
              <CardContent className="p-4">
                <div className="flex items-start gap-4">
                  {/* Icon */}
                  <div className={`flex-shrink-0 w-10 h-10 rounded-full ${iconColor} flex items-center justify-center`}>
                    <Icon className="h-5 w-5 text-white" />
                  </div>

                  {/* Content */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <p className="font-medium">{actionLabel}</p>
                      <Badge variant="outline" className="text-xs">
                        {activity.action_type}
                      </Badge>
                    </div>

                    {activity.description && (
                      <p className="text-sm text-muted-foreground mb-2">
                        {activity.description}
                      </p>
                    )}

                    <div className="flex items-center gap-4 text-xs text-muted-foreground">
                      <div className="flex items-center gap-1">
                        <Clock className="h-3 w-3" />
                        {new Date(activity.created_at).toLocaleString('zh-CN')}
                      </div>

                      {activity.client_ip && (
                        <div className="flex items-center gap-1">
                          <Server className="h-3 w-3" />
                          {activity.client_ip}
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>
    </div>
  );
}
