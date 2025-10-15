import { useMemo } from 'react'
import type {
  NetworkTopologyEdge,
  NetworkTopologyResponse,
} from '@/services/service-monitor'

import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'

interface NetworkTopologyMatrixProps {
  data: NetworkTopologyResponse | null
  className?: string
}

function getLatencyColor(latency: number): string {
  if (latency === 0) return 'bg-gray-100 dark:bg-gray-800' // No data
  if (latency < 50) return 'bg-green-100 dark:bg-green-900' // Excellent
  if (latency < 100) return 'bg-yellow-100 dark:bg-yellow-900' // Good
  if (latency < 200) return 'bg-orange-100 dark:bg-orange-900' // Fair
  return 'bg-red-100 dark:bg-red-900' // Poor
}

function getLatencyColorClass(latency: number): string {
  if (latency === 0) return 'text-gray-600 dark:text-gray-400'
  if (latency < 50) return 'text-green-600 dark:text-green-400'
  if (latency < 100) return 'text-yellow-600 dark:text-yellow-400'
  if (latency < 200) return 'text-orange-600 dark:text-orange-400'
  return 'text-red-600 dark:text-red-400'
}

function formatLatency(latency: number): string {
  if (latency === 0) return '-'
  if (latency < 1) return '<1ms'
  return `${Math.round(latency)}ms`
}

