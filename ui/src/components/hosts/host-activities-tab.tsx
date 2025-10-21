import { useQuery } from '@tanstack/react-query'
import {
  AlertCircle,
  Clock,
  Loader2,
  Server,
  Terminal,
  User,
} from 'lucide-react'

import { devopsAPI } from '@/lib/api-client'

// T038: 适配统一审计 API 的数据结构
interface AuditEvent {
  id: string
  timestamp: number
  action: string
  subsystem: string
  resource: {
    type: string
    identifier: string
    data: Record<string, string>
  }
  user: {
    uid: string
    username: string
    type: string
  }
  client_ip: string
  user_agent: string
  data: Record<string, string>
  created_at: string
}

interface HostActivitiesTabProps {
  hostId: string
}

const ACTION_ICONS: Record<string, any> = {
  terminal: Terminal,
  agent: Server,
  user: User,
  system: AlertCircle,
}

const ACTION_TYPE_COLORS: Record<string, string> = {
  terminal: 'bg-blue-500',
  agent: 'bg-green-500',
  user: 'bg-purple-500',
  system: 'bg-orange-500',
}

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
}

export function HostActivitiesTab({ hostId }: HostActivitiesTabProps) {
  const { data, isLoading } = useQuery({
    queryKey: ['host-activities', hostId],
    queryFn: async () => {
      const response = await devopsAPI.vms.hosts.getActivities(hostId)
      return (response as any).data
    },
    refetchInterval: 30000, // Refresh every 30 seconds
  })

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  // T038: 适配统一审计 API 返回的数据结构
  const events = (data?.events || []) as AuditEvent[]

  if (events.length === 0) {
    return (
      <div className="text-center py-12">
        <p className="text-muted-foreground">暂无活动记录</p>
      </div>
    )
  }

  return (
    <div className="space-y-2">
      {events.map((event: AuditEvent) => {
        // 从 data 字段提取 action_type 和 description
        const actionType = event.data?.action_type || 'system'
        const description = event.data?.description || ''
        const Icon = ACTION_ICONS[actionType] || AlertCircle
        const iconColor = ACTION_TYPE_COLORS[actionType] || 'bg-gray-500'
        const actionLabel = ACTION_LABELS[event.action] || event.action

        return (
          <div
            key={event.id}
            className="flex items-center gap-3 px-4 py-2 rounded-lg border hover:bg-accent/50 transition-colors"
          >
            {/* Icon */}
            <div
              className={`flex-shrink-0 w-8 h-8 rounded-full ${iconColor} flex items-center justify-center`}
            >
              <Icon className="h-4 w-4 text-white" />
            </div>

            {/* Action Label */}
            <div className="flex-shrink-0 min-w-[100px]">
              <span className="text-sm font-medium">{actionLabel}</span>
            </div>

            {/* Description (optional, truncated) */}
            {description && (
              <div className="flex-1 min-w-0">
                <span className="text-sm text-muted-foreground truncate block">
                  {description}
                </span>
              </div>
            )}

            {/* Time */}
            <div className="flex-shrink-0 flex items-center gap-1 text-xs text-muted-foreground">
              <Clock className="h-3 w-3" />
              <span>
                {new Date(event.created_at).toLocaleString('zh-CN', {
                  month: '2-digit',
                  day: '2-digit',
                  hour: '2-digit',
                  minute: '2-digit',
                })}
              </span>
            </div>

            {/* Client IP (if available) */}
            {event.client_ip && (
              <div className="flex-shrink-0 flex items-center gap-1 text-xs text-muted-foreground">
                <Server className="h-3 w-3" />
                <span>{event.client_ip}</span>
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}
