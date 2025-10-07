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
            tickFormatter={(value) => defaultFormatter(value)}
          />
          <Tooltip
            contentStyle={{
              backgroundColor: 'hsl(var(--popover))',
              border: '1px solid hsl(var(--border))',
              borderRadius: '8px',
            }}
            labelFormatter={(value) => new Date(value).toLocaleString('zh-CN')}
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
