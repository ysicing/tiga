import { useMemo } from 'react'

import { cn } from '@/lib/utils'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'

interface HeatmapData {
  date: string
  dayIndex: number
  status: 'good' | 'low' | 'down' | 'nodata'
  uptime: number // percentage (0-100)
  avgDelay: number // milliseconds
  up: number // successful count
  down: number // failed count
}

interface AvailabilityHeatmapProps {
  serviceId: string
  serviceName?: string
  data: {
    delay: number[] // 30 elements
    up: number[] // 30 elements
    down: number[] // 30 elements
  }
  className?: string
}

function getStatus(uptime: number): HeatmapData['status'] {
  if (uptime === 0) return 'nodata'
  if (uptime > 95) return 'good'
  if (uptime > 80) return 'low'
  return 'down'
}

function getColorByStatus(status: HeatmapData['status']): string {
  switch (status) {
    case 'good':
      return 'bg-green-500 hover:bg-green-600'
    case 'low':
      return 'bg-orange-500 hover:bg-orange-600'
    case 'down':
      return 'bg-red-500 hover:bg-red-600'
    default:
      return 'bg-gray-300 hover:bg-gray-400'
  }
}

function getStatusLabel(status: HeatmapData['status']): string {
  switch (status) {
    case 'good':
      return '良好'
    case 'low':
      return '可用性偏低'
    case 'down':
      return '故障'
    default:
      return '无数据'
  }
}

export function AvailabilityHeatmap({
  serviceName,
  data,
  className,
}: AvailabilityHeatmapProps) {
  const heatmapData: HeatmapData[] = useMemo(() => {
    if (!data || !data.up || !data.down || !data.delay) {
      return []
    }

    return data.up.map((up, index) => {
      const down = data.down[index] || 0
      const total = up + down
      const uptime = total > 0 ? (up / total) * 100 : 0

      const status = getStatus(uptime)
      const date = new Date()
      // index 0 is today, so we subtract (29 - index) days
      date.setDate(date.getDate() - (29 - index))

      return {
        date: date.toISOString().split('T')[0],
        dayIndex: index,
        status,
        uptime,
        avgDelay: data.delay[index] || 0,
        up,
        down,
      }
    })
  }, [data])

  if (!heatmapData.length) {
    return (
      <div className={cn('text-center py-8 text-gray-500', className)}>
        暂无30天可用性数据
      </div>
    )
  }

  return (
    <div className={cn('space-y-4', className)}>
      {serviceName && (
        <h3 className="text-lg font-semibold">
          {serviceName} - 30天可用性热力图
        </h3>
      )}

      <TooltipProvider>
        <div className="grid grid-cols-30 gap-1">
          {heatmapData.map((item) => (
            <Tooltip key={item.dayIndex}>
              <TooltipTrigger asChild>
                <div
                  className={cn(
                    'w-full aspect-square rounded cursor-pointer transition-all hover:scale-110 hover:shadow-md',
                    getColorByStatus(item.status)
                  )}
                  aria-label={`${item.date}: ${item.uptime.toFixed(2)}% 可用性`}
                />
              </TooltipTrigger>
              <TooltipContent side="top" className="max-w-xs">
                <div className="text-sm space-y-1">
                  <p className="font-medium">{item.date}</p>
                  <p className="text-gray-300">
                    状态:{' '}
                    <span className="font-medium">
                      {getStatusLabel(item.status)}
                    </span>
                  </p>
                  <p className="text-gray-300">
                    可用率:{' '}
                    <span className="font-medium text-white">
                      {item.uptime.toFixed(2)}%
                    </span>
                  </p>
                  {item.avgDelay > 0 && (
                    <p className="text-gray-300">
                      平均延迟:{' '}
                      <span className="font-medium text-white">
                        {item.avgDelay.toFixed(2)}ms
                      </span>
                    </p>
                  )}
                  <p className="text-gray-300">
                    成功:{' '}
                    <span className="font-medium text-green-400">
                      {item.up}
                    </span>{' '}
                    | 失败:{' '}
                    <span className="font-medium text-red-400">
                      {item.down}
                    </span>
                  </p>
                </div>
              </TooltipContent>
            </Tooltip>
          ))}
        </div>
      </TooltipProvider>

      {/* Legend */}
      <div className="flex items-center gap-4 text-sm">
        <span className="text-gray-600">图例:</span>
        <div className="flex items-center gap-2">
          <div className="w-4 h-4 rounded bg-green-500" />
          <span className="text-gray-700">良好 (&gt;95%)</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-4 h-4 rounded bg-orange-500" />
          <span className="text-gray-700">可用性偏低 (80-95%)</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-4 h-4 rounded bg-red-500" />
          <span className="text-gray-700">故障 (&lt;80%)</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-4 h-4 rounded bg-gray-300" />
          <span className="text-gray-700">无数据</span>
        </div>
      </div>

      {/* Time axis labels */}
      <div className="flex justify-between text-xs text-gray-500 mt-2">
        <span>30天前</span>
        <span>今天</span>
      </div>
    </div>
  )
}
