import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';

interface MetricDataPoint {
  timestamp: string;
  value: number;
}

interface MonitorChartProps {
  data: MetricDataPoint[];
  title: string;
  dataKey?: string;
  unit?: string;
  color?: string;
  height?: number;
}

export function MonitorChart({
  data,
  title,
  dataKey = 'value',
  unit = '%',
  color = '#3b82f6',
  height = 300,
}: MonitorChartProps) {
  // Convert UTC time to Beijing time (UTC+8)
  const toBeijingTime = (date: Date): Date => {
    return new Date(date.getTime() + (8 * 60 * 60 * 1000));
  };

  const formatTime = (value: string, format: 'short' | 'full' = 'short'): string => {
    const date = new Date(value);
    const beijingDate = toBeijingTime(date);

    if (format === 'short') {
      const hours = beijingDate.getHours().toString().padStart(2, '0');
      const minutes = beijingDate.getMinutes().toString().padStart(2, '0');
      return `${hours}:${minutes}`;
    } else {
      const year = beijingDate.getFullYear();
      const month = (beijingDate.getMonth() + 1).toString().padStart(2, '0');
      const day = beijingDate.getDate().toString().padStart(2, '0');
      const hours = beijingDate.getHours().toString().padStart(2, '0');
      const minutes = beijingDate.getMinutes().toString().padStart(2, '0');
      const seconds = beijingDate.getSeconds().toString().padStart(2, '0');
      return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
    }
  };

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
  );
}
