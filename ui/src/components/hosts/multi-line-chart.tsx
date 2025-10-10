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

interface DataLine {
  dataKey: string;
  name: string;
  color: string;
}

interface MultiLineChartProps {
  data: any[];
  title: string;
  lines: DataLine[];
  unit?: string;
  height?: number;
  formatValue?: (value: number) => string;
}

export function MultiLineChart({
  data,
  title,
  lines,
  unit = '',
  height = 300,
  formatValue,
}: MultiLineChartProps) {
  const defaultFormatter = (value: number) => {
    if (formatValue) return formatValue(value);
    return `${value.toFixed(2)}${unit}`;
  };

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
            tickFormatter={(value) => defaultFormatter(value)}
          />
          <Tooltip
            contentStyle={{
              backgroundColor: 'hsl(var(--popover))',
              border: '1px solid hsl(var(--border))',
              borderRadius: '8px',
            }}
            labelFormatter={(value) => formatTime(value, 'full')}
            formatter={(value: number, name: string) => [
              defaultFormatter(value),
              name,
            ]}
          />
          <Legend />
          {lines.map((line) => (
            <Line
              key={line.dataKey}
              type="monotone"
              dataKey={line.dataKey}
              stroke={line.color}
              strokeWidth={2}
              dot={false}
              name={line.name}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
