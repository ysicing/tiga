import { useMemo } from 'react'
import { ServiceHistoryInfo } from '@/services/service-monitor'
import { format } from 'date-fns'

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

interface HttpProbeHeatmapProps {
  data: ServiceHistoryInfo[]
  className?: string
  title?: string
  description?: string
}

export function HttpProbeHeatmap({
  data,
  className,
  title = 'HTTP 服务探测延迟',
  description = '使用方块显示 HTTP 服务的探测延迟情况',
}: HttpProbeHeatmapProps) {
  const heatmapData = useMemo(() => {
    if (!data || data.length === 0) {
      return []
    }

    // Process each service
    return data.map((service) => {
      const blocks = service.timestamps.map((ts, index) => {
        const delay = service.avg_delays[index]

        // Determine color based on delay
        let colorClass = ''
        if (delay < 100) {
          colorClass = 'bg-green-500' // Fast: < 100ms
        } else if (delay < 300) {
          colorClass = 'bg-yellow-500' // Normal: 100-300ms
        } else if (delay < 1000) {
          colorClass = 'bg-orange-500' // Slow: 300-1000ms
        } else {
          colorClass = 'bg-red-500' // Very slow: > 1000ms
        }

        return {
          timestamp: ts,
          delay,
          time: format(new Date(ts), 'HH:mm'),
          colorClass,
        }
      })

      // Calculate average delay
      const avgDelay =
        service.avg_delays.length > 0
          ? service.avg_delays.reduce((a, b) => a + b, 0) /
            service.avg_delays.length
          : 0

      return {
        service_monitor_id: service.service_monitor_id,
        service_name: service.service_monitor_name,
        blocks,
        avgDelay,
      }
    })
  }, [data])

  if (!data || data.length === 0) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle>{title}</CardTitle>
          <CardDescription>{description}</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-gray-500">
            暂无 HTTP 探测数据
          </div>
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
        <div className="space-y-6">
          {heatmapData.map((serviceData) => (
            <div key={serviceData.service_monitor_id} className="space-y-2">
              <div className="flex items-center justify-between">
                <h4 className="font-medium text-sm truncate flex-1">
                  {serviceData.service_name}
                </h4>
                <span className="text-xs text-muted-foreground ml-2">
                  平均: {serviceData.avgDelay.toFixed(2)}ms
                </span>
              </div>

              <div className="flex flex-wrap gap-1">
                {serviceData.blocks.map((block, index) => (
                  <div
                    key={index}
                    className={`w-8 h-8 rounded ${block.colorClass} transition-all hover:scale-110 cursor-pointer`}
                    title={`${block.time}: ${block.delay.toFixed(2)}ms`}
                  />
                ))}
              </div>
            </div>
          ))}
        </div>

        {/* Legend */}
        <div className="mt-6 pt-4 border-t">
          <div className="flex items-center gap-6 text-sm text-muted-foreground flex-wrap">
            <div className="flex items-center gap-2">
              <div className="w-4 h-4 rounded bg-green-500" />
              <span>&lt; 100ms (快)</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-4 h-4 rounded bg-yellow-500" />
              <span>100-300ms (正常)</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-4 h-4 rounded bg-orange-500" />
              <span>300-1000ms (慢)</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-4 h-4 rounded bg-red-500" />
              <span>&gt; 1000ms (很慢)</span>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
