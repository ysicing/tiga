import { useQuery } from '@tanstack/react-query';
import { Clock, Terminal, User, Server, AlertCircle, Loader2 } from 'lucide-react';
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
    <div className="space-y-2">
      {activities.map((activity: HostActivity) => {
        const Icon = ACTION_ICONS[activity.action_type] || AlertCircle;
        const iconColor = ACTION_TYPE_COLORS[activity.action_type] || 'bg-gray-500';
        const actionLabel = ACTION_LABELS[activity.action] || activity.action;

        return (
          <div
            key={activity.id}
            className="flex items-center gap-3 px-4 py-2 rounded-lg border hover:bg-accent/50 transition-colors"
          >
            {/* Icon */}
            <div className={`flex-shrink-0 w-8 h-8 rounded-full ${iconColor} flex items-center justify-center`}>
              <Icon className="h-4 w-4 text-white" />
            </div>

            {/* Action Label */}
            <div className="flex-shrink-0 min-w-[100px]">
              <span className="text-sm font-medium">{actionLabel}</span>
            </div>

            {/* Description (optional, truncated) */}
            {activity.description && (
              <div className="flex-1 min-w-0">
                <span className="text-sm text-muted-foreground truncate block">
                  {activity.description}
                </span>
              </div>
            )}

            {/* Time */}
            <div className="flex-shrink-0 flex items-center gap-1 text-xs text-muted-foreground">
              <Clock className="h-3 w-3" />
              <span>{new Date(activity.created_at).toLocaleString('zh-CN', {
                month: '2-digit',
                day: '2-digit',
                hour: '2-digit',
                minute: '2-digit',
              })}</span>
            </div>

            {/* Client IP (if available) */}
            {activity.client_ip && (
              <div className="flex-shrink-0 flex items-center gap-1 text-xs text-muted-foreground">
                <Server className="h-3 w-3" />
                <span>{activity.client_ip}</span>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
