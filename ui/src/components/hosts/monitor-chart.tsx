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
  return (
    <div className="w-full">
      <h3 className="text-sm font-medium mb-2">{title}</h3>
      <ResponsiveContainer width="100%" height={height}>
        <LineChart data={data}>
          <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
          <XAxis
            dataKey="timestamp"
            tick={{ fontSize: 12 }}
            tickFormatter={(value) => {
              const date = new Date(value);
              return date.toLocaleTimeString('zh-CN', {
                hour: '2-digit',
                minute: '2-digit',
              });
            }}
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
            labelFormatter={(value) => new Date(value).toLocaleString('zh-CN')}
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
