import { clsx, type ClassValue } from 'clsx'
import { format, formatDistance } from 'date-fns'
import { TFunction } from 'i18next'
import { twMerge } from 'tailwind-merge'

import { PodMetrics } from '@/types/api'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

// Simple debounce function for string input handlers with cancel support
export function debounce(fn: (value: string) => void, delay: number) {
  let timeout: NodeJS.Timeout | null = null

  const debouncedFn = function (value: string) {
    if (timeout) {
      clearTimeout(timeout)
    }
    timeout = setTimeout(() => {
      fn(value)
    }, delay)
  }

  debouncedFn.cancel = function () {
    if (timeout) {
      clearTimeout(timeout)
      timeout = null
    }
  }

  return debouncedFn
}

export function getAge(timestamp: string): string {
  const target = new Date(timestamp)
  const now = new Date()
  const diffMs = now.getTime() - target.getTime()
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))
  const diffHours = Math.floor(
    (diffMs % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60)
  )
  const diffMinutes = Math.floor((diffMs % (1000 * 60 * 60)) / (1000 * 60))
  const diffSeconds = Math.floor((diffMs % (1000 * 60)) / 1000)

  if (diffDays > 0) {
    return `${diffDays}d`
  } else if (diffHours > 0) {
    return `${diffHours}h`
  } else if (diffMinutes > 0) {
    return `${diffMinutes}m`
  } else {
    return `${diffSeconds}s`
  }
}

export function formatDate(timestamp: string, addTo = false): string {
  const date = new Date(timestamp)
  const s = format(date, 'yyyy-MM-dd HH:mm:ss')
  return addTo ? `${s} (${formatDistance(new Date(), date)})` : s
}

export function formatChartXTicks(
  timestamp: string,
  isSameDay: boolean
): string {
  const options: Intl.DateTimeFormatOptions = {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }
  if (!isSameDay) {
    options.year = 'numeric'
    options.month = '2-digit'
    options.day = '2-digit'
  }
  return new Date(timestamp).toLocaleString(undefined, options)
}

// Format bytes to human readable format
export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'

  const k = 1024
  const sizes = ['B', 'KiB', 'MiB', 'GiB', 'TiB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))

  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// Format CPU cores
export function formatCPU(cores: string | number): string {
  if (typeof cores === 'string') {
    if (cores.endsWith('m')) {
      const milliCores = parseInt(cores.slice(0, -1))
      return `${(milliCores / 1000).toFixed(3)} cores`
    }
    return `${cores} cores`
  }
  return `${cores} cores`
}

// Format memory
export function formatMemory(memory: string | number): string {
  if (typeof memory === 'number') {
    return formatBytes(memory)
  }

  const units = {
    Ki: 1024,
    Mi: 1024 * 1024,
    Gi: 1024 * 1024 * 1024,
    Ti: 1024 * 1024 * 1024 * 1024,
    K: 1000,
    M: 1000 * 1000,
    G: 1000 * 1000 * 1000,
    T: 1000 * 1000 * 1000 * 1000,
  }

  for (const [suffix, multiplier] of Object.entries(units)) {
    if (memory.endsWith(suffix)) {
      const value = parseFloat(memory.slice(0, -suffix.length))
      return formatBytes(value * multiplier)
    }
  }

  // If no unit, assume bytes
  const numValue = parseFloat(memory)
  if (!isNaN(numValue)) {
    return formatBytes(numValue)
  }

  return memory
}

export function formatPodMetrics(metric: PodMetrics): {
  cpu: number
  memory: number
} {
  let cpu = 0
  let memory = 0
  metric.containers.forEach((container) => {
    const cpuUsage = parseInt(container.usage.cpu, 10) || 0
    if (container.usage.cpu.endsWith('n')) {
      cpu += cpuUsage / 1e9 // nanocores to millicores
    } else if (container.usage.cpu.endsWith('m')) {
      cpu += cpuUsage
    }
    const memoryUsage = parseInt(container.usage.memory, 10) || 0
    if (container.usage.memory.endsWith('Ki')) {
      memory += memoryUsage * 1024
    } else if (container.usage.memory.endsWith('Mi')) {
      memory += memoryUsage * 1024 * 1024
    } else if (container.usage.memory.endsWith('Gi')) {
      memory += memoryUsage * 1024 * 1024 * 1024
    }
  })

  return { cpu, memory }
}
export interface RBACErrorInfo {
  user: string
  verb: string
  resource: string
  namespace?: string
  cluster: string
}

export function parseRBACError(errorMessage: string): RBACErrorInfo | null {
  const namespacePattern =
    /user (.+) does not have permission to (.+) (.+) in namespace (.+) on cluster (.+)/
  const namespaceMatch = errorMessage.match(namespacePattern)

  if (namespaceMatch) {
    return {
      user: namespaceMatch[1],
      verb: namespaceMatch[2],
      resource: namespaceMatch[3],
      namespace: namespaceMatch[4],
      cluster: namespaceMatch[5],
    }
  }

  return null
}

export function isRBACError(errorMessage: string): boolean {
  return !!parseRBACError(errorMessage)
}

export function translateError(error: Error | unknown, t: TFunction): string {
  if (!(error instanceof Error)) {
    return t('common.error', {
      error: String(error),
    })
  }
  const rbacInfo = parseRBACError(error.message)

  if (!rbacInfo) {
    return error.message
  }

  if (rbacInfo.namespace) {
    return t('rbac.noPermissionNamespace', {
      user: rbacInfo.user,
      verb: t(`rbac.verb.${rbacInfo.verb}`, {
        defaultValue: rbacInfo.verb,
      }),
      resource: t(`nav.${rbacInfo.resource}`, {
        defaultValue: rbacInfo.resource,
      }),
      namespace: rbacInfo.namespace === 'All' ? 'All' : rbacInfo.namespace,
      cluster: rbacInfo.cluster,
    })
  }

  return error.message
}
