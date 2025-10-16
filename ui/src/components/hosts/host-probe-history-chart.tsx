import { useState } from 'react'
import { ServiceHistoryInfo } from '@/services/service-monitor'
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

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

interface HostProbeHistoryChartProps {
  data: ServiceHistoryInfo[]
  hostName?: string
  height?: number
}

// Generate distinct colors for multiple lines
const generateColors = (count: number): string[] => {
  const colors = [
    '#8884d8', // Blue
    '#82ca9d', // Green
    '#ffc658', // Orange
    '#ff7c7c', // Red
    '#a48ce6', // Purple
    '#f59e0b', // Amber
    '#10b981', // Emerald
    '#3b82f6', // Sky blue
    '#ec4899', // Pink
    '#06b6d4', // Cyan
  ]

  if (count <= colors.length) {
    return colors.slice(0, count)
  }

  // If we need more colors, generate them dynamically
  const additionalColors: string[] = []
  for (let i = colors.length; i < count; i++) {
    const hue = (i * 137) % 360 // Golden angle for good distribution
    additionalColors.push(`hsl(${hue}, 70%, 50%)`)
  }

  return [...colors, ...additionalColors]
}

export function HostProbeHistoryChart({
  data,
  hostName,
  height = 400,
}: HostProbeHistoryChartProps) {
  const [activeTab, setActiveTab] = useState<'latency' | 'uptime'>('latency')

  // Format time
  const formatTime = (timestamp: number): string => {
    const date = new Date(timestamp)
    const hours = date.getHours().toString().padStart(2, '0')
    const minutes = date.getMinutes().toString().padStart(2, '0')
    return `${hours}:${minutes}`
  }

  const formatFullTime = (timestamp: number): string => {
    const date = new Date(timestamp)
    const year = date.getFullYear()
    const month = (date.getMonth() + 1).toString().padStart(2, '0')
    const day = date.getDate().toString().padStart(2, '0')
    const hours = date.getHours().toString().padStart(2, '0')
    const minutes = date.getMinutes().toString().padStart(2, '0')
    return `${year}-${month}-${day} ${hours}:${minutes}`
  }

  // Transform data for chart
  const transformData = (metric: 'latency' | 'uptime') => {
    if (data.length === 0) return []

    // Get all unique timestamps across all services
    const allTimestamps = new Set<number>()
    data.forEach((service) => {
      service.timestamps.forEach((ts) => allTimestamps.add(ts))
    })

    const sortedTimestamps = Array.from(allTimestamps).sort((a, b) => a - b)

    // Create chart data points
    return sortedTimestamps.map((timestamp) => {
      const point: any = { timestamp }

      data.forEach((service) => {
        const index = service.timestamps.indexOf(timestamp)
        if (index !== -1) {
          const value =
            metric === 'latency'
              ? service.avg_delays[index]
              : service.uptimes[index]
          point[service.service_monitor_id] = value
        }
      })

      return point
    })
  }

  const chartData = transformData(activeTab)
  const colors = generateColors(data.length)

  const formatValue = (value: number) => {
    if (activeTab === 'latency') {
      return `${value.toFixed(2)}ms`
    } else {
      return `${value.toFixed(1)}%`
    }
  }

  if (data.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Node Probe History</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center h-64 text-muted-foreground">
            No probe history data available
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Node Probe History {hostName && `- ${hostName}`}</CardTitle>
      </CardHeader>
      <CardContent>
        <Tabs
          value={activeTab}
          onValueChange={(v) => setActiveTab(v as 'latency' | 'uptime')}
        >
          <TabsList className="mb-4">
            <TabsTrigger value="latency">Latency</TabsTrigger>
            <TabsTrigger value="uptime">Uptime</TabsTrigger>
          </TabsList>

          <TabsContent value="latency" className="mt-0">
            <ResponsiveContainer width="100%" height={height}>
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                <XAxis
                  dataKey="timestamp"
                  tick={{ fontSize: 12 }}
                  tickFormatter={formatTime}
                />
                <YAxis
                  tick={{ fontSize: 12 }}
                  tickFormatter={(value) => `${value}ms`}
                  label={{
                    value: 'Latency (ms)',
                    angle: -90,
                    position: 'insideLeft',
                  }}
                />
                <Tooltip
                  contentStyle={{
                    backgroundColor: 'hsl(var(--popover))',
                    border: '1px solid hsl(var(--border))',
                    borderRadius: '8px',
                  }}
                  labelFormatter={formatFullTime}
                  formatter={formatValue}
                />
                <Legend />
                {data.map((service, index) => (
                  <Line
                    key={service.service_monitor_id}
                    type="monotone"
                    dataKey={service.service_monitor_id}
                    name={service.service_monitor_name}
                    stroke={colors[index]}
                    strokeWidth={2}
                    dot={false}
                    connectNulls
                  />
                ))}
              </LineChart>
            </ResponsiveContainer>
          </TabsContent>

          <TabsContent value="uptime" className="mt-0">
            <ResponsiveContainer width="100%" height={height}>
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                <XAxis
                  dataKey="timestamp"
                  tick={{ fontSize: 12 }}
                  tickFormatter={formatTime}
                />
                <YAxis
                  tick={{ fontSize: 12 }}
                  tickFormatter={(value) => `${value}%`}
                  label={{
                    value: 'Uptime (%)',
                    angle: -90,
                    position: 'insideLeft',
                  }}
                  domain={[0, 100]}
                />
                <Tooltip
                  contentStyle={{
                    backgroundColor: 'hsl(var(--popover))',
                    border: '1px solid hsl(var(--border))',
                    borderRadius: '8px',
                  }}
                  labelFormatter={formatFullTime}
                  formatter={formatValue}
                />
                <Legend />
                {data.map((service, index) => (
                  <Line
                    key={service.service_monitor_id}
                    type="monotone"
                    dataKey={service.service_monitor_id}
                    name={service.service_monitor_name}
                    stroke={colors[index]}
                    strokeWidth={2}
                    dot={false}
                    connectNulls
                  />
                ))}
              </LineChart>
            </ResponsiveContainer>
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  )
}
