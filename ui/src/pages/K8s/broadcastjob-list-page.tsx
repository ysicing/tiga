import { useCallback, useMemo } from 'react'
import { createColumnHelper } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'

import { getBroadcastJobStatus } from '@/lib/k8s'
import { formatDate } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { ResourceTable } from '@/components/resource-table'
import { BroadcastJob } from '@/types/k8s'

export function BroadcastJobListPage() {
  const { t } = useTranslation()

  const columnHelper = createColumnHelper<BroadcastJob>()

  const columns = useMemo(
    () => [
      columnHelper.accessor('metadata.name', {
        header: t('common.name'),
        cell: ({ row }) => (
          <div className="font-medium text-blue-500 hover:underline">
            <Link
              to={`/k8s/broadcastjobs/${row.original.metadata.namespace}/${
                row.original.metadata.name
              }`}
            >
              {row.original.metadata.name}
            </Link>
          </div>
        ),
      }),
      columnHelper.accessor((row) => row.status, {
        id: 'completions',
        header: t('openkruise.broadcastjobs.completions', 'Completions'),
        cell: ({ row }) => {
          const status = row.original.status
          const succeeded = status?.succeeded || 0
          const desired = status?.desired || 0
          return (
            <div>
              {succeeded} / {desired}
            </div>
          )
        },
      }),
      columnHelper.accessor((row) => row.status, {
        id: 'active',
        header: t('openkruise.broadcastjobs.active', 'Active'),
        cell: ({ row }) => {
          const active = row.original.status?.active || 0
          return <div>{active}</div>
        },
      }),
      columnHelper.accessor('status.phase', {
        header: t('common.status'),
        cell: ({ row }) => {
          const status = getBroadcastJobStatus(row.original)
          const variantMap: Record<string, 'default' | 'destructive' | 'outline' | 'secondary'> = {
            'Running': 'default',
            'Succeeded': 'secondary',
            'Failed': 'destructive',
            'Paused': 'outline',
            'Pending': 'outline',
            'Unknown': 'outline',
          }
          return (
            <Badge variant={variantMap[status] || 'outline'}>
              {status}
            </Badge>
          )
        },
      }),
      columnHelper.accessor('metadata.creationTimestamp', {
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

  const broadcastJobSearchFilter = useCallback(
    (broadcastJob: BroadcastJob, query: string) => {
      return (
        broadcastJob.metadata.name.toLowerCase().includes(query) ||
        (broadcastJob.metadata.namespace?.toLowerCase() || '').includes(query)
      )
    },
    []
  )

  return (
    <ResourceTable
      resourceName={t('openkruise.broadcastjobs.title', 'BroadcastJobs')}
      resourceType="broadcastjobs"
      columns={columns}
      searchQueryFilter={broadcastJobSearchFilter}
    />
  )
}
