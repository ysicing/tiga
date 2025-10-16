import React, { useState } from 'react'
import { ServiceMonitorService } from '@/services/service-monitor'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { format, subDays, subHours } from 'date-fns'
import {
  Activity,
  AlertCircle,
  ArrowLeft,
  CheckCircle,
  Edit,
  Globe,
  Pause,
  Play,
  RefreshCw,
  Server,
  Wifi,
  XCircle,
} from 'lucide-react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { toast } from 'sonner'

import { devopsAPI } from '@/lib/api-client'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { AvailabilityHeatmap } from '@/components/service-monitor/availability-heatmap'

const ServiceMonitorDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [timePeriod, setTimePeriod] = useState<
    '15m' | '1h' | '6h' | '12h' | '1d' | '1w' | '1m'
  >('1d')

  // Fetch monitor details
  const { data: monitor, isLoading: monitorLoading } = useQuery({
    queryKey: ['service-monitor', id],
    queryFn: () => ServiceMonitorService.get(id!),
    enabled: !!id,
  })

  // Fetch probe history
  const { data: probeHistory = [] } = useQuery({
    queryKey: ['service-monitor-history', id, timePeriod],
    queryFn: () => {
      // For now, fetch all recent history without time filtering
      // The limit will restrict the number of results
      const limitMap = {
        '15m': 100,
        '1h': 200,
        '6h': 500,
        '12h': 720,
        '1d': 1440,
        '1w': 1000,
        '1m': 1000,
      }

      return ServiceMonitorService.getProbeHistory(id!, {
        limit: limitMap[timePeriod],
      })
    },
    enabled: !!id,
    refetchInterval: 30000, // Refresh every 30 seconds
  })

  // Fetch host nodes for displaying node names
  const { data: hostNodes = [] } = useQuery({
    queryKey: ['host-nodes'],
    queryFn: async () => {
      const res: any = await devopsAPI.vms.hosts.list()
      if (res?.code !== 0) {
        throw new Error(res?.message || 'Failed to fetch hosts')
      }
      return res.data?.items ?? []
    },
  })

  // Fetch availability stats
  const { data: stats } = useQuery({
    queryKey: ['service-monitor-stats', id, timePeriod],
    queryFn: () => {
      // Map our time periods to the API expected format
      const periodMap = {
        '15m': '15m',
        '1h': '1h',
        '6h': '6h',
        '12h': '12h',
        '1d': '24h',
        '1w': '7d',
        '1m': '30d',
      }
      return ServiceMonitorService.getAvailabilityStats(id!, {
        period: periodMap[timePeriod] as any,
      })
    },
    enabled: !!id,
  })

  // Fetch 30-day availability data for heatmap
  const { data: monthlyData } = useQuery({
    queryKey: ['service-monitor-monthly', id],
    queryFn: async () => {
      const overview = await ServiceMonitorService.getOverview()
      const serviceData = overview.services[id!]
      if (!serviceData) {
        return null
      }
      return {
        delay: serviceData.delay,
        up: serviceData.up,
        down: serviceData.down,
      }
    },
    enabled: !!id,
    refetchInterval: 60000, // Refresh every 60 seconds
  })

  // Toggle enabled mutation
  const toggleMutation = useMutation({
    mutationFn: (enabled: boolean) =>
      ServiceMonitorService.update(id!, { enabled }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['service-monitor', id] })
      toast.success('监控状态已更新')
    },
  })

  // Trigger manual probe
  const triggerProbeMutation = useMutation({
    mutationFn: () => ServiceMonitorService.triggerProbe(id!),
    onSuccess: () => {
      toast.success('探测任务已触发，正在执行...')
      // Invalidate all related queries to refresh data
      queryClient.invalidateQueries({ queryKey: ['service-monitor', id] })
      queryClient.invalidateQueries({
        queryKey: ['service-monitor-history', id],
      })
      queryClient.invalidateQueries({ queryKey: ['service-monitor-stats', id] })
      queryClient.invalidateQueries({
        queryKey: ['service-monitor-monthly', id],
      })
    },
    onError: (error) => {
      toast.error(`探测失败: ${error.message}`)
    },
  })

  const getTypeIcon = (type?: string) => {
    switch (type) {
      case 'HTTP':
        return <Globe className="h-5 w-5" />
      case 'TCP':
        return <Server className="h-5 w-5" />
      case 'ICMP':
        return <Wifi className="h-5 w-5" />
      default:
        return <Activity className="h-5 w-5" />
    }
  }

  const getStatusIcon = (status?: string) => {
    switch (status) {
      case 'up':
        return <CheckCircle className="h-5 w-5 text-green-500" />
      case 'down':
        return <XCircle className="h-5 w-5 text-red-500" />
      case 'degraded':
        return <AlertCircle className="h-5 w-5 text-yellow-500" />
      default:
        return <AlertCircle className="h-5 w-5 text-gray-500" />
    }
  }

  // Filter probe history based on selected time period
  const getFilteredHistory = () => {
    const now = new Date()
    let startTime: Date

    switch (timePeriod) {
      case '15m':
        startTime = new Date(now.getTime() - 15 * 60 * 1000)
        break
      case '1h':
        startTime = subHours(now, 1)
        break
      case '6h':
        startTime = subHours(now, 6)
        break
      case '12h':
        startTime = subHours(now, 12)
        break
      case '1d':
        startTime = subDays(now, 1)
        break
      case '1w':
        startTime = subDays(now, 7)
        break
      case '1m':
        startTime = subDays(now, 30)
        break
      default:
        startTime = subDays(now, 1)
    }

    return probeHistory.filter((result) => {
      const resultTime = new Date(result.timestamp)
      return resultTime >= startTime && resultTime <= now
    })
  }

  const filteredHistory = getFilteredHistory()

  // Prepare chart data with appropriate time formatting based on period
  const getTimeFormat = (date: Date) => {
    switch (timePeriod) {
      case '15m':
      case '1h':
      case '6h':
      case '12h':
        return format(date, 'HH:mm')
      case '1d':
        return format(date, 'HH:mm')
      case '1w':
        return format(date, 'MM-dd HH:mm')
      case '1m':
        return format(date, 'MM-dd')
      default:
        return format(date, 'HH:mm')
    }
  }

  // Prepare chart data with multi-node support
  const chartData = React.useMemo(() => {
    if (filteredHistory.length === 0) return []

    // Create a map of nodeId to nodeName for quick lookup
    const nodeNameMap: { [nodeId: string]: string } = {
      server: '服务器',
    }
    hostNodes.forEach((host: any) => {
      nodeNameMap[host.id] =
        host.name || host.hostname || `节点-${host.id.slice(0, 8)}`
    })

    // Group results by timestamp first
    const timeGroups: { [time: string]: { [nodeName: string]: number } } = {}

    filteredHistory
      .sort(
        (a, b) =>
          new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
      )
      .forEach((result) => {
        const time = getTimeFormat(new Date(result.timestamp))
        const nodeId = result.host_node_id || 'server'
        const nodeName = nodeNameMap[nodeId] || `节点-${nodeId.slice(0, 8)}`

        if (!timeGroups[time]) {
          timeGroups[time] = {}
        }

        // If multiple results for same node at same time, take average
        if (timeGroups[time][nodeName]) {
          timeGroups[time][nodeName] =
            (timeGroups[time][nodeName] + result.latency) / 2
        } else {
          timeGroups[time][nodeName] = result.latency
        }
      })

    // Convert to array format for recharts
    return Object.entries(timeGroups).map(([time, nodes]) => ({
      time,
      ...nodes,
    }))
  }, [filteredHistory, hostNodes])

  // Get all node names for chart lines
  const nodeNames = React.useMemo(() => {
    const nameSet = new Set<string>()

    // Create node name map
    const nodeNameMap: { [nodeId: string]: string } = {
      server: '服务器',
    }
    hostNodes.forEach((host: any) => {
      nodeNameMap[host.id] =
        host.name || host.hostname || `节点-${host.id.slice(0, 8)}`
    })

    // Collect all unique node names from history
    filteredHistory.forEach((result) => {
      const nodeId = result.host_node_id || 'server'
      const nodeName = nodeNameMap[nodeId] || `节点-${nodeId.slice(0, 8)}`
      nameSet.add(nodeName)
    })

    return Array.from(nameSet)
  }, [filteredHistory, hostNodes])

  // Colors for different nodes
  const nodeColors = [
    '#3b82f6',
    '#ef4444',
    '#10b981',
    '#f59e0b',
    '#8b5cf6',
    '#ec4899',
  ]

  // Calculate uptime timeline with appropriate grouping based on time period
  const getGroupingFormat = (date: Date) => {
    switch (timePeriod) {
      case '15m':
      case '1h':
        return format(date, 'HH:mm') // Group by minute
      case '6h':
      case '12h':
      case '1d':
        return format(date, 'HH:00') // Group by hour
      case '1w':
        return format(date, 'MM-dd') // Group by day
      case '1m':
        return format(date, 'MM-dd') // Group by day
      default:
        return format(date, 'HH:00')
    }
  }

  const uptimeData = filteredHistory
    .sort(
      (a, b) =>
        new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
    )
    .reduce((acc: any[], result) => {
      const groupKey = getGroupingFormat(new Date(result.timestamp))
      const existing = acc.find((item) => item.time === groupKey)

      if (existing) {
        existing.total += 1
        if (result.success) existing.successful += 1
      } else {
        acc.push({
          time: groupKey,
          total: 1,
          successful: result.success ? 1 : 0,
          percentage: 0,
        })
      }

      return acc
    }, [])
    .map((item) => ({
      ...item,
      percentage: (item.successful / item.total) * 100,
    }))

  if (monitorLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto" />
          <p className="mt-2 text-muted-foreground">加载中...</p>
        </div>
      </div>
    )
  }

  if (!monitor) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <p className="text-muted-foreground">监控服务未找到</p>
          <Button
            className="mt-4"
            onClick={() => navigate('/vms/service-monitors/list')}
          >
            返回列表
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => navigate('/vms/service-monitors/list')}
          >
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <div className="flex items-center gap-2">
              {getTypeIcon(monitor.type)}
              <h1 className="text-2xl font-bold">{monitor.name}</h1>
              {getStatusIcon(monitor.status)}
            </div>
            <p className="text-muted-foreground">{monitor.target}</p>
          </div>
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            onClick={() => triggerProbeMutation.mutate()}
            disabled={triggerProbeMutation.isPending}
          >
            <RefreshCw
              className={`mr-2 h-4 w-4 ${triggerProbeMutation.isPending ? 'animate-spin' : ''}`}
            />
            {triggerProbeMutation.isPending ? '探测中...' : '手动探测'}
          </Button>
          <Button
            variant="outline"
            onClick={() => toggleMutation.mutate(!monitor.enabled)}
          >
            {monitor.enabled ? (
              <>
                <Pause className="mr-2 h-4 w-4" />
                暂停监控
              </>
            ) : (
              <>
                <Play className="mr-2 h-4 w-4" />
                启用监控
              </>
            )}
          </Button>
          <Button onClick={() => navigate(`/vms/service-monitors/${id}/edit`)}>
            <Edit className="mr-2 h-4 w-4" />
            编辑
          </Button>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid gap-4 md:grid-cols-5">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">当前状态</CardTitle>
          </CardHeader>
          <CardContent>
            {monitor.status === 'up' && (
              <Badge className="bg-green-500">在线</Badge>
            )}
            {monitor.status === 'down' && (
              <Badge variant="destructive">离线</Badge>
            )}
            {monitor.status === 'degraded' && (
              <Badge className="bg-yellow-500">降级</Badge>
            )}
            {(monitor.status === 'unknown' || !monitor.status) && (
              <Badge variant="outline">未知</Badge>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">可用率</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {stats?.uptime_percentage?.toFixed(2) || 0}%
            </div>
            <p className="text-xs text-muted-foreground">
              {stats?.successful_checks || 0}/{stats?.total_checks || 0} 成功
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">平均延迟</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {stats?.avg_latency?.toFixed(2) || 0}ms
            </div>
            <p className="text-xs text-muted-foreground">
              最大: {stats?.max_latency || 0}ms
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">检测间隔</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{monitor.interval}s</div>
            <p className="text-xs text-muted-foreground">
              超时: {monitor.timeout}s
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">最后检测</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-sm">
              {monitor.last_check_time
                ? format(new Date(monitor.last_check_time), 'MM-dd HH:mm:ss')
                : '-'}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Time Period Selector */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>监控数据</CardTitle>
            <Select
              value={timePeriod}
              onValueChange={(value: any) => setTimePeriod(value)}
            >
              <SelectTrigger className="w-[150px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="15m">最近15分钟</SelectItem>
                <SelectItem value="1h">最近1小时</SelectItem>
                <SelectItem value="6h">最近6小时</SelectItem>
                <SelectItem value="12h">最近12小时</SelectItem>
                <SelectItem value="1d">最近1天</SelectItem>
                <SelectItem value="1w">最近1周</SelectItem>
                <SelectItem value="1m">最近1月</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardHeader>
      </Card>

      {/* Charts */}
      <Tabs defaultValue="latency" className="space-y-4">
        <TabsList>
          <TabsTrigger value="latency">响应时间</TabsTrigger>
          <TabsTrigger value="availability">可用性</TabsTrigger>
          <TabsTrigger value="logs">探测日志</TabsTrigger>
          <TabsTrigger value="config">配置信息</TabsTrigger>
        </TabsList>

        <TabsContent value="latency" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>响应时间趋势</CardTitle>
              <CardDescription>
                显示服务的响应时间变化趋势（毫秒）
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={300}>
                <LineChart data={chartData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="time" />
                  <YAxis />
                  <Tooltip />
                  <Legend />
                  {nodeNames.map((nodeName, index) => (
                    <Line
                      key={nodeName}
                      type="monotone"
                      dataKey={nodeName}
                      stroke={nodeColors[index % nodeColors.length]}
                      name={nodeName}
                      strokeWidth={2}
                      dot={false}
                      connectNulls
                    />
                  ))}
                </LineChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="availability" className="space-y-4">
          {monthlyData && (
            <Card>
              <CardHeader>
                <CardTitle>30天可用性热力图</CardTitle>
                <CardDescription>
                  显示最近30天的服务可用性趋势（绿色: &gt;95%, 橙色: 80-95%,
                  红色: &lt;80%）
                </CardDescription>
              </CardHeader>
              <CardContent>
                <AvailabilityHeatmap
                  serviceId={id!}
                  serviceName={monitor.name}
                  data={monthlyData}
                />
              </CardContent>
            </Card>
          )}

          <Card>
            <CardHeader>
              <CardTitle>可用性时间线</CardTitle>
              <CardDescription>
                {(timePeriod === '15m' || timePeriod === '1h') &&
                  '显示每分钟的服务可用率'}
                {(timePeriod === '6h' ||
                  timePeriod === '12h' ||
                  timePeriod === '1d') &&
                  '显示每小时的服务可用率'}
                {(timePeriod === '1w' || timePeriod === '1m') &&
                  '显示每天的服务可用率'}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {/* Status grid */}
                <div className="flex flex-wrap gap-1">
                  {uptimeData.map((item, index) => {
                    const getStatusColor = (percentage: number) => {
                      if (percentage >= 95) return 'bg-green-500'
                      if (percentage >= 80) return 'bg-yellow-500'
                      if (percentage > 0) return 'bg-red-500'
                      return 'bg-gray-300'
                    }

                    return (
                      <div
                        key={index}
                        className={`w-8 h-8 rounded ${getStatusColor(item.percentage)} transition-all hover:scale-110 cursor-pointer`}
                        title={`${item.time}: ${item.percentage.toFixed(1)}% (${item.successful}/${item.total})`}
                      />
                    )
                  })}
                </div>

                {/* Legend */}
                <div className="flex items-center gap-6 text-sm text-muted-foreground">
                  <div className="flex items-center gap-2">
                    <div className="w-4 h-4 rounded bg-green-500" />
                    <span>&gt;95%</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <div className="w-4 h-4 rounded bg-yellow-500" />
                    <span>80-95%</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <div className="w-4 h-4 rounded bg-red-500" />
                    <span>&lt;80%</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <div className="w-4 h-4 rounded bg-gray-300" />
                    <span>无数据</span>
                  </div>
                </div>

                {/* Summary statistics */}
                <div className="grid grid-cols-3 gap-4 pt-4 border-t">
                  <div>
                    <p className="text-sm text-muted-foreground">平均可用率</p>
                    <p className="text-2xl font-bold">
                      {uptimeData.length > 0
                        ? (
                            uptimeData.reduce(
                              (sum, item) => sum + item.percentage,
                              0
                            ) / uptimeData.length
                          ).toFixed(2)
                        : '0'}
                      %
                    </p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">总检测次数</p>
                    <p className="text-2xl font-bold">
                      {uptimeData.reduce((sum, item) => sum + item.total, 0)}
                    </p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">成功次数</p>
                    <p className="text-2xl font-bold">
                      {uptimeData.reduce(
                        (sum, item) => sum + item.successful,
                        0
                      )}
                    </p>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="logs" className="space-y-4">
          <Card>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>时间</TableHead>
                    <TableHead>执行节点</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead>延迟</TableHead>
                    {monitor.type === 'HTTP' && <TableHead>状态码</TableHead>}
                    <TableHead>错误信息</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredHistory.length === 0 ? (
                    <TableRow>
                      <TableCell
                        colSpan={monitor.type === 'HTTP' ? 6 : 5}
                        className="text-center py-8 text-muted-foreground"
                      >
                        暂无探测日志数据
                      </TableCell>
                    </TableRow>
                  ) : (
                    (() => {
                      // Create node name map outside of the loop for efficiency
                      const nodeNameMap: { [nodeId: string]: string } = {
                        server: '服务器',
                      }
                      hostNodes.forEach((host: any) => {
                        nodeNameMap[host.id] =
                          host.name ||
                          host.hostname ||
                          `节点-${host.id.slice(0, 8)}`
                      })

                      return filteredHistory
                        .sort(
                          (a, b) =>
                            new Date(b.timestamp).getTime() -
                            new Date(a.timestamp).getTime()
                        )
                        .slice(0, 50)
                        .map((result) => {
                          const nodeId = result.host_node_id || 'server'
                          const nodeName =
                            nodeNameMap[nodeId] || `节点-${nodeId.slice(0, 8)}`

                          return (
                            <TableRow key={result.id}>
                              <TableCell>
                                {format(
                                  new Date(result.timestamp),
                                  'MM-dd HH:mm:ss'
                                )}
                              </TableCell>
                              <TableCell>
                                <Badge variant="outline">{nodeName}</Badge>
                              </TableCell>
                              <TableCell>
                                {result.success ? (
                                  <Badge className="bg-green-500">成功</Badge>
                                ) : (
                                  <Badge variant="destructive">失败</Badge>
                                )}
                              </TableCell>
                              <TableCell>{result.latency}ms</TableCell>
                              {monitor.type === 'HTTP' && (
                                <TableCell>
                                  {result.http_status_code || '-'}
                                </TableCell>
                              )}
                              <TableCell className="max-w-xs truncate">
                                {result.error_message || '-'}
                              </TableCell>
                            </TableRow>
                          )
                        })
                    })()
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="config" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>监控配置</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm font-medium">类型</p>
                  <p className="text-sm text-muted-foreground">
                    {monitor.type}
                  </p>
                </div>
                <div>
                  <p className="text-sm font-medium">目标</p>
                  <p className="text-sm text-muted-foreground font-mono">
                    {monitor.target}
                  </p>
                </div>
                <div>
                  <p className="text-sm font-medium">检测间隔</p>
                  <p className="text-sm text-muted-foreground">
                    {monitor.interval}秒
                  </p>
                </div>
                <div>
                  <p className="text-sm font-medium">超时时间</p>
                  <p className="text-sm text-muted-foreground">
                    {monitor.timeout}秒
                  </p>
                </div>
                {monitor.type === 'HTTP' && (
                  <>
                    <div>
                      <p className="text-sm font-medium">HTTP方法</p>
                      <p className="text-sm text-muted-foreground">
                        {monitor.http_method || 'GET'}
                      </p>
                    </div>
                    <div>
                      <p className="text-sm font-medium">期望状态码</p>
                      <p className="text-sm text-muted-foreground">
                        {monitor.expect_status || 200}
                      </p>
                    </div>
                  </>
                )}
                <div>
                  <p className="text-sm font-medium">失败阈值</p>
                  <p className="text-sm text-muted-foreground">
                    {monitor.failure_threshold}次连续失败
                  </p>
                </div>
                <div>
                  <p className="text-sm font-medium">恢复阈值</p>
                  <p className="text-sm text-muted-foreground">
                    {monitor.recovery_threshold}次连续成功
                  </p>
                </div>
                <div>
                  <p className="text-sm font-medium">失败通知</p>
                  <p className="text-sm text-muted-foreground">
                    {monitor.notify_on_failure ? '启用' : '禁用'}
                  </p>
                </div>
                <div>
                  <p className="text-sm font-medium">创建时间</p>
                  <p className="text-sm text-muted-foreground">
                    {format(
                      new Date(monitor.created_at),
                      'yyyy-MM-dd HH:mm:ss'
                    )}
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}

export default ServiceMonitorDetailPage
