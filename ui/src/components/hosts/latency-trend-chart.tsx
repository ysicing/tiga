import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  ReferenceLine,
} from 'recharts';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { ServiceProbeResult } from '@/services/service-monitor';

interface LatencyTrendChartProps {
  data: ServiceProbeResult[];
  title?: string;
  height?: number;
  threshold?: number; // Warning threshold in ms
}

interface ChartDataPoint {
  timestamp: string;
  latency: number;
  success: boolean;
}

export function LatencyTrendChart({
  data,
  title = 'Latency Trend',
  height = 300,
  threshold,
}: LatencyTrendChartProps) {
  // Transform probe results to chart data
  const chartData: ChartDataPoint[] = data.map((result) => ({
    timestamp: result.timestamp,
    latency: result.latency,
    success: result.success,
  }));

  // Format time
  const formatTime = (value: string): string => {
    const date = new Date(value);
    const hours = date.getHours().toString().padStart(2, '0');
    const minutes = date.getMinutes().toString().padStart(2, '0');
    return `${hours}:${minutes}`;
  };

  const formatFullTime = (value: string): string => {
    const date = new Date(value);
    const year = date.getFullYear();
    const month = (date.getMonth() + 1).toString().padStart(2, '0');
    const day = date.getDate().toString().padStart(2, '0');
    const hours = date.getHours().toString().padStart(2, '0');
    const minutes = date.getMinutes().toString().padStart(2, '0');
    return `${year}-${month}-${day} ${hours}:${minutes}`;
  };

  const formatLatency = (value: number): string => {
    return `${value.toFixed(2)}ms`;
  };

  // Calculate statistics
  const stats = chartData.reduce(
    (acc, point) => {
      if (point.success) {
        acc.successCount++;
        acc.totalLatency += point.latency;
        acc.minLatency = Math.min(acc.minLatency, point.latency);
        acc.maxLatency = Math.max(acc.maxLatency, point.latency);
      } else {
        acc.failureCount++;
      }
      return acc;
    },
    {
      successCount: 0,
      failureCount: 0,
      totalLatency: 0,
      minLatency: Infinity,
      maxLatency: 0,
    }
  );

  const avgLatency = stats.successCount > 0 ? stats.totalLatency / stats.successCount : 0;
  const uptimePercentage = chartData.length > 0
    ? (stats.successCount / chartData.length) * 100
    : 0;

  if (chartData.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>{title}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center h-64 text-muted-foreground">
            No probe data available
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <span>{title}</span>
          <div className="flex gap-4 text-sm font-normal">
            <span className="text-muted-foreground">
              Avg: <span className="font-medium text-foreground">{avgLatency.toFixed(2)}ms</span>
            </span>
            <span className="text-muted-foreground">
              Min: <span className="font-medium text-foreground">{stats.minLatency === Infinity ? '-' : stats.minLatency.toFixed(2)}ms</span>
            </span>
            <span className="text-muted-foreground">
              Max: <span className="font-medium text-foreground">{stats.maxLatency.toFixed(2)}ms</span>
            </span>
            <span className="text-muted-foreground">
              Uptime: <span className="font-medium text-foreground">{uptimePercentage.toFixed(1)}%</span>
            </span>
          </div>
        </CardTitle>
      </CardHeader>
      <CardContent>
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
              tickFormatter={formatLatency}
              label={{ value: 'Latency (ms)', angle: -90, position: 'insideLeft' }}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: 'hsl(var(--popover))',
                border: '1px solid hsl(var(--border))',
                borderRadius: '8px',
              }}
              labelFormatter={formatFullTime}
              formatter={(value: number, _name: string, props: any) => {
                const status = props.payload.success ? 'Success' : 'Failed';
                const statusColor = props.payload.success ? 'text-green-600' : 'text-red-600';
                return [
                  <div key="tooltip">
                    <div>{formatLatency(value)}</div>
                    <div className={`text-xs ${statusColor}`}>{status}</div>
                  </div>,
                  'Latency',
                ];
              }}
            />
            {threshold && (
              <ReferenceLine
                y={threshold}
                stroke="orange"
                strokeDasharray="5 5"
                label={{
                  value: `Threshold: ${threshold}ms`,
                  position: 'right',
                  fill: 'orange',
                  fontSize: 12,
                }}
              />
            )}
            <Line
              type="monotone"
              dataKey="latency"
              stroke="hsl(var(--primary))"
              strokeWidth={2}
              dot={(props: any) => {
                const { cx, cy, payload } = props;
                const color = payload.success ? 'hsl(var(--primary))' : 'hsl(var(--destructive))';
                return (
                  <circle
                    cx={cx}
                    cy={cy}
                    r={3}
                    fill={color}
                    stroke="white"
                    strokeWidth={1}
                  />
                );
              }}
              connectNulls
            />
          </LineChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  );
}
