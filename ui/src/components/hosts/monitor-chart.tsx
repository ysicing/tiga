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

interface MetricDataPoint {
  timestamp: string
  value: number
}

interface MonitorChartProps {
  data: MetricDataPoint[]
  title: string
  dataKey?: string
  unit?: string
  color?: string
  height?: number
}

export function MonitorChart({
  data,
  title,
  dataKey = 'value',
  unit = '%',
  color = '#3b82f6',
  height = 300,
}: MonitorChartProps) {
  const formatTime = (
    value: string,
    format: 'short' | 'full' = 'short'
  ): string => {
    const date = new Date(value)

    if (format === 'short') {
      return new Intl.DateTimeFormat('zh-CN', {
        hour: '2-digit',
        minute: '2-digit',
        hour12: false,
        timeZone: 'Asia/Shanghai',
      }).format(date)
    }

    return new Intl.DateTimeFormat('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
      timeZone: 'Asia/Shanghai',
    })
      .format(date)
      .replace(/\//g, '-')
      .replace(',', '')
  }

  return (
    <div className="w-full">
      <h3 className="text-sm font-medium mb-2">{title}</h3>
      <ResponsiveContainer width="100%" height={height}>
        <LineChart data={data}>
          <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
          <XAxis
            dataKey="timestamp"
            tick={{ fontSize: 12 }}
            tickFormatter={(value) => formatTime(value, 'short')}
          />
          <YAxis
            tick={{ fontSize: 12 }}
            tickFormatter={(value) => `${value}${unit}`}
          />
          <Tooltip
            contentStyle={{
              backgroundColor: 'hsl(var(--popover))',
              border: '1px solid hsl(var(--border))',
              borderRadius: '8px',
            }}
            labelFormatter={(value) => formatTime(value, 'full')}
            formatter={(value: number) => [`${value.toFixed(2)}${unit}`, title]}
          />
          <Legend />
          <Line
            type="monotone"
            dataKey={dataKey}
            stroke={color}
            strokeWidth={2}
            dot={false}
            name={title}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}
