import { useCallback, useMemo } from 'react'
import { createColumnHelper } from '@tanstack/react-table'
import { Pod } from 'kubernetes-types/core/v1'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'

import { PodWithMetrics } from '@/types/api'
import { getPodStatus } from '@/lib/k8s'
import { formatDate } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { MetricCell } from '@/components/metrics-cell'
import { PodStatusIcon } from '@/components/pod-status-icon'
import { ResourceTable } from '@/components/resource-table'

export function PodListPage() {
  const { t } = useTranslation()
  // Define column helper outside of any hooks
  const columnHelper = createColumnHelper<PodWithMetrics>()

  // Define columns for the pod table - moved outside render cycle for better performance
  const columns = useMemo(
    () => [
      columnHelper.accessor('metadata.name', {
        header: t('common.name'),
        cell: ({ row }) => (
          <div className="font-medium text-blue-500 hover:underline">
            <Link
              to={`/pods/${row.original.metadata!.namespace}/${
                row.original.metadata!.name
              }`}
            >
              {row.original.metadata!.name}
            </Link>
          </div>
        ),
      }),
      columnHelper.accessor((row) => row.status?.containerStatuses, {
        id: 'containers',
        header: t('pods.ready'),
        cell: ({ row }) => {
          const status = getPodStatus(row.original)
          return (
            <div>
              {status.readyContainers} / {status.totalContainers}
            </div>
          )
        },
      }),
      columnHelper.accessor((row) => row.status?.phase, {
        header: t('common.status'),
        enableColumnFilter: true,
        cell: ({ row }) => {
          const status = getPodStatus(row.original)
          return (
            <Badge variant="outline" className="text-muted-foreground px-1.5">
              <PodStatusIcon status={status.reason} />
              {status.reason}
            </Badge>
          )
        },
      }),
      columnHelper.accessor((row) => row.status, {
        id: 'restarts',
        header: t('pods.restarts'),
        cell: ({ row }) => {
          const status = getPodStatus(row.original)
          return (
            <span className="text-muted-foreground text-sm">
              {status.restartString}
            </span>
          )
        },
      }),
      columnHelper.accessor((row) => row.metrics?.cpuUsage || 0, {
        id: 'cpu',
        header: 'CPU',
        cell: ({ row }) => (
          <MetricCell metrics={row.original.metrics} type="cpu" />
        ),
      }),
      columnHelper.accessor((row) => row.metrics?.memoryUsage || 0, {
        id: 'memory',
        header: 'Memory',
        cell: ({ row }) => (
          <MetricCell metrics={row.original.metrics} type="memory" />
        ),
      }),
      columnHelper.accessor((row) => row.status?.podIP, {
        id: 'podIP',
        header: 'IP',
        cell: ({ getValue }) => {
          const ip = getValue() || '-'
          return (
            <span className="text-muted-foreground text-sm font-mono">
              {ip}
            </span>
          )
        },
      }),
      columnHelper.accessor((row) => row.spec?.nodeName, {
        id: 'nodeName',
        header: t('pods.node'),
        enableColumnFilter: true,
        cell: ({ row }) => {
          if (row.original.spec?.nodeName) {
            return (
              <div className="font-medium text-blue-500 hover:underline">
                <Link to={`/nodes/${row.original.spec?.nodeName}`}>
                  {row.original.spec?.nodeName}
                </Link>
              </div>
            )
          }
          return '-'
        },
      }),
      columnHelper.accessor((row) => row.metadata?.creationTimestamp, {
        id: 'creationTimestamp',
        header: t('common.created'),
        cell: ({ getValue }) => {
          const dateStr = formatDate(getValue() || '')
          return (
            <span className="text-muted-foreground text-sm">{dateStr}</span>
          )
        },
      }),
    ],
    [columnHelper, t]
  )

  // Custom filter for pod search
  const podSearchFilter = useCallback((pod: Pod, query: string) => {
    return (
      pod.metadata?.name?.toLowerCase().includes(query) ||
      (pod.spec?.nodeName?.toLowerCase() || '').includes(query) ||
      (pod.status?.podIP?.toLowerCase() || '').includes(query)
    )
  }, [])

  return (
    <ResourceTable<Pod>
      resourceName="Pods"
      columns={columns}
      clusterScope={false}
      searchQueryFilter={podSearchFilter}
    />
  )
}