export function NetworkTopologyMatrix({
  data,
  className,
}: NetworkTopologyMatrixProps) {
  const matrixData = useMemo(() => {
    if (!data || !data.nodes || data.nodes.length === 0) {
      return null
    }

    // Separate hosts and services
    const hosts = data.nodes.filter((n) => n.type === 'host')
    const services = data.nodes.filter((n) => n.type === 'service')

    // Build matrix data structure
    const matrix: Array<Array<NetworkTopologyEdge | null>> = []

    for (const host of hosts) {
      const row: Array<NetworkTopologyEdge | null> = []
      for (const service of services) {
        const edge = data.matrix?.[host.id]?.[service.id] || null
        row.push(edge)
      }
      matrix.push(row)
    }

    return {
      hosts,
      services,
      matrix,
    }
  }, [data])

  if (!matrixData) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle>网络拓扑矩阵</CardTitle>
          <CardDescription>显示节点间的网络连接状态和延迟</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-gray-500">暂无网络拓扑数据</div>
        </CardContent>
      </Card>
    )
  }

  const { hosts, services, matrix } = matrixData

  return (
    <Card className={className}>
      <CardHeader>
        <CardTitle>网络拓扑矩阵</CardTitle>
        <CardDescription>
          显示各监控节点对服务的探测延迟和连接状态
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto">
          <TooltipProvider>
            <table className="min-w-full">
              <thead>
                <tr>
                  <th className="sticky left-0 z-10 bg-background px-4 py-2 text-left font-medium">
                    节点 / 服务
                  </th>
                  {services.map((service) => (
                    <th
                      key={service.id}
                      className="px-2 py-2 text-center font-medium min-w-[100px]"
                    >
                      <div className="flex flex-col items-center gap-1">
                        <span
                          className="text-xs truncate max-w-[100px]"
                          title={service.name}
                        >
                          {service.name}
                        </span>
                        <Badge
                          variant={service.is_online ? 'default' : 'secondary'}
                          className={cn(
                            'text-xs',
                            service.is_online ? 'bg-green-500 text-white' : ''
                          )}
                        >
                          {service.is_online ? '在线' : '离线'}
                        </Badge>
                      </div>
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {hosts.map((host, rowIndex) => (
                  <tr key={host.id} className="border-t">
                    <td className="sticky left-0 z-10 bg-background px-4 py-2 font-medium">
                      <div className="flex items-center gap-2">
                        <span className="text-sm">{host.name}</span>
                        <Badge
                          variant={host.is_online ? 'default' : 'secondary'}
                          className={cn(
                            'text-xs',
                            host.is_online ? 'bg-green-500 text-white' : ''
                          )}
                        >
                          {host.is_online ? '在线' : '离线'}
                        </Badge>
                      </div>
                    </td>
                    {services.map((service, colIndex) => {
                      const edge = matrix[rowIndex][colIndex]

                      if (!edge) {
                        return (
                          <td
                            key={service.id}
                            className="px-2 py-2 text-center"
                          >
                            <div className="w-full h-12 flex items-center justify-center bg-gray-50 dark:bg-gray-900 rounded">
                              <span className="text-xs text-gray-400">N/A</span>
                            </div>
                          </td>
                        )
                      }

                      return (
                        <td key={service.id} className="px-2 py-2">
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <div
                                className={cn(
                                  'w-full h-12 flex flex-col items-center justify-center rounded cursor-pointer transition-all hover:scale-105',
                                  getLatencyColor(edge.avg_latency)
                                )}
                              >
                                <span
                                  className={cn(
                                    'text-sm font-semibold',
                                    getLatencyColorClass(edge.avg_latency)
                                  )}
                                >
                                  {formatLatency(edge.avg_latency)}
                                </span>
                                <span className="text-xs text-gray-600 dark:text-gray-400">
                                  {edge.success_rate.toFixed(0)}%
                                </span>
                              </div>
                            </TooltipTrigger>
                            <TooltipContent side="top" className="max-w-xs">
                              <div className="text-sm space-y-1">
                                <p className="font-medium">
                                  {host.name} → {service.name}
                                </p>
                                <div className="grid grid-cols-2 gap-x-2 text-xs">
                                  <p className="text-gray-300">平均延迟:</p>
                                  <p className="text-white font-medium">
                                    {formatLatency(edge.avg_latency)}
                                  </p>
                                  <p className="text-gray-300">最小延迟:</p>
                                  <p className="text-white">
                                    {formatLatency(edge.min_latency)}
                                  </p>
                                  <p className="text-gray-300">最大延迟:</p>
                                  <p className="text-white">
                                    {formatLatency(edge.max_latency)}
                                  </p>
                                  <p className="text-gray-300">成功率:</p>
                                  <p className="text-white">
                                    {edge.success_rate.toFixed(2)}%
                                  </p>
                                  <p className="text-gray-300">丢包率:</p>
                                  <p className="text-white">
                                    {edge.packet_loss.toFixed(2)}%
                                  </p>
                                  <p className="text-gray-300">探测次数:</p>
                                  <p className="text-white">
                                    {edge.probe_count}
                                  </p>
                                </div>
                                {edge.last_probe_time && (
                                  <p className="text-xs text-gray-400 pt-1">
                                    最后探测:{' '}
                                    {new Date(
                                      edge.last_probe_time
                                    ).toLocaleString()}
                                  </p>
                                )}
                              </div>
                            </TooltipContent>
                          </Tooltip>
                        </td>
                      )
                    })}
                  </tr>
                ))}
              </tbody>
            </table>
          </TooltipProvider>
        </div>

        {/* Legend */}
        <div className="mt-6 flex items-center gap-4 text-sm">
          <span className="text-gray-600 dark:text-gray-400">延迟图例:</span>
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 rounded bg-green-100 dark:bg-green-900" />
            <span className="text-gray-700 dark:text-gray-300">
              优秀 (&lt;50ms)
            </span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 rounded bg-yellow-100 dark:bg-yellow-900" />
            <span className="text-gray-700 dark:text-gray-300">
              良好 (50-100ms)
            </span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 rounded bg-orange-100 dark:bg-orange-900" />
            <span className="text-gray-700 dark:text-gray-300">
              一般 (100-200ms)
            </span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 rounded bg-red-100 dark:bg-red-900" />
            <span className="text-gray-700 dark:text-gray-300">
              较差 (&gt;200ms)
            </span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 rounded bg-gray-100 dark:bg-gray-800" />
            <span className="text-gray-700 dark:text-gray-300">无数据</span>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
