import { useMemo } from 'react'
import { ServiceHistoryInfo } from '@/services/service-monitor'
import { format } from 'date-fns'
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

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

interface NodeProbeChartProps {
  data: ServiceHistoryInfo[]
  className?: string
  title?: string
  description?: string
  metricType?: 'latency' | 'uptime'
}

// Color palette for multiple lines (up to 10 services)
const LINE_COLORS = [
  '#3b82f6', // blue
  '#10b981', // green
  '#f59e0b', // amber
  '#ef4444', // red
  '#8b5cf6', // purple
  '#ec4899', // pink
  '#14b8a6', // teal
  '#f97316', // orange
  '#6366f1', // indigo
  '#84cc16', // lime
]

export function NodeProbeChart({
  data,
  className,
  title = '节点探测历史',
  description = '显示各节点对服务的探测延迟和可用性',
  metricType = 'latency',
}: NodeProbeChartProps) {
  const chartData = useMemo(() => {
    if (!data || data.length === 0) {
      return []
    }

    // Collect all unique timestamps across all services
    const timestampSet = new Set<number>()
    data.forEach((service) => {
      service.timestamps.forEach((ts) => timestampSet.add(ts))
    })

    // Sort timestamps
    const sortedTimestamps = Array.from(timestampSet).sort((a, b) => a - b)

    // Build chart data points
    const points = sortedTimestamps.map((ts) => {
      const point: any = {
        time: format(new Date(ts), 'HH:mm'),
        timestamp: ts,
      }

      // Add data for each service at this timestamp
      data.forEach((service) => {
        const tsIndex = service.timestamps.indexOf(ts)
        if (tsIndex !== -1) {
          const key = `${service.service_monitor_name}`
          if (metricType === 'latency') {
            point[key] = service.avg_delays[tsIndex]
          } else {
            point[key] = service.uptimes[tsIndex]
          }
        }
      })

      return point
    })

    return points
  }, [data, metricType])

  if (!data || data.length === 0) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle>{title}</CardTitle>
          <CardDescription>{description}</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-gray-500">暂无探测历史数据</div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className={className}>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={350}>
          <LineChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="time" tick={{ fontSize: 12 }} />
            <YAxis
              label={{
                value: metricType === 'latency' ? '延迟 (ms)' : '可用率 (%)',
                angle: -90,
                position: 'insideLeft',
              }}
              domain={metricType === 'uptime' ? [0, 100] : ['auto', 'auto']}
            />
            <Tooltip
              labelFormatter={(value) => `时间: ${value}`}
              formatter={(value: number, name: string) => [
                metricType === 'latency'
                  ? `${value.toFixed(2)}ms`
                  : `${value.toFixed(2)}%`,
                name,
              ]}
            />
            <Legend />
            {data.map((service, index) => (
              <Line
                key={service.service_monitor_id}
                type="monotone"
                dataKey={service.service_monitor_name}
                stroke={LINE_COLORS[index % LINE_COLORS.length]}
                strokeWidth={2}
                dot={false}
                connectNulls
              />
            ))}
          </LineChart>
        </ResponsiveContainer>

        {/* Service legend with stats */}
        <div className="mt-4 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
          {data.map((service, index) => {
            const avgDelay =
              service.avg_delays.length > 0
                ? service.avg_delays.reduce((a, b) => a + b, 0) /
                  service.avg_delays.length
                : 0
            const avgUptime =
              service.uptimes.length > 0
                ? service.uptimes.reduce((a, b) => a + b, 0) /
                  service.uptimes.length
                : 0

            return (
              <div
                key={service.service_monitor_id}
                className="flex items-center gap-2 text-sm p-2 rounded border"
              >
                <div
                  className="w-3 h-3 rounded-full"
                  style={{
                    backgroundColor: LINE_COLORS[index % LINE_COLORS.length],
                  }}
                />
                <div className="flex-1 min-w-0">
                  <p className="font-medium truncate">
                    {service.service_monitor_name}
                  </p>
                  <p className="text-xs text-gray-500">
                    {metricType === 'latency'
                      ? `平均: ${avgDelay.toFixed(2)}ms`
                      : `平均: ${avgUptime.toFixed(2)}%`}
                  </p>
                </div>
              </div>
            )
          })}
        </div>
      </CardContent>
    </Card>
  )
}
